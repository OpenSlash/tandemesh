// Package github implements the GitHub task-source integration.
//
// It is the first adapter built on the uniform integration.Integration value:
// one manifest, one webhook slot, one work-writer slot.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/ingress"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Adapter is the GitHub implementation of the webhook and work-writer slots.
type Adapter struct {
	Secret     string
	HTTPClient *http.Client
	BaseURL    string // defaults to https://api.github.com
}

// New builds the GitHub integration.
func New(secret string) *integration.Integration {
	base := os.Getenv("ORCA_GITHUB_API_URL")
	if base == "" {
		base = "https://api.github.com"
	}
	a := &Adapter{Secret: secret, HTTPClient: &http.Client{Timeout: 20 * time.Second}, BaseURL: base}
	return &integration.Integration{
		Manifest: integration.Manifest{
			ID:   "github",
			Name: "GitHub",
			Kind: integration.KindTask,
			Auth: integration.AuthToken,
			AuthOptions: []integration.AuthOption{
				{Kind: integration.AuthApp, LabelKey: "providers.authApp", Default: true},
				{Kind: integration.AuthOAuth2, LabelKey: "providers.authOAuth"},
				{Kind: integration.AuthToken, LabelKey: "providers.authToken"},
			},
			Capabilities: []integration.Capability{integration.CapWebhook, integration.CapComment, integration.CapAttach},
			ConfigFields: []integration.ConfigField{
				{Key: "token", LabelKey: "providers.token", Type: "password", Required: true, Secret: true},
				{Key: "instance", LabelKey: "providers.repo", Type: "text", Required: true, Placeholder: "owner/repo"},
				{Key: "baseUrl", LabelKey: "providers.baseUrl", Type: "url", Placeholder: "https://api.github.com"},
				{Key: "webhookSecret", LabelKey: "providers.webhookSecret", Type: "password", Secret: true},
			},
		},
		Webhook:   a,
		Registrar: a,
		Probe:     a,
		Reader:    a,
		Work:      a,
	}
}

// Verify implements integration.Webhook.
func (a *Adapter) Verify(r *http.Request, body []byte) error {
	return ingress.VerifyGitHub(a.Secret, body, r.Header.Get("X-Hub-Signature-256"))
}

// Parse implements integration.Webhook.
func (a *Adapter) Parse(body []byte) (integration.Event, error) {
	p, err := ingress.ParseGitHub(body)
	if err != nil {
		return integration.Event{}, err
	}
	labels := make([]string, 0, len(p.Issue.Labels))
	for _, l := range p.Issue.Labels {
		labels = append(labels, l.Name)
	}
	kind := p.Action
	if p.Comment.Body != "" {
		kind = "comment"
	}
	return integration.Event{
		Delivery: p.Issue.HTMLURL,
		Kind:     kind,
		Actor:    p.Sender.Login,
		Labels:   labels,
		Assignee: p.Assignee.Login,
		Text:     p.Comment.Body,
		Raw:      body,
	}, nil
}

// Normalize implements integration.Webhook.
func (a *Adapter) Normalize(ev integration.Event) (domain.WorkRef, domain.Snapshot, string, string, error) {
	p, err := ingress.ParseGitHub(ev.Raw)
	if err != nil {
		return domain.WorkRef{}, domain.Snapshot{}, "", "", err
	}
	work, snap, prompt, skip := ingress.NormalizeGitHub(p)
	return work, snap, prompt, skip, nil
}

// Comment implements integration.WorkWriter.
func (a *Adapter) Comment(ctx context.Context, work domain.WorkRef, cred integration.Credential, markdown string) (string, error) {
	if cred.AccessToken == "" {
		return "", fmt.Errorf("github: access token is required to comment")
	}
	body, _ := json.Marshal(map[string]string{"body": markdown})
	url := fmt.Sprintf("%s/repos/%s/issues/%s/comments", a.baseURLFor(cred), work.Instance, work.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("github comment: status %d", resp.StatusCode)
	}
	var out struct {
		HTMLURL string `json:"html_url"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return out.HTMLURL, nil
}

// Attach implements integration.WorkWriter by posting a link as a comment.
func (a *Adapter) Attach(ctx context.Context, work domain.WorkRef, cred integration.Credential, kind, url string) error {
	_, err := a.Comment(ctx, work, cred, fmt.Sprintf("%s: %s", strings.Title(kind), url))
	return err
}

// Transition implements integration.WorkWriter. GitHub only exposes open and
// closed, so unsupported targets return integration.ErrUnsupported.
func (a *Adapter) Transition(ctx context.Context, work domain.WorkRef, cred integration.Credential, to string) error {
	state := ""
	switch to {
	case "closed", "done", "completed":
		state = "closed"
	case "open", "todo", "in_progress":
		state = "open"
	default:
		return integration.ErrUnsupported
	}
	body, _ := json.Marshal(map[string]string{"state": state})
	url := fmt.Sprintf("%s/repos/%s/issues/%s", a.baseURLFor(cred), work.Instance, work.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("github transition: status %d", resp.StatusCode)
	}
	return nil
}

func (a *Adapter) baseURLFor(cred integration.Credential) string {
	if cred.BaseURL != "" {
		return strings.TrimRight(cred.BaseURL, "/")
	}
	return a.baseURL()
}

func (a *Adapter) baseURL() string {
	if a.BaseURL != "" {
		return strings.TrimRight(a.BaseURL, "/")
	}
	return "https://api.github.com"
}

func (a *Adapter) client() *http.Client {
	if a.HTTPClient != nil {
		return a.HTTPClient
	}
	return http.DefaultClient
}

// Assign implements integration.AssigneeWriter.
func (a *Adapter) Assign(ctx context.Context, work domain.WorkRef, cred integration.Credential, assignee string) error {
	if cred.AccessToken == "" {
		return fmt.Errorf("github: access token is required to assign")
	}
	body, _ := json.Marshal(map[string][]string{"assignees": {assignee}})
	url := fmt.Sprintf("%s/repos/%s/issues/%s", a.baseURLFor(cred), work.Instance, work.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("github assign: status %d", resp.StatusCode)
	}
	return nil
}

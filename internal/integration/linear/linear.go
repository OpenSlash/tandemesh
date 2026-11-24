// Package linear implements the Linear task-source integration.
package linear

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Adapter is the Linear implementation of the webhook and work-writer slots.
type Adapter struct {
	Secret     string
	HTTPClient *http.Client
	BaseURL    string // defaults to https://api.linear.app/graphql
}

// New builds the Linear integration.
func New(secret string) *integration.Integration {
	a := &Adapter{Secret: secret, HTTPClient: &http.Client{Timeout: 20 * time.Second}, BaseURL: "https://api.linear.app/graphql"}
	return &integration.Integration{
		Manifest: integration.Manifest{
			ID:   "linear",
			Name: "Linear",
			Kind: integration.KindTask,
			Auth: integration.AuthToken,
			AuthOptions: []integration.AuthOption{
				{Kind: integration.AuthOAuth2, LabelKey: "providers.authOAuth", Default: true, Fields: []integration.ConfigField{
					{Key: "instance", LabelKey: "providers.teamKey", Type: "text", Required: true, Placeholder: "ENG"},
				}},
				{Kind: integration.AuthToken, LabelKey: "providers.authToken", Fields: []integration.ConfigField{
					{Key: "token", LabelKey: "providers.token", Type: "password", Required: true, Secret: true},
					{Key: "instance", LabelKey: "providers.teamKey", Type: "text", Required: true, Placeholder: "ENG"},
					{Key: "baseUrl", LabelKey: "providers.baseUrl", Type: "url", Placeholder: "https://api.linear.app/graphql"},
				}},
			},
			Capabilities: []integration.Capability{integration.CapWebhook, integration.CapComment},
			ConfigFields: []integration.ConfigField{
				{Key: "token", LabelKey: "providers.token", Type: "password", Required: true, Secret: true},
				{Key: "instance", LabelKey: "providers.teamKey", Type: "text", Required: true, Placeholder: "ENG"},
				{Key: "baseUrl", LabelKey: "providers.baseUrl", Type: "url", Placeholder: "https://api.linear.app/graphql"},
			},
		},
		Webhook:   a,
		Registrar: a,
		Probe:     a,
		Reader:    a,
		Work:      a,
	}
}

// Verify implements integration.Webhook. Linear signs the raw body with
// HMAC-SHA256 and sends the hex digest in Linear-Signature.
func (a *Adapter) Verify(r *http.Request, body []byte) error {
	if a.Secret == "" {
		return nil // development mode
	}
	sig := r.Header.Get("Linear-Signature")
	if sig == "" {
		return errors.New("linear: missing signature")
	}
	want, err := hex.DecodeString(strings.TrimSpace(sig))
	if err != nil {
		return errors.New("linear: malformed signature")
	}
	mac := hmac.New(sha256.New, []byte(a.Secret))
	mac.Write(body)
	if !hmac.Equal(want, mac.Sum(nil)) {
		return errors.New("linear: signature mismatch")
	}
	return nil
}

type payload struct {
	Action         string `json:"action"`
	Type           string `json:"type"`
	OrganizationID string `json:"organizationId"`
	Data           struct {
		ID         string `json:"id"`
		Identifier string `json:"identifier"`
		Title      string `json:"title"`
		URL        string `json:"url"`
		UpdatedAt  string `json:"updatedAt"`
		Labels     []struct {
			Name string `json:"name"`
		} `json:"labels"`
		Assignee *struct {
			Name string `json:"name"`
		} `json:"assignee"`
		Team struct {
			Key string `json:"key"`
		} `json:"team"`
		Body  string `json:"body"`
		Issue *struct {
			ID         string `json:"id"`
			Identifier string `json:"identifier"`
			Title      string `json:"title"`
			URL        string `json:"url"`
			UpdatedAt  string `json:"updatedAt"`
			Team       struct {
				Key string `json:"key"`
			} `json:"team"`
		} `json:"issue"`
	} `json:"data"`
	Actor struct {
		Name string `json:"name"`
	} `json:"actor"`
}

// Parse implements integration.Webhook.
func (a *Adapter) Parse(body []byte) (integration.Event, error) {
	var p payload
	if err := json.Unmarshal(body, &p); err != nil {
		return integration.Event{}, fmt.Errorf("linear: decode payload: %w", err)
	}
	if p.Type != "" && p.Type != "Issue" && p.Type != "Comment" {
		return integration.Event{}, fmt.Errorf("linear: unsupported type %q", p.Type)
	}
	issueID := p.Data.ID
	if p.Type == "Comment" && p.Data.Issue != nil {
		issueID = p.Data.Issue.ID
	}
	if issueID == "" {
		return integration.Event{}, errors.New("linear: missing issue id")
	}
	labels := make([]string, 0, len(p.Data.Labels))
	for _, l := range p.Data.Labels {
		labels = append(labels, l.Name)
	}
	assignee := ""
	if p.Data.Assignee != nil {
		assignee = p.Data.Assignee.Name
	}
	kind := p.Action
	text := ""
	if p.Type == "Comment" {
		kind = "comment"
		text = p.Data.Body
	}
	return integration.Event{
		Delivery: issueID + ":" + p.Data.UpdatedAt,
		Kind:     kind,
		Actor:    p.Actor.Name,
		Labels:   labels,
		Assignee: assignee,
		Text:     text,
		Raw:      body,
	}, nil
}

// Normalize implements integration.Webhook.
func (a *Adapter) Normalize(ev integration.Event) (domain.WorkRef, domain.Snapshot, string, string, error) {
	var p payload
	if err := json.Unmarshal(ev.Raw, &p); err != nil {
		return domain.WorkRef{}, domain.Snapshot{}, "", "", err
	}
	if p.Type == "Comment" && p.Data.Issue != nil {
		issue := p.Data.Issue
		instance := p.OrganizationID
		if instance == "" {
			instance = issue.Team.Key
		}
		work := domain.WorkRef{Provider: "linear", Instance: instance, Type: "issue", ID: issue.ID, URL: issue.URL, Revision: issue.UpdatedAt}
		snap := domain.Snapshot{Title: issue.Title, BodyHash: hash(issue.Title), Labels: ev.Labels, Author: ev.Actor}
		return work, snap, "评论：" + ev.Text, "", nil
	}
	instance := p.OrganizationID
	if instance == "" {
		instance = p.Data.Team.Key
	}
	work := domain.WorkRef{
		Provider: "linear",
		Instance: instance,
		Type:     "issue",
		ID:       p.Data.ID,
		URL:      p.Data.URL,
		Revision: p.Data.UpdatedAt,
	}
	snap := domain.Snapshot{
		Title:    p.Data.Title,
		BodyHash: hash(p.Data.Title),
		Labels:   ev.Labels,
		Author:   ev.Actor,
	}
	if ev.Kind != "create" && ev.Kind != "update" {
		return work, snap, "", "SKIP_EVENT_ACTION", nil
	}
	prompt := "Fix the following Linear issue.\n\nTitle: " + p.Data.Title + "\nURL: " + p.Data.URL
	return work, snap, prompt, "", nil
}

// Comment implements integration.WorkWriter via the Linear GraphQL API.
func (a *Adapter) Comment(ctx context.Context, work domain.WorkRef, cred integration.Credential, markdown string) (string, error) {
	if cred.AccessToken == "" {
		return "", errors.New("linear: access token is required to comment")
	}
	query := `mutation Comment($issueId: String!, $body: String!) {
		commentCreate(input: {issueId: $issueId, body: $body}) {
			success
			comment { url }
		}
	}`
	body, _ := json.Marshal(map[string]any{
		"query":     query,
		"variables": map[string]string{"issueId": work.ID, "body": markdown},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL(), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", cred.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("linear comment: status %d", resp.StatusCode)
	}
	var out struct {
		Data struct {
			CommentCreate struct {
				Success bool `json:"success"`
				Comment struct {
					URL string `json:"url"`
				} `json:"comment"`
			} `json:"commentCreate"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Errors) > 0 {
		return "", fmt.Errorf("linear comment: %s", out.Errors[0].Message)
	}
	if !out.Data.CommentCreate.Success {
		return "", errors.New("linear comment: not created")
	}
	return out.Data.CommentCreate.Comment.URL, nil
}

// Transition implements integration.WorkWriter. Linear needs a workflow state
// id, which the adapter cannot resolve without a state lookup; report unsupported
// until the lookup lands.
func (a *Adapter) Transition(context.Context, domain.WorkRef, integration.Credential, string) error {
	return integration.ErrUnsupported
}

// Attach implements integration.WorkWriter by posting a link as a comment.
func (a *Adapter) Attach(ctx context.Context, work domain.WorkRef, cred integration.Credential, kind, url string) error {
	_, err := a.Comment(ctx, work, cred, fmt.Sprintf("%s: %s", kind, url))
	return err
}

func (a *Adapter) baseURL() string {
	if a.BaseURL != "" {
		return a.BaseURL
	}
	return "https://api.linear.app/graphql"
}

func (a *Adapter) client() *http.Client {
	if a.HTTPClient != nil {
		return a.HTTPClient
	}
	return http.DefaultClient
}

func hash(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

// Assign implements integration.AssigneeWriter.
func (a *Adapter) Assign(ctx context.Context, work domain.WorkRef, cred integration.Credential, assignee string) error {
	if cred.AccessToken == "" {
		return errors.New("linear: access token is required to assign")
	}
	query := `mutation Assign($id: String!, $assigneeId: String!) {
		issueUpdate(id: $id, input: {assigneeId: $assigneeId}) { success }
	}`
	body, _ := json.Marshal(map[string]any{
		"query":     query,
		"variables": map[string]string{"id": work.ID, "assigneeId": assignee},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", cred.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("linear assign: status %d", resp.StatusCode)
	}
	var out struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return err
	}
	if len(out.Errors) > 0 {
		return fmt.Errorf("linear assign: %s", out.Errors[0].Message)
	}
	return nil
}

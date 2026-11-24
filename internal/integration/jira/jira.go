// Package jira implements the Jira Cloud task-source integration.
package jira

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Adapter implements the Jira webhook, work-writer slots.
type Adapter struct {
	Secret     string
	HTTPClient *http.Client
}

// New builds the Jira integration.
func New(secret string) *integration.Integration {
	a := &Adapter{Secret: secret, HTTPClient: &http.Client{Timeout: 20 * time.Second}}
	return &integration.Integration{
		Manifest: integration.Manifest{
			ID:   "jira",
			Name: "Jira",
			Kind: integration.KindTask,
			Auth: integration.AuthToken,
			AuthOptions: []integration.AuthOption{
				{Kind: integration.AuthOAuth2, LabelKey: "providers.authOAuth", Default: true, Fields: []integration.ConfigField{
					{Key: "instance", LabelKey: "providers.jiraSite", Type: "url", Required: true, Placeholder: "https://company.atlassian.net"},
				}},
				{Kind: integration.AuthToken, LabelKey: "providers.authToken", Fields: []integration.ConfigField{
					{Key: "email", LabelKey: "providers.email", Type: "text", Required: true, Placeholder: "you@company.com"},
					{Key: "token", LabelKey: "providers.token", Type: "password", Required: true, Secret: true},
					{Key: "instance", LabelKey: "providers.jiraSite", Type: "url", Required: true, Placeholder: "https://company.atlassian.net"},
					{Key: "webhookSecret", LabelKey: "providers.webhookSecret", Type: "password", Secret: true},
				}},
			},
			Capabilities: []integration.Capability{integration.CapWebhook, integration.CapComment, integration.CapTransition},
			ConfigFields: []integration.ConfigField{
				{Key: "email", LabelKey: "providers.email", Type: "text", Required: true, Placeholder: "you@company.com"},
				{Key: "token", LabelKey: "providers.token", Type: "password", Required: true, Secret: true},
				{Key: "instance", LabelKey: "providers.jiraSite", Type: "url", Required: true, Placeholder: "https://company.atlassian.net"},
				{Key: "webhookSecret", LabelKey: "providers.webhookSecret", Type: "password", Secret: true},
			},
		},
		Webhook:   a,
		Registrar: a,
		Work:      a,
		Reader:    a,
		States:    a,
		Probe:     a,
	}
}

// Verify implements integration.Webhook. Jira sends X-Hub-Signature with the
// sha256= prefix when a secret is configured.
func (a *Adapter) Verify(r *http.Request, body []byte) error {
	if a.Secret == "" {
		return nil
	}
	sig := r.Header.Get("X-Hub-Signature")
	const prefix = "sha256="
	if !strings.HasPrefix(sig, prefix) {
		return errors.New("jira: missing signature")
	}
	want, err := hex.DecodeString(strings.TrimPrefix(sig, prefix))
	if err != nil {
		return errors.New("jira: malformed signature")
	}
	mac := hmac.New(sha256.New, []byte(a.Secret))
	mac.Write(body)
	if !hmac.Equal(want, mac.Sum(nil)) {
		return errors.New("jira: signature mismatch")
	}
	return nil
}

// Parse implements integration.Webhook.
func (a *Adapter) Parse(body []byte) (integration.Event, error) {
	var p struct {
		WebhookEvent string `json:"webhookEvent"`
		Issue        struct {
			ID     string `json:"id"`
			Key    string `json:"key"`
			Self   string `json:"self"`
			Fields struct {
				Summary  string   `json:"summary"`
				Labels   []string `json:"labels"`
				Priority *struct {
					Name string `json:"name"`
				} `json:"priority"`
				Components []struct {
					Name string `json:"name"`
				} `json:"components"`
				FixVersions []struct {
					Name string `json:"name"`
				} `json:"fixVersions"`
				Attachment []struct {
					ID       string `json:"id"`
					Filename string `json:"filename"`
					Content  string `json:"content"`
					MimeType string `json:"mimeType"`
					Size     int64  `json:"size"`
					Author   struct {
						DisplayName string `json:"displayName"`
					} `json:"author"`
				} `json:"attachment"`
				Updated  string `json:"updated"`
				Assignee *struct {
					DisplayName string `json:"displayName"`
				} `json:"assignee"`
			} `json:"fields"`
		} `json:"issue"`
		User struct {
			DisplayName string `json:"displayName"`
		} `json:"user"`
		Comment struct {
			Body   string `json:"body"`
			Author struct {
				DisplayName string `json:"displayName"`
			} `json:"author"`
		} `json:"comment"`
	}
	if err := json.Unmarshal(body, &p); err != nil {
		return integration.Event{}, fmt.Errorf("jira: decode payload: %w", err)
	}
	if p.Issue.Key == "" {
		return integration.Event{}, errors.New("jira: missing issue key")
	}
	assignee := ""
	if p.Issue.Fields.Assignee != nil {
		assignee = p.Issue.Fields.Assignee.DisplayName
	}
	priority := ""
	if p.Issue.Fields.Priority != nil {
		priority = p.Issue.Fields.Priority.Name
	}
	components := make([]string, 0, len(p.Issue.Fields.Components))
	for _, component := range p.Issue.Fields.Components {
		components = append(components, component.Name)
	}
	versions := make([]string, 0, len(p.Issue.Fields.FixVersions))
	for _, version := range p.Issue.Fields.FixVersions {
		versions = append(versions, version.Name)
	}
	attachments := make([]integration.Attachment, 0, len(p.Issue.Fields.Attachment))
	for _, attachment := range p.Issue.Fields.Attachment {
		attachments = append(attachments, integration.Attachment{
			ID: attachment.ID, Name: attachment.Filename, URL: attachment.Content,
			Mime: attachment.MimeType, Size: attachment.Size, Author: attachment.Author.DisplayName,
		})
	}
	kind := p.WebhookEvent
	text := ""
	if strings.HasPrefix(kind, "comment_") {
		kind = "comment"
		text = p.Comment.Body
	}
	return integration.Event{
		Delivery:    p.Issue.Key + ":" + p.Issue.Fields.Updated,
		Kind:        kind,
		Actor:       p.User.DisplayName,
		Labels:      projectFields(p.Issue.Fields.Labels, priority, components, versions),
		Assignee:    assignee,
		Attachments: attachments,
		Text:        text,
		Raw:         body,
	}, nil
}

// Normalize implements integration.Webhook.
func (a *Adapter) Normalize(ev integration.Event) (domain.WorkRef, domain.Snapshot, string, string, error) {
	var p struct {
		Issue struct {
			ID     string `json:"id"`
			Key    string `json:"key"`
			Self   string `json:"self"`
			Fields struct {
				Summary string   `json:"summary"`
				Labels  []string `json:"labels"`
				Updated string   `json:"updated"`
			} `json:"fields"`
		} `json:"issue"`
		User struct {
			DisplayName string `json:"displayName"`
		} `json:"user"`
		Comment struct {
			Body   string `json:"body"`
			Author struct {
				DisplayName string `json:"displayName"`
			} `json:"author"`
		} `json:"comment"`
	}
	if err := json.Unmarshal(ev.Raw, &p); err != nil {
		return domain.WorkRef{}, domain.Snapshot{}, "", "", err
	}
	instance := instanceOf(p.Issue.Self)
	work := domain.WorkRef{
		Provider: "jira",
		Instance: instance,
		Type:     "issue",
		ID:       p.Issue.Key,
		URL:      strings.TrimRight(instance, "/") + "/browse/" + p.Issue.Key,
		Revision: p.Issue.Fields.Updated,
	}
	snap := domain.Snapshot{
		Title:    p.Issue.Fields.Summary,
		BodyHash: hash(p.Issue.Fields.Summary),
		Labels:   ev.Labels,
		Author:   ev.Actor,
	}
	switch ev.Kind {
	case "jira:issue_created", "jira:issue_updated", "comment":
	default:
		return work, snap, "", "SKIP_EVENT_ACTION", nil
	}
	prompt := "Fix the following Jira issue.\n\nKey: " + p.Issue.Key + "\nTitle: " + p.Issue.Fields.Summary + "\nURL: " + work.URL
	if ev.Kind == "comment" {
		prompt = "评论：" + ev.Text
	}
	return work, snap, prompt, "", nil
}

// Comment implements integration.WorkWriter (Jira REST v2).
func (a *Adapter) Comment(ctx context.Context, work domain.WorkRef, cred integration.Credential, markdown string) (string, error) {
	body, _ := json.Marshal(map[string]string{"body": markdown})
	endpoint := strings.TrimRight(work.Instance, "/") + "/rest/api/2/issue/" + url.PathEscape(work.ID) + "/comment"
	resp, err := a.do(ctx, http.MethodPost, endpoint, cred, body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("jira comment: status %d", resp.StatusCode)
	}
	var out struct {
		Self string `json:"self"`
		ID   string `json:"id"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return out.Self, nil
}

// Attach implements integration.WorkWriter by posting a link as a comment.
func (a *Adapter) Attach(ctx context.Context, work domain.WorkRef, cred integration.Credential, kind, link string) error {
	_, err := a.Comment(ctx, work, cred, fmt.Sprintf("%s: %s", kind, link))
	return err
}

// Transition implements integration.WorkWriter by resolving the transition name.
func (a *Adapter) Transition(ctx context.Context, work domain.WorkRef, cred integration.Credential, to string) error {
	base := strings.TrimRight(work.Instance, "/") + "/rest/api/2/issue/" + url.PathEscape(work.ID)
	resp, err := a.do(ctx, http.MethodGet, base+"/transitions", cred, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var list struct {
		Transitions []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			To   struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"to"`
		} `json:"transitions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return err
	}
	target := ""
	for _, t := range list.Transitions {
		if t.To.ID == to || strings.EqualFold(t.Name, to) || strings.EqualFold(t.To.Name, to) {
			target = t.ID
			break
		}
	}
	if target == "" {
		return integration.ErrUnsupported
	}
	body, _ := json.Marshal(map[string]any{"transition": map[string]string{"id": target}})
	post, err := a.do(ctx, http.MethodPost, base+"/transitions", cred, body)
	if err != nil {
		return err
	}
	defer post.Body.Close()
	if post.StatusCode >= 300 {
		return fmt.Errorf("jira transition: status %d", post.StatusCode)
	}
	return nil
}

func (a *Adapter) do(ctx context.Context, method, endpoint string, cred integration.Credential, body []byte) (*http.Response, error) {
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, err
	}
	account := cred.Account
	if account == "" {
		account = cred.Subject
	}
	if cred.Fields["auth"] != "oauth" && account != "" && strings.Contains(account, "@") {
		raw := base64.StdEncoding.EncodeToString([]byte(account + ":" + cred.AccessToken))
		req.Header.Set("Authorization", "Basic "+raw)
	} else {
		req.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	return a.client().Do(req)
}

func (a *Adapter) client() *http.Client {
	if a.HTTPClient != nil {
		return a.HTTPClient
	}
	return http.DefaultClient
}

func instanceOf(self string) string {
	u, err := url.Parse(self)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return self
	}
	return u.Scheme + "://" + u.Host
}

func hash(v string) string {
	sum := sha256.Sum256([]byte(v))
	return hex.EncodeToString(sum[:])
}

// Assign implements integration.AssigneeWriter.
func (a *Adapter) Assign(ctx context.Context, work domain.WorkRef, cred integration.Credential, assignee string) error {
	base := strings.TrimRight(work.Instance, "/") + "/rest/api/2/issue/" + url.PathEscape(work.ID) + "/assignee"
	body, _ := json.Marshal(map[string]string{"accountId": assignee})
	resp, err := a.do(ctx, http.MethodPut, base, cred, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("jira assign: status %d", resp.StatusCode)
	}
	return nil
}

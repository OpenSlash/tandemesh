package plane

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

type webhookPayload struct {
	Event       string          `json:"event"`
	Action      string          `json:"action"`
	WebhookID   string          `json:"webhook_id"`
	WorkspaceID string          `json:"workspace_id"`
	Data        json.RawMessage `json:"data"`
	Activity    struct {
		Actor struct {
			DisplayName string `json:"display_name"`
		} `json:"actor"`
	} `json:"activity"`
}

type webhookIssue struct {
	ID              string          `json:"id"`
	SequenceID      int             `json:"sequence_id"`
	Name            string          `json:"name"`
	Description     string          `json:"description_stripped"`
	DescriptionHTML string          `json:"description_html"`
	Project         string          `json:"project"`
	Workspace       string          `json:"workspace"`
	WorkspaceSlug   string          `json:"workspace_slug"`
	Permalink       string          `json:"permalink"`
	UpdatedAt       string          `json:"updated_at"`
	Labels          []any           `json:"labels"`
	LabelDetails    []label         `json:"label_details"`
	AssigneeDetails []assignee      `json:"assignee_details"`
	State           json.RawMessage `json:"state"`
}

func (a *Adapter) credentialSecret() string {
	return a.Secret
}

// Verify implements integration.Webhook using Plane's HMAC-SHA256 signature.
func (a *Adapter) Verify(r *http.Request, body []byte) error {
	return a.VerifyWithSecret(r, body, a.Secret)
}

// VerifyWithSecret checks the HMAC signature with a per-connection secret.
func (a *Adapter) VerifyWithSecret(r *http.Request, body []byte, secret string) error {
	if secret == "" {
		return fmt.Errorf("plane: webhook secret is not configured")
	}
	sig := strings.TrimSpace(r.Header.Get("X-Plane-Signature"))
	if sig == "" {
		return fmt.Errorf("plane: missing signature")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(sig)) {
		return fmt.Errorf("plane: signature mismatch")
	}
	return nil
}

// Parse implements integration.Webhook.
func (a *Adapter) Parse(body []byte) (integration.Event, error) {
	var p webhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return integration.Event{}, err
	}
	issue, err := issueFromWebhook(p.Data)
	if err != nil {
		return integration.Event{}, err
	}
	labels := webhookLabelNames(issue)
	kind := p.Action
	if kind == "" {
		kind = p.Event
	}
	delivery := p.WebhookID + ":" + p.Action + ":" + issue.UpdatedAt
	if p.WebhookID == "" {
		delivery = issue.ID + ":" + p.Action + ":" + issue.UpdatedAt
	}
	ev := integration.Event{
		Delivery: delivery,
		Kind:     kind,
		Actor:    p.Activity.Actor.DisplayName,
		Labels:   labels,
		Assignee: webhookAssigneeName(issue),
		Text:     issue.Description,
		Raw:      body,
	}
	if kind == "comment" {
		var direct struct {
			CommentStripped string `json:"comment_stripped"`
			CommentHTML     string `json:"comment_html"`
		}
		var nested struct {
			Comment struct {
				CommentStripped string `json:"comment_stripped"`
				CommentHTML     string `json:"comment_html"`
			} `json:"comment"`
		}
		text := ""
		if json.Unmarshal(p.Data, &direct) == nil {
			text = direct.CommentStripped
			if text == "" {
				text = stripHTML(direct.CommentHTML)
			}
		}
		if text == "" && json.Unmarshal(p.Data, &nested) == nil {
			text = nested.Comment.CommentStripped
			if text == "" {
				text = stripHTML(nested.Comment.CommentHTML)
			}
		}
		if text != "" {
			ev.Text = text
		}
	}
	return ev, nil
}

// Normalize implements integration.Webhook.
func (a *Adapter) Normalize(ev integration.Event) (domain.WorkRef, domain.Snapshot, string, string, error) {
	var p webhookPayload
	if err := json.Unmarshal(ev.Raw, &p); err != nil {
		return domain.WorkRef{}, domain.Snapshot{}, "", "", err
	}
	if p.Action == "deleted" {
		return domain.WorkRef{}, domain.Snapshot{}, "", "SKIP_DELETED", nil
	}
	issue, err := issueFromWebhook(p.Data)
	if err != nil {
		return domain.WorkRef{}, domain.Snapshot{}, "", "", err
	}
	slug := issue.WorkspaceSlug
	if slug == "" {
		slug = issue.Workspace
	}
	instance := ""
	if slug != "" {
		instance = slug + "/" + issue.Project
	}
	url := issue.Permalink
	if url == "" && a.AppURL != "" && slug != "" {
		url = fmt.Sprintf("%s/%s/projects/%s/issues/%s", a.AppURL, slug, issue.Project, issue.ID)
	}
	body := issue.Description
	if body == "" {
		body = stripHTML(issue.DescriptionHTML)
	}
	work := domain.WorkRef{
		Provider: "plane", Instance: instance, Type: "issue",
		ID: issue.ID, URL: url, Revision: issue.UpdatedAt,
	}
	snap := domain.Snapshot{
		Title: issue.Name, Body: body, Labels: webhookLabelNames(issue), Author: webhookAssigneeName(issue),
	}
	prompt := issue.Name
	if body != "" {
		prompt += "\n\n" + body
	}
	return work, snap, prompt, "", nil
}

func issueFromWebhook(raw json.RawMessage) (webhookIssue, error) {
	var direct webhookIssue
	if err := json.Unmarshal(raw, &direct); err == nil && direct.ID != "" {
		return direct, nil
	}
	var nested struct {
		Issue   *webhookIssue `json:"issue"`
		Comment struct {
			Issue string `json:"issue"`
		} `json:"comment"`
	}
	if err := json.Unmarshal(raw, &nested); err != nil {
		return webhookIssue{}, fmt.Errorf("plane: decode issue: %w", err)
	}
	if nested.Issue != nil {
		return *nested.Issue, nil
	}
	return webhookIssue{}, fmt.Errorf("plane: webhook has no issue")
}

func stripHTML(s string) string {
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<p>", "")
	s = strings.ReplaceAll(s, "</p>", "\n")
	for {
		start := strings.Index(s, "<")
		if start < 0 {
			break
		}
		end := strings.Index(s[start:], ">")
		if end < 0 {
			break
		}
		s = s[:start] + s[start+end+1:]
	}
	return html.UnescapeString(strings.TrimSpace(s))
}

func (a *Adapter) webhookDelivery(r *http.Request) string {
	return r.Header.Get("X-Plane-Delivery") + ":" + time.Now().UTC().Format(time.RFC3339Nano)
}

func webhookLabelNames(item webhookIssue) []string {
	out := []string{}
	for _, l := range item.LabelDetails {
		if l.Name != "" {
			out = append(out, l.Name)
		}
	}
	if len(out) == 0 {
		for _, raw := range item.Labels {
			switch v := raw.(type) {
			case string:
				out = append(out, v)
			case map[string]any:
				if name, ok := v["name"].(string); ok {
					out = append(out, name)
				}
			}
		}
	}
	return out
}

func webhookAssigneeName(item webhookIssue) string {
	for _, a := range item.AssigneeDetails {
		if a.DisplayName != "" {
			return a.DisplayName
		}
		if a.Email != "" {
			return a.Email
		}
	}
	return ""
}

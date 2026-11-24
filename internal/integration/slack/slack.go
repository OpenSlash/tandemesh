// Package slack implements the Slack chat integration.
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Adapter is the Slack implementation of the chat slot.
type Adapter struct {
	HTTPClient *http.Client
	BaseURL    string // defaults to https://slack.com/api
}

// New builds the Slack integration.
func New() *integration.Integration {
	a := &Adapter{HTTPClient: &http.Client{Timeout: 20 * time.Second}, BaseURL: "https://slack.com/api"}
	return &integration.Integration{
		Manifest: integration.Manifest{
			ID:   "slack",
			Name: "Slack",
			Kind: integration.KindChat,
			Auth: integration.AuthBot,
			AuthOptions: []integration.AuthOption{
				{Kind: integration.AuthOAuth2, LabelKey: "providers.authOAuth", Default: true, Fields: []integration.ConfigField{
					{Key: "channel", LabelKey: "providers.channel", Type: "text", Placeholder: "C01234567"},
				}},
				{Kind: integration.AuthBot, LabelKey: "providers.authBot", Fields: []integration.ConfigField{
					{Key: "token", LabelKey: "providers.token", Type: "password", Required: true, Secret: true},
					{Key: "channel", LabelKey: "providers.channel", Type: "text", Placeholder: "C01234567"},
				}},
			},
			Capabilities: []integration.Capability{integration.CapSend, integration.CapApprove},
			ConfigFields: []integration.ConfigField{
				{Key: "token", LabelKey: "providers.token", Type: "password", Required: true, Secret: true},
				{Key: "channel", LabelKey: "providers.channel", Type: "text", Placeholder: "C01234567"},
			}},
		Chat:     a,
		Probe:    a,
		Approver: a,
	}
}

type apiResponse struct {
	OK      bool   `json:"ok"`
	Error   string `json:"error"`
	Channel string `json:"channel"`
	TS      string `json:"ts"`
}

// Send implements integration.Chat.
func (a *Adapter) Send(ctx context.Context, cred integration.Credential, msg integration.ChatMessage) (string, error) {
	payload := map[string]any{
		"channel":   msg.Channel,
		"text":      msg.Text,
		"thread_ts": msg.Thread,
	}
	if len(msg.Blocks) > 0 {
		payload["blocks"] = msg.Blocks
	}
	return a.call(ctx, cred, "chat.postMessage", payload)
}

// RequestApproval implements integration.Approver with a Block Kit card.
func (a *Adapter) RequestApproval(ctx context.Context, cred integration.Credential, card integration.ApprovalCard) (string, error) {
	blocks := []any{
		map[string]any{
			"type": "section",
			"text": map[string]any{
				"type": "mrkdwn",
				"text": fmt.Sprintf("*Approval required* · `%s`\n```%s```\nRisk: *%s*", card.Tool, card.Command, card.Risk),
			},
		},
		map[string]any{
			"type": "actions",
			"elements": []any{
				map[string]any{
					"type":  "button",
					"text":  map[string]any{"type": "plain_text", "text": "Approve"},
					"style": "primary",
					"url":   card.ApproveURL,
				},
				map[string]any{
					"type":  "button",
					"text":  map[string]any{"type": "plain_text", "text": "Deny"},
					"style": "danger",
					"url":   card.DenyURL,
				},
			},
		},
	}
	raw, _ := json.Marshal(blocks)
	return a.Send(ctx, cred, integration.ChatMessage{
		Channel: card.Channel,
		Text:    "Approval required: " + card.Tool,
		Blocks:  raw,
	})
}

// Update implements integration.Chat.
func (a *Adapter) Update(ctx context.Context, cred integration.Credential, ref string, msg integration.ChatMessage) error {
	channel, ts, err := splitRef(ref)
	if err != nil {
		return err
	}
	_, err = a.call(ctx, cred, "chat.update", map[string]any{
		"channel": channel,
		"ts":      ts,
		"text":    msg.Text,
	})
	return err
}

func (a *Adapter) call(ctx context.Context, cred integration.Credential, method string, payload map[string]any) (string, error) {
	if cred.AccessToken == "" {
		return "", errors.New("slack: bot token is required")
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL()+"/"+method, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+cred.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("slack %s: status %d", method, resp.StatusCode)
	}
	var out apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if !out.OK {
		return "", fmt.Errorf("slack %s: %s", method, out.Error)
	}
	return out.Channel + ":" + out.TS, nil
}

func splitRef(ref string) (channel, ts string, err error) {
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errors.New("slack: invalid message reference")
	}
	return parts[0], parts[1], nil
}

func (a *Adapter) baseURL() string {
	if a.BaseURL != "" {
		return strings.TrimRight(a.BaseURL, "/")
	}
	return "https://slack.com/api"
}

func (a *Adapter) client() *http.Client {
	if a.HTTPClient != nil {
		return a.HTTPClient
	}
	return http.DefaultClient
}

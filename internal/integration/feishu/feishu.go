// Package feishu implements the Feishu (Lark) chat integration using a custom
// bot webhook.
package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Adapter sends messages to a Feishu custom-bot webhook.
type Adapter struct {
	Webhook    string
	BaseURL    string
	HTTPClient *http.Client
}

// New builds the Feishu integration. An empty webhook keeps the manifest
// discoverable but sends nothing.
func New(webhook string) *integration.Integration {
	a := &Adapter{Webhook: webhook, HTTPClient: &http.Client{Timeout: 20 * time.Second}}
	return &integration.Integration{
		Manifest: integration.Manifest{
			ID:   "feishu",
			Name: "Feishu",
			Kind: integration.KindChat,
			Auth: integration.AuthNone,
			AuthOptions: []integration.AuthOption{
				{Kind: integration.AuthBot, LabelKey: "providers.authApp", Default: true, Fields: []integration.ConfigField{
					{Key: "appId", LabelKey: "providers.appId", Type: "text", Required: true, Placeholder: "cli_..."},
					{Key: "appSecret", LabelKey: "providers.appSecret", Type: "password", Required: true, Secret: true},
					{Key: "chatId", LabelKey: "providers.chatId", Type: "text", Required: true, Placeholder: "oc_..."},
				}},
				{Kind: integration.AuthNone, LabelKey: "providers.authWebhook", Fields: []integration.ConfigField{
					{Key: "webhookUrl", LabelKey: "providers.webhookUrl", Type: "url", Required: true, Placeholder: "https://open.feishu.cn/open-apis/bot/v2/hook/..."},
				}},
			},
			Capabilities: []integration.Capability{integration.CapSend, integration.CapApprove},
			ConfigFields: []integration.ConfigField{
				{Key: "webhookUrl", LabelKey: "providers.webhookUrl", Type: "url", Placeholder: "https://open.feishu.cn/open-apis/bot/v2/hook/..."},
				{Key: "appId", LabelKey: "providers.appId", Type: "text", Placeholder: "cli_..."},
				{Key: "appSecret", LabelKey: "providers.appSecret", Type: "password", Secret: true},
				{Key: "chatId", LabelKey: "providers.chatId", Type: "text", Placeholder: "oc_..."},
			},
		},
		Chat:     a,
		Approver: a,
		Probe:    a,
	}
}

// Send implements integration.Chat with a text message.
func (a *Adapter) Send(ctx context.Context, cred integration.Credential, msg integration.ChatMessage) (string, error) {
	if appID(cred) != "" {
		return a.sendApp(ctx, cred, "text", map[string]string{"text": msg.Text})
	}
	payload := map[string]any{
		"msg_type": "text",
		"content":  map[string]string{"text": msg.Text},
	}
	return a.post(ctx, webhookFor(cred, a.Webhook), payload)
}

// Update implements integration.Chat. Feishu custom-bot messages cannot be
// edited, so updates are not supported.
func (a *Adapter) Update(context.Context, integration.Credential, string, integration.ChatMessage) error {
	return integration.ErrUnsupported
}

// RequestApproval implements integration.Approver with an interactive card.
func (a *Adapter) RequestApproval(ctx context.Context, cred integration.Credential, card integration.ApprovalCard) (string, error) {
	if appID(cred) != "" {
		text := fmt.Sprintf("Approval required: %s", card.ApproveURL)
		return a.sendApp(ctx, cred, "text", map[string]string{"text": text})
	}
	text := fmt.Sprintf("**Approval required** · `%s`\n```%s```\nRisk: **%s**", card.Tool, card.Command, card.Risk)
	payload := map[string]any{
		"msg_type": "interactive",
		"card": map[string]any{
			"config": map[string]any{"wide_screen_mode": true},
			"elements": []any{
				map[string]any{
					"tag":  "div",
					"text": map[string]any{"tag": "lark_md", "content": text},
				},
				map[string]any{
					"tag": "action",
					"actions": []any{
						map[string]any{
							"tag":  "button",
							"text": map[string]any{"tag": "plain_text", "content": "Approve"},
							"type": "primary",
							"url":  card.ApproveURL,
						},
						map[string]any{
							"tag":  "button",
							"text": map[string]any{"tag": "plain_text", "content": "Deny"},
							"type": "danger",
							"url":  card.DenyURL,
						},
					},
				},
			},
		},
	}
	return a.post(ctx, webhookFor(cred, a.Webhook), payload)
}

func webhookFor(cred integration.Credential, fallback string) string {
	if cred.Fields != nil {
		if value := strings.TrimSpace(cred.Fields["webhookUrl"]); value != "" {
			return value
		}
	}
	return fallback
}

func (a *Adapter) post(ctx context.Context, webhook string, payload map[string]any) (string, error) {
	if webhook == "" {
		return "", fmt.Errorf("feishu: webhook is not configured")
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("feishu: status %d", resp.StatusCode)
	}
	var out struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Code != 0 {
		return "", fmt.Errorf("feishu: %d %s", out.Code, out.Msg)
	}
	return "", nil
}

func (a *Adapter) client() *http.Client {
	if a.HTTPClient != nil {
		return a.HTTPClient
	}
	return http.DefaultClient
}

// trim is kept for future endpoint normalisation.
func trim(v string) string { return strings.TrimRight(v, "/") }

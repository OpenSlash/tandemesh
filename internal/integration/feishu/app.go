package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// Probe validates app credentials via tenant_access_token, or accepts a webhook configuration.
func (a *Adapter) Probe(ctx context.Context, cred integration.Credential) error {
	if appID(cred) == "" {
		return nil
	}
	_, err := a.tenantToken(ctx, cred)
	return err
}

func appID(cred integration.Credential) string {
	if cred.Fields == nil {
		return ""
	}
	return strings.TrimSpace(cred.Fields["appId"])
}

func appSecret(cred integration.Credential) string {
	if cred.Fields != nil {
		if value := strings.TrimSpace(cred.Fields["appSecret"]); value != "" {
			return value
		}
	}
	return cred.AccessToken
}

func (a *Adapter) tenantToken(ctx context.Context, cred integration.Credential) (string, error) {
	payload := map[string]string{"app_id": appID(cred), "app_secret": appSecret(cred)}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.base()+"/open-apis/auth/v3/tenant_access_token/internal", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 || out.Code != 0 || out.TenantAccessToken == "" {
		return "", fmt.Errorf("feishu: tenant token status %d code %d %s", resp.StatusCode, out.Code, out.Msg)
	}
	return out.TenantAccessToken, nil
}

func (a *Adapter) sendApp(ctx context.Context, cred integration.Credential, msgType string, content any) (string, error) {
	chatID := ""
	if cred.Fields != nil {
		chatID = strings.TrimSpace(cred.Fields["chatId"])
	}
	if chatID == "" {
		return "", fmt.Errorf("feishu: chatId is required")
	}
	token, err := a.tenantToken(ctx, cred)
	if err != nil {
		return "", err
	}
	raw, _ := json.Marshal(content)
	payload := map[string]any{"receive_id": chatID, "msg_type": msgType, "content": string(raw)}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.base()+"/open-apis/im/v1/messages?receive_id_type=chat_id", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := a.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			MessageID string `json:"message_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if resp.StatusCode >= 300 || out.Code != 0 {
		return "", fmt.Errorf("feishu: send status %d code %d %s", resp.StatusCode, out.Code, out.Msg)
	}
	return out.Data.MessageID, nil
}

func (a *Adapter) base() string {
	if a.BaseURL != "" {
		return strings.TrimRight(a.BaseURL, "/")
	}
	return "https://open.feishu.cn"
}

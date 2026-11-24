package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func (a *Adapter) RegisterWebhook(ctx context.Context, cred integration.Credential, instance, url, secret string) (string, error) {
	if cred.AccessToken == "" {
		return "", fmt.Errorf("github: missing token")
	}
	body, _ := json.Marshal(map[string]any{
		"name":   "web",
		"active": true,
		"events": []string{"issues", "issue_comment"},
		"config": map[string]string{
			"url":          url,
			"content_type": "json",
			"secret":       secret,
		},
	})
	endpoint := fmt.Sprintf("%s/repos/%s/hooks", a.baseURLFor(cred), instance)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
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
		return "", fmt.Errorf("github webhook: status %d", resp.StatusCode)
	}
	var out struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", out.ID), nil
}

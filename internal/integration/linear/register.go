package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// RegisterWebhook creates a Linear webhook for issue and comment events.
func (a *Adapter) RegisterWebhook(ctx context.Context, cred integration.Credential, instance, url, secret string) (string, error) {
	if cred.AccessToken == "" {
		return "", fmt.Errorf("linear: missing token")
	}
	query := `mutation CreateWebhook($url: String!, $secret: String) {
	  webhookCreate(input: {url: $url, secret: $secret, resourceTypes: ["Issue", "Comment"], allPublicTeams: true}) {
	    success
	    webhook { id }
	  }
	}`
	body, _ := json.Marshal(map[string]any{"query": query, "variables": map[string]any{"url": url, "secret": secret}})
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
		return "", fmt.Errorf("linear webhook: status %d", resp.StatusCode)
	}
	var out struct {
		Data struct {
			WebhookCreate struct {
				Success bool `json:"success"`
				Webhook struct {
					ID string `json:"id"`
				} `json:"webhook"`
			} `json:"webhookCreate"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Errors) > 0 {
		return "", fmt.Errorf("linear webhook: %s", out.Errors[0].Message)
	}
	return out.Data.WebhookCreate.Webhook.ID, nil
}

package jira

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// RegisterWebhook creates a Jira Cloud dynamic webhook for issue and comment
// events. Jira may require an OAuth 2.0 app with manage:jira-webhook; when the
// API rejects the credential the caller receives a clear error.
func (a *Adapter) RegisterWebhook(ctx context.Context, cred integration.Credential, instance, url, secret string) (string, error) {
	base := strings.TrimRight(instance, "/")
	if base == "" {
		base = strings.TrimRight(cred.BaseURL, "/")
	}
	if base == "" {
		return "", fmt.Errorf("jira webhook: site URL is required")
	}
	body, _ := json.Marshal(map[string]any{
		"url": url,
		"webhooks": []map[string]any{{
			"events": []string{
				"jira:issue_created", "jira:issue_updated",
				"comment_created", "comment_updated",
			},
		}},
	})
	resp, err := a.do(ctx, http.MethodPost, base+"/rest/api/3/webhook", cred, body)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("jira webhook: status %d %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out struct {
		Results []struct {
			CreatedWebhookID int64 `json:"createdWebhookId"`
			Errors           []struct {
				Message string `json:"message"`
			} `json:"errors"`
		} `json:"webhookRegistrationResult"`
	}
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&out); err != nil {
		return "", fmt.Errorf("jira webhook: invalid response: %s", strings.TrimSpace(string(raw)))
	}
	if len(out.Results) == 0 {
		return "", fmt.Errorf("jira webhook: empty registration result")
	}
	if len(out.Results[0].Errors) > 0 {
		return "", fmt.Errorf("jira webhook: %s", out.Results[0].Errors[0].Message)
	}
	return fmt.Sprintf("%d", out.Results[0].CreatedWebhookID), nil
}

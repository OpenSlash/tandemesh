package plane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// RegisterWebhook creates a workspace webhook scoped to the configured project.
func (a *Adapter) RegisterWebhook(ctx context.Context, cred integration.Credential, instance, url, secret string) (string, error) {
	slug, project, err := splitInstance(instance)
	if err != nil {
		return "", err
	}
	body, _ := json.Marshal(map[string]any{
		"url":       url,
		"is_active": true,
		"project":   project,
	})
	endpoint := a.apiAt(cred, "/workspaces/"+slug+"/webhooks/")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("X-API-Key", cred.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.client().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("plane webhook: status %d", resp.StatusCode)
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.ID == "" {
		out.ID = strings.TrimSpace(project + ":webhook")
	}
	return out.ID, nil
}

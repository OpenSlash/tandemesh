package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/ingress"
)

type appWebhookContextKey struct{}

func appWebhookVerified(ctx context.Context) bool {
	ok, _ := ctx.Value(appWebhookContextKey{}).(bool)
	return ok
}

// githubAppWebhook resolves an app-level GitHub webhook to a repository
// binding, then reuses the automation-scoped webhook pipeline.
func (s *Server) githubAppWebhook(w http.ResponseWriter, r *http.Request) {
	if s.cfg.GitHubAppWebhookSecret == "" {
		writeErr(w, http.StatusBadRequest, "E_APP", "GitHub App webhook secret is not configured")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", "body too large")
		return
	}
	if err := ingress.VerifyGitHub(s.cfg.GitHubAppWebhookSecret, body, r.Header.Get("X-Hub-Signature-256")); err != nil {
		writeErr(w, http.StatusUnauthorized, "E_SIG", "signature verification failed")
		return
	}
	var pr githubAppPRPayload
	if err := json.Unmarshal(body, &pr); err == nil && pr.PullRequest.Number > 0 {
		deliveryID := s.recordAppDelivery(r.Context(), r.Header.Get("X-GitHub-Delivery"), "pull_request", body)
		linked := s.linkGitHubPullRequestEvent(r.Context(), pr)
		_ = s.store.UpdateWebhookDelivery(r.Context(), deliveryID, "accepted", "", "")
		writeJSON(w, http.StatusAccepted, map[string]any{"linked": linked})
		return
	}
	var push githubAppPushPayload
	if err := json.Unmarshal(body, &push); err == nil && len(push.Commits) > 0 {
		deliveryID := s.recordAppDelivery(r.Context(), r.Header.Get("X-GitHub-Delivery"), "push", body)
		linked := s.linkGitHubPushEvent(r.Context(), push)
		_ = s.store.UpdateWebhookDelivery(r.Context(), deliveryID, "accepted", "", "")
		writeJSON(w, http.StatusAccepted, map[string]any{"linked": linked})
		return
	}
	payload, err := ingress.ParseGitHub(body)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"skipped": "NON_ISSUE_EVENT"})
		return
	}
	if payload.Repository.FullName == "" {
		writeJSON(w, http.StatusOK, map[string]string{"skipped": "NO_REPOSITORY"})
		return
	}
	binding, err := s.store.FindBinding(r.Context(), "github", payload.Repository.FullName)
	if err != nil || binding == nil {
		writeJSON(w, http.StatusOK, map[string]string{"skipped": "NO_BINDING"})
		return
	}
	conn, err := s.store.GetConnection(r.Context(), binding.ConnectionID)
	if err != nil || conn == nil {
		writeErr(w, http.StatusNotFound, "E_CONNECTION", "connection not found")
		return
	}
	var automation *domain.Automation
	if conn.AutomationID != "" {
		automation, _ = s.store.GetAutomation(r.Context(), conn.AutomationID)
	}
	if automation == nil {
		if items, err := s.store.ListAutomations(r.Context(), conn.WorkspaceID); err == nil {
			for _, item := range items {
				if item.ConnectionID == conn.ID && item.Enabled {
					automation = item
					break
				}
			}
		}
	}
	if automation == nil {
		writeErr(w, http.StatusNotFound, "E_AUTOMATION", "connection has no event automation")
		return
	}
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "github")
	rctx.URLParams.Add("automation", automation.ID)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, appWebhookContextKey{}, true)
	r = r.WithContext(ctx)
	r.Body = io.NopCloser(bytes.NewReader(body))
	s.webhook(w, r)
}

func (s *Server) recordAppDelivery(ctx context.Context, delivery, kind string, body []byte) string {
	id := "whd_" + ulid.Make().String()
	_ = s.store.CreateWebhookDelivery(ctx, &domain.WebhookDelivery{
		ID: id, Provider: "github", Delivery: delivery, Kind: kind, Body: string(body),
	})
	return id
}

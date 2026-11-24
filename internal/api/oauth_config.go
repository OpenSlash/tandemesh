package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

const oauthAppSubject = "oauth-app"

var oauthAppProviders = map[string]bool{
	"github": true, "linear": true, "slack": true, "gitlab": true, "jira": true, "outlook": true, "notion": true,
}

func (s *Server) githubClientCredentials(ctx context.Context) (string, string, bool) {
	cfg := oauthClientConfig{ClientID: s.cfg.GitHubClientID, ClientSecret: s.cfg.GitHubClientSecret}
	if !s.applyStoredOAuth(ctx, &cfg, "github") {
		return "", "", false
	}
	return cfg.ClientID, cfg.ClientSecret, true
}

func (s *Server) applyStoredOAuth(ctx context.Context, cfg *oauthClientConfig, provider string) bool {
	if cfg.ClientID != "" && cfg.ClientSecret != "" {
		return true
	}
	if s.store == nil || s.vault == nil {
		return false
	}
	app, err := s.store.GetOAuthApp(ctx, provider)
	if err != nil || app == nil {
		return false
	}
	cred, ok, err := s.vault.Get(ctx, provider, oauthAppSubject)
	if err != nil || !ok || cred.AccessToken == "" {
		return false
	}
	cfg.ClientID = app.ClientID
	cfg.ClientSecret = cred.AccessToken
	return true
}

type oauthAppRequest struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

func (s *Server) getOAuthApp(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if !oauthAppProviders[provider] {
		writeErr(w, http.StatusNotFound, "E_PROVIDER", "provider does not use OAuth")
		return
	}
	clientID := ""
	configured := false
	if provider == "github" {
		if id, _, ok := s.githubClientCredentials(r.Context()); ok {
			clientID = id
			configured = true
		}
	} else if cfg, ok := s.oauthProviderConfig(r.Context(), provider); ok {
		clientID = cfg.ClientID
		configured = true
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"provider": provider, "configured": configured, "clientId": clientID,
		"clientSecret": maskSecret(configured), "callbackUrl": s.oauthRedirectURI(provider),
	})
}

func (s *Server) putOAuthApp(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if !oauthAppProviders[provider] {
		writeErr(w, http.StatusNotFound, "E_PROVIDER", "provider does not use OAuth")
		return
	}
	var req oauthAppRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	req.ClientID = strings.TrimSpace(req.ClientID)
	req.ClientSecret = strings.TrimSpace(req.ClientSecret)
	if req.ClientID == "" {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "clientId is required")
		return
	}
	if req.ClientSecret == "" {
		if existing, ok, _ := s.vault.Get(r.Context(), provider, oauthAppSubject); ok {
			req.ClientSecret = existing.AccessToken
		}
	}
	if req.ClientSecret == "" {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "clientSecret is required")
		return
	}
	if err := s.store.PutOAuthApp(r.Context(), provider, req.ClientID); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if err := s.vault.Put(r.Context(), integration.Credential{
		Provider: provider, Subject: oauthAppSubject, AccessToken: req.ClientSecret,
	}); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_VAULT", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"provider": provider, "configured": true, "clientId": req.ClientID,
		"clientSecret": maskSecret(true), "callbackUrl": s.oauthRedirectURI(provider),
	})
}

func (s *Server) deleteOAuthApp(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	if !oauthAppProviders[provider] {
		writeErr(w, http.StatusNotFound, "E_PROVIDER", "provider does not use OAuth")
		return
	}
	_ = s.store.DeleteOAuthApp(r.Context(), provider)
	_, _ = s.vault.Delete(r.Context(), provider, oauthAppSubject)
	w.WriteHeader(http.StatusNoContent)
}

func maskSecret(configured bool) string {
	if !configured {
		return ""
	}
	return "••••••••"
}

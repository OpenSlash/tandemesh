package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// providerCredential mirrors the runtime credential resolution: vault first,
// then environment fallbacks.
func (s *Server) providerCredential(ctx context.Context, provider string, subjects ...string) integration.Credential {
	if provider == "github" && len(subjects) > 0 && subjects[0] != "" {
		if cred, ok := s.githubAppCredential(ctx, subjects[0]); ok {
			return cred
		}
	}
	subject := "default"
	if len(subjects) > 0 && subjects[0] != "" {
		subject = subjects[0]
	}
	if s.vault != nil {
		if cred, ok, err := s.vault.Ensure(ctx, provider, subject); err == nil && ok && cred.AccessToken != "" {
			return cred
		}
	}
	switch provider {
	case "github":
		return integration.Credential{Provider: provider, AccessToken: s.cfg.GitHubToken}
	case "linear":
		return integration.Credential{Provider: provider, AccessToken: s.cfg.LinearToken}
	case "jira":
		return integration.Credential{Provider: provider, Subject: s.cfg.JiraEmail, AccessToken: s.cfg.JiraToken}
	case "notion":
		return integration.Credential{Provider: provider, AccessToken: s.cfg.NotionToken}
	case "outlook":
		return integration.Credential{Provider: provider, AccessToken: s.cfg.GraphToken}
	case "outline":
		return integration.Credential{Provider: provider, AccessToken: s.cfg.OutlineToken}
	case "slack":
		return integration.Credential{Provider: provider, AccessToken: s.cfg.SlackToken}
	}
	return integration.Credential{Provider: provider}
}

// probeProvider checks a stored credential against the provider.
func (s *Server) probeProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ad, ok := s.integrations.Get(id)
	if !ok {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "provider not found")
		return
	}
	if ad.Probe == nil {
		writeErr(w, http.StatusBadRequest, "E_PROBE", "provider does not support probing")
		return
	}
	cred := s.providerCredential(r.Context(), id)
	if cred.AccessToken == "" {
		writeErr(w, http.StatusBadRequest, "E_CREDENTIAL", "no credential configured for "+id)
		return
	}
	if err := ad.Probe.Probe(r.Context(), cred); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"ok": false, "provider": id, "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "provider": id})
}

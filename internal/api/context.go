package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

// contextCredential resolves the token for a document provider.
func (s *Server) contextCredential(ctx context.Context, provider string) integration.Credential {
	if s.vault != nil {
		if cred, ok, err := s.vault.Ensure(ctx, provider, "default"); err == nil && ok && cred.AccessToken != "" {
			return cred
		}
	}
	switch provider {
	case "notion":
		return integration.Credential{Provider: provider, AccessToken: s.cfg.NotionToken}
	case "outlook":
		return integration.Credential{Provider: provider, AccessToken: s.cfg.GraphToken}
	case "outline":
		return integration.Credential{Provider: provider, AccessToken: s.cfg.OutlineToken}
	}
	return integration.Credential{Provider: provider}
}

// contextSearch proxies a read-only search to a document integration.
func (s *Server) contextSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeErr(w, http.StatusBadRequest, "E_QUERY", "q is required")
		return
	}
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = "notion"
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	ad, ok := s.integrations.Get(provider)
	if !ok || ad.Docs == nil {
		writeErr(w, http.StatusServiceUnavailable, "E_PROVIDER", provider+" is not configured")
		return
	}
	hits, err := ad.Docs.Search(r.Context(), s.contextCredential(r.Context(), provider), query, limit)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_CONTEXT", err.Error())
		return
	}
	if hits == nil {
		hits = []integration.DocHit{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": hits})
}

// contextFetch returns one document with its source URL for citation.
func (s *Server) contextFetch(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("id")
	if ref == "" {
		ref = r.URL.Query().Get("url")
	}
	if ref == "" {
		writeErr(w, http.StatusBadRequest, "E_URL", "id or url is required")
		return
	}
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		provider = "notion"
	}
	ad, ok := s.integrations.Get(provider)
	if !ok || ad.Docs == nil {
		writeErr(w, http.StatusServiceUnavailable, "E_PROVIDER", provider+" is not configured")
		return
	}
	doc, err := ad.Docs.Fetch(r.Context(), s.contextCredential(r.Context(), provider), ref)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_CONTEXT", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

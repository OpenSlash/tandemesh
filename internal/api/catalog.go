package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

type catalogRequest struct {
	Token        string `json:"token"`
	BaseURL      string `json:"baseUrl"`
	Workspace    string `json:"workspace"`
	ConnectionID string `json:"connectionId"`
}

func (s *Server) integrationCatalog(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	ad, ok := s.integrations.Get(provider)
	if !ok {
		writeErr(w, http.StatusNotFound, "E_PROVIDER", "provider is not registered")
		return
	}
	reader, ok := ad.Reader.(integration.CatalogReader)
	if !ok {
		writeErr(w, http.StatusBadRequest, "E_UNSUPPORTED", "provider does not expose a catalog")
		return
	}
	var req catalogRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	cred := integration.Credential{
		Provider: provider, AccessToken: strings.TrimSpace(req.Token), BaseURL: strings.TrimRight(strings.TrimSpace(req.BaseURL), "/"),
	}
	if cred.AccessToken == "" && req.ConnectionID != "" && s.vault != nil {
		if source, err := s.store.GetConnection(r.Context(), req.ConnectionID); err == nil && source != nil && source.Provider == provider {
			if stored, ok, err := s.vault.Get(r.Context(), provider, source.ID); err == nil && ok {
				cred.AccessToken = stored.AccessToken
				if cred.BaseURL == "" {
					cred.BaseURL = source.BaseURL
				}
			}
		}
	}
	if cred.AccessToken == "" {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "token is required")
		return
	}
	if req.Workspace == "" {
		items, err := reader.ListWorkspaces(r.Context(), cred)
		if err != nil {
			writeErr(w, http.StatusBadGateway, "E_CATALOG", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"workspaces": items})
		return
	}
	items, err := reader.ListProjects(r.Context(), cred, req.Workspace)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "E_CATALOG", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": items})
}

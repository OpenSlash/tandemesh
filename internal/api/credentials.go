package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

type credentialRequest struct {
	Provider     string   `json:"provider"`
	Subject      string   `json:"subject"`
	AccessToken  string   `json:"accessToken"`
	RefreshToken string   `json:"refreshToken"`
	ExpiresAt    string   `json:"expiresAt"`
	Scopes       []string `json:"scopes"`
}

func (s *Server) listCredentials(w http.ResponseWriter, r *http.Request) {
	if s.vault == nil {
		writeErr(w, http.StatusServiceUnavailable, "E_VAULT", "credential vault is not configured")
		return
	}
	items, err := s.vault.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if items == nil {
		items = nil
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) putCredential(w http.ResponseWriter, r *http.Request) {
	if s.vault == nil {
		writeErr(w, http.StatusServiceUnavailable, "E_VAULT", "credential vault is not configured")
		return
	}
	var req credentialRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if strings.TrimSpace(req.Provider) == "" || req.AccessToken == "" {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "provider and accessToken are required")
		return
	}
	cred := integration.Credential{
		Provider:     req.Provider,
		Subject:      req.Subject,
		AccessToken:  req.AccessToken,
		RefreshToken: req.RefreshToken,
		Scopes:       req.Scopes,
	}
	if req.ExpiresAt != "" {
		if t, err := time.Parse(time.RFC3339, req.ExpiresAt); err == nil {
			cred.ExpiresAt = t
		} else {
			writeErr(w, http.StatusBadRequest, "E_EXPIRES", "expiresAt must be RFC3339")
			return
		}
	}
	if err := s.vault.Put(r.Context(), cred); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_VAULT", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteCredential(w http.ResponseWriter, r *http.Request) {
	if s.vault == nil {
		writeErr(w, http.StatusServiceUnavailable, "E_VAULT", "credential vault is not configured")
		return
	}
	ok, err := s.vault.Delete(r.Context(), chi.URLParam(r, "provider"), chi.URLParam(r, "subject"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "credential not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

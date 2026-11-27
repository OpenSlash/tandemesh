package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

type identityRequest struct {
	Provider   string `json:"provider"`
	ExternalID string `json:"externalId"`
	Login      string `json:"login"`
	Email      string `json:"email"`
	MemberID   string `json:"memberId"`
}

func (s *Server) listExternalIdentities(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	// Tasks synced before identities were collected during sync still name
	// their people; list them too so they can be mapped.
	if actors, err := s.store.ListTaskActors(r.Context(), workspaceID); err == nil {
		for _, actor := range actors {
			_, _ = s.store.EnsureExternalIdentity(r.Context(), workspaceID, actor.Provider, actor.Login, actor.Login, "")
		}
	}
	items, err := s.store.ListExternalIdentities(r.Context(), workspaceID, r.URL.Query().Get("provider"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) createExternalIdentity(w http.ResponseWriter, r *http.Request) {
	workspaceID := chi.URLParam(r, "workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	var req identityRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	req.Provider = strings.TrimSpace(req.Provider)
	req.ExternalID = strings.TrimSpace(req.ExternalID)
	req.MemberID = strings.TrimSpace(req.MemberID)
	if req.Provider == "" || req.ExternalID == "" || req.MemberID == "" {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "provider, externalId and memberId are required")
		return
	}
	identity := &domain.ExternalIdentity{
		ID: "ext_" + ulid.Make().String(), WorkspaceID: workspaceID,
		Provider: req.Provider, ExternalID: req.ExternalID,
		Login: req.Login, Email: req.Email, MemberID: req.MemberID,
	}
	if identity.Login == "" {
		identity.Login = identity.ExternalID
	}
	if err := s.store.UpsertExternalIdentity(r.Context(), identity); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"identity": identity})
}

func (s *Server) deleteExternalIdentity(w http.ResponseWriter, r *http.Request) {
	identity, err := s.store.GetExternalIdentity(r.Context(), chi.URLParam(r, "id"))
	if err != nil || identity == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "identity not found")
		return
	}
	if !s.requireWorkspace(w, r, identity.WorkspaceID) {
		return
	}
	if _, err := s.store.DeleteExternalIdentity(r.Context(), identity.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) patchExternalIdentity(w http.ResponseWriter, r *http.Request) {
	identity, err := s.store.GetExternalIdentity(r.Context(), chi.URLParam(r, "id"))
	if err != nil || identity == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "identity not found")
		return
	}
	if !s.requireWorkspace(w, r, identity.WorkspaceID) {
		return
	}
	var req identityRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.MemberID != "" {
		identity.MemberID = strings.TrimSpace(req.MemberID)
	}
	if req.Login != "" {
		identity.Login = req.Login
	}
	if req.Email != "" {
		identity.Email = req.Email
	}
	if err := s.store.UpsertExternalIdentity(r.Context(), identity); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"identity": identity})
}

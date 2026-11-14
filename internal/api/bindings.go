package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

type bindingRequest struct {
	ExternalID string            `json:"externalId"`
	Name       string            `json:"name"`
	Kind       string            `json:"kind"`
	Direction  string            `json:"direction"`
	AccountID  string            `json:"accountId"`
	Settings   map[string]string `json:"settings"`
	Enabled    *bool             `json:"enabled"`
}

func bindingKind(provider string) string {
	switch provider {
	case "github", "gitlab", "bitbucket":
		return "repo"
	case "linear", "jira", "plane":
		return "project"
	case "slack", "feishu":
		return "channel"
	default:
		return "resource"
	}
}

func normalizeBindingDirection(value string) string {
	if value == "read" {
		return "read"
	}
	return "read_write"
}

func (s *Server) listConnectionBindings(w http.ResponseWriter, r *http.Request) {
	conn, err := s.store.GetConnection(r.Context(), chi.URLParam(r, "id"))
	if err != nil || conn == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "connection not found")
		return
	}
	if !s.requireWorkspace(w, r, conn.WorkspaceID) {
		return
	}
	items, err := s.store.ListBindings(r.Context(), conn.WorkspaceID, conn.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) createConnectionBinding(w http.ResponseWriter, r *http.Request) {
	conn, err := s.store.GetConnection(r.Context(), chi.URLParam(r, "id"))
	if err != nil || conn == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "connection not found")
		return
	}
	if !s.requireWorkspace(w, r, conn.WorkspaceID) {
		return
	}
	var req bindingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	req.ExternalID = strings.TrimSpace(req.ExternalID)
	if req.ExternalID == "" {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "externalId is required")
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	binding := &domain.ResourceBinding{
		ID: "bind_" + ulid.Make().String(), WorkspaceID: conn.WorkspaceID,
		ConnectionID: conn.ID, Provider: conn.Provider, ExternalID: req.ExternalID,
		Name: req.Name, Kind: req.Kind, Direction: normalizeBindingDirection(req.Direction),
		AccountID: req.AccountID, Settings: req.Settings, Enabled: enabled,
	}
	if binding.Name == "" {
		binding.Name = binding.ExternalID
	}
	if binding.Kind == "" {
		binding.Kind = "resource"
	}
	if err := s.store.UpsertBinding(r.Context(), binding); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"binding": binding})
}

func (s *Server) patchBinding(w http.ResponseWriter, r *http.Request) {
	binding, err := s.store.GetBinding(r.Context(), chi.URLParam(r, "id"))
	if err != nil || binding == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "binding not found")
		return
	}
	if !s.requireWorkspace(w, r, binding.WorkspaceID) {
		return
	}
	var req bindingRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.Name != "" {
		binding.Name = req.Name
	}
	if req.Kind != "" {
		binding.Kind = req.Kind
	}
	if req.AccountID != "" {
		binding.AccountID = req.AccountID
	}
	if req.Direction != "" {
		binding.Direction = normalizeBindingDirection(req.Direction)
	}
	if req.Settings != nil {
		binding.Settings = req.Settings
	}
	if req.Enabled != nil {
		binding.Enabled = *req.Enabled
	}
	if err := s.store.UpsertBinding(r.Context(), binding); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"binding": binding})
}

func (s *Server) deleteBinding(w http.ResponseWriter, r *http.Request) {
	binding, err := s.store.GetBinding(r.Context(), chi.URLParam(r, "id"))
	if err != nil || binding == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "binding not found")
		return
	}
	if !s.requireWorkspace(w, r, binding.WorkspaceID) {
		return
	}
	if _, err := s.store.DeleteBinding(r.Context(), binding.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

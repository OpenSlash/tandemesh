package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Server) listSavedViews(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListSavedViews(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if items == nil {
		items = []*domain.SavedView{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type savedViewRequest struct {
	Name    string            `json:"name"`
	Filters map[string]string `json:"filters"`
}

func (s *Server) createSavedView(w http.ResponseWriter, r *http.Request) {
	var req savedViewRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.Name == "" {
		writeErr(w, http.StatusBadRequest, "E_NAME", "name is required")
		return
	}
	allowed := map[string]bool{"status": true, "label": true, "agent": true, "provider": true, "inbox": true, "assignee": true, "iteration": true}
	filters := map[string]string{}
	for key, value := range req.Filters {
		if allowed[key] && value != "" {
			filters[key] = value
		}
	}
	view := &domain.SavedView{
		ID: "view_" + ulid.Make().String(), WorkspaceID: chi.URLParam(r, "id"),
		Name: req.Name, Filters: filters,
	}
	if err := s.store.CreateSavedView(r.Context(), view); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, view)
}

type renameViewRequest struct {
	Name string `json:"name"`
}

func (s *Server) renameSavedView(w http.ResponseWriter, r *http.Request) {
	var req renameViewRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeErr(w, http.StatusBadRequest, "E_NAME", "name is required")
		return
	}
	ok, err := s.store.RenameSavedView(r.Context(), chi.URLParam(r, "view"), req.Name)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "view not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"name": req.Name})
}

func (s *Server) deleteSavedView(w http.ResponseWriter, r *http.Request) {
	ok, err := s.store.DeleteSavedView(r.Context(), chi.URLParam(r, "view"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "view not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

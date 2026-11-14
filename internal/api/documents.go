package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

type documentRequest struct {
	WorkspaceID string   `json:"workspaceId"`
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	Tags        []string `json:"tags"`
}

func (s *Server) listDocuments(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspaceId")
	if workspaceID == "" {
		writeErr(w, http.StatusBadRequest, "E_WORKSPACE", "workspaceId is required")
		return
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	items, err := s.store.ListDocuments(r.Context(), workspaceID, 100)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"documents": items})
}

func (s *Server) getDocument(w http.ResponseWriter, r *http.Request) {
	doc, err := s.store.GetDocument(r.Context(), chi.URLParam(r, "id"))
	if err != nil || doc == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "document not found")
		return
	}
	if !s.requireWorkspace(w, r, doc.WorkspaceID) {
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) createDocument(w http.ResponseWriter, r *http.Request) {
	var req documentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		writeErr(w, http.StatusBadRequest, "E_TITLE", "title is required")
		return
	}
	if req.WorkspaceID == "" {
		writeErr(w, http.StatusBadRequest, "E_WORKSPACE", "workspaceId is required")
		return
	}
	if !s.requireWorkspace(w, r, req.WorkspaceID) {
		return
	}
	doc := &domain.Document{
		ID:          "doc_" + ulid.Make().String(),
		WorkspaceID: req.WorkspaceID,
		Title:       strings.TrimSpace(req.Title),
		Body:        req.Body,
		Tags:        req.Tags,
		CreatedBy:   s.cfg.User,
	}
	if err := s.store.CreateDocument(r.Context(), doc); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, doc)
}

func (s *Server) updateDocument(w http.ResponseWriter, r *http.Request) {
	doc, err := s.store.GetDocument(r.Context(), chi.URLParam(r, "id"))
	if err != nil || doc == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "document not found")
		return
	}
	if !s.requireWorkspace(w, r, doc.WorkspaceID) {
		return
	}
	var req documentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if strings.TrimSpace(req.Title) != "" {
		doc.Title = strings.TrimSpace(req.Title)
	}
	doc.Body = req.Body
	doc.Tags = req.Tags
	if err := s.store.UpdateDocument(r.Context(), doc); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, doc)
}

func (s *Server) deleteDocument(w http.ResponseWriter, r *http.Request) {
	doc, err := s.store.GetDocument(r.Context(), chi.URLParam(r, "id"))
	if err != nil || doc == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "document not found")
		return
	}
	if !s.requireWorkspace(w, r, doc.WorkspaceID) {
		return
	}
	if err := s.store.DeleteDocument(r.Context(), doc.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

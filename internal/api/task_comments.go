package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Server) listTaskComments(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	items, err := s.store.ListTaskComments(r.Context(), task.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if items == nil {
		items = []*domain.TaskComment{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type commentRequest struct {
	Body   string `json:"body"`
	Author string `json:"author"`
}

func (s *Server) addTaskComment(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	var req commentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	req.Body = strings.TrimSpace(req.Body)
	if req.Body == "" {
		writeErr(w, http.StatusBadRequest, "E_BODY", "body is required")
		return
	}
	author := req.Author
	if author == "" {
		author = userOf(r)
	}
	comment := &domain.TaskComment{
		ID: "cmt_" + ulid.Make().String(), TaskID: task.ID, Author: author,
		Body: req.Body, CreatedAt: time.Now().UTC(),
	}
	if err := s.store.AddTaskComment(r.Context(), comment); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	s.recordTaskEvent(r.Context(), task.ID, "task.comment", map[string]any{
		"commentId": comment.ID, "body": comment.Body, "author": comment.Author,
	})
	writeJSON(w, http.StatusCreated, comment)
}

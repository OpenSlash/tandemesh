package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

type taskNoteRequest struct {
	Author string `json:"author"`
	Body   string `json:"body"`
}

func (s *Server) listTaskNotes(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	notes, err := s.store.ListTaskNotes(r.Context(), task.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if notes == nil {
		notes = []*domain.TaskNote{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": notes})
}

// addTaskNote records a local instruction. It is not written to the source.
func (s *Server) addTaskNote(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireTask(w, r, task) {
		return
	}
	var req taskNoteRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	req.Body = strings.TrimSpace(req.Body)
	if req.Body == "" {
		writeErr(w, http.StatusBadRequest, "E_BODY", "body is required")
		return
	}
	if req.Author == "" {
		req.Author = "human"
	}
	note := &domain.TaskNote{
		ID:        "note_" + ulid.Make().String(),
		TaskID:    task.ID,
		Author:    req.Author,
		Body:      req.Body,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.AddTaskNote(r.Context(), note); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	s.recordTaskEvent(r.Context(), task.ID, "task.note", map[string]any{
		"noteId": note.ID, "body": note.Body, "author": note.Author,
	})
	writeJSON(w, http.StatusCreated, note)
}

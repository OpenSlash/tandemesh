package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

type taskCommandRequest struct {
	Text   string `json:"text"`
	Author string `json:"author"`
	Repo   string `json:"repo"`
	Base   string `json:"base"`
}

// taskCommand interprets a short instruction on a task:
//
//	/orca team             launch lead + implementer + reviewer
//	/orca run <agent>      start one implementer run
//	/orca note <text>      record a team note
//	anything else          record a team note
func (s *Server) taskCommand(w http.ResponseWriter, r *http.Request) {
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
	var req taskCommandRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		writeErr(w, http.StatusBadRequest, "E_TEXT", "text is required")
		return
	}
	author := req.Author
	if author == "" {
		author = "human"
	}
	body := text
	if strings.HasPrefix(strings.ToLower(body), "/orca") {
		body = strings.TrimSpace(body[len("/orca"):])
	}
	words := strings.Fields(body)
	if len(words) == 0 {
		writeErr(w, http.StatusBadRequest, "E_TEXT", "empty command")
		return
	}

	switch strings.ToLower(words[0]) {
	case "team":
		items, err := s.launchTeam(r.Context(), task, req.Repo, req.Base, []teamRoleRequest{
			{Role: domain.RoleLead, AgentID: "claude"},
			{Role: domain.RoleImplementer, AgentID: "codex"},
			{Role: domain.RoleReviewer, AgentID: "claude"},
		})
		if err != nil {
			writeErr(w, http.StatusBadRequest, "E_TEAM", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"action": "team", "items": items})
		return
	case "run":
		agent := "claude"
		if len(words) > 1 {
			agent = words[1]
		}
		prompt, err := s.taskPrompt(r.Context(), task, "")
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		run, err := s.createTaskRun(r.Context(), task, agent, domain.RoleImplementer, req.Repo, req.Base, prompt)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"action": "run", "run": run})
		return
	case "note":
		if len(words) > 1 {
			text = strings.TrimSpace(strings.TrimPrefix(body, words[0]))
		}
	}

	note := &domain.TaskNote{
		ID:        "note_" + ulid.Make().String(),
		TaskID:    task.ID,
		Author:    author,
		Body:      text,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.AddTaskNote(r.Context(), note); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	s.recordTaskEvent(r.Context(), task.ID, "task.note", map[string]any{
		"noteId": note.ID, "body": note.Body, "author": note.Author,
	})
	writeJSON(w, http.StatusCreated, map[string]any{"action": "note", "note": note})
}

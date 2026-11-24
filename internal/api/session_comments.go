package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Server) listSessionComments(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	items, err := s.store.ListSessionComments(r.Context(), sess.ID, r.URL.Query().Get("status"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type createSessionCommentRequest struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Side string `json:"side"`
	Body string `json:"body"`
}

func (s *Server) createSessionComment(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	var req createSessionCommentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if strings.TrimSpace(req.File) == "" || strings.TrimSpace(req.Body) == "" || req.Line <= 0 {
		writeErr(w, http.StatusBadRequest, "E_COMMENT", "file, line and body are required")
		return
	}
	if req.Side != "old" {
		req.Side = "new"
	}
	comment, err := s.store.CreateSessionComment(r.Context(), &domain.SessionComment{
		SessionID: sess.ID,
		File:      req.File,
		Line:      req.Line,
		Side:      req.Side,
		Body:      strings.TrimSpace(req.Body),
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, comment)
}

func (s *Server) sendSessionComments(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	if sess.Status == "closed" {
		writeErr(w, http.StatusConflict, "E_SESSION_CLOSED", "session is closed")
		return
	}
	comments, err := s.store.ListSessionComments(r.Context(), sess.ID, "open")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if len(comments) == 0 {
		writeErr(w, http.StatusBadRequest, "E_COMMENT", "there are no open review comments")
		return
	}
	var prompt strings.Builder
	prompt.WriteString("Please address these review comments from the diff:\n\n")
	for _, comment := range comments {
		fmt.Fprintf(&prompt, "- %s:%d (%s)\n  %s\n", comment.File, comment.Line, comment.Side, comment.Body)
	}
	prompt.WriteString("\nUpdate the code and explain the changes.")
	text := prompt.String()

	run, err := s.enqueueReviewRun(r, sess, text)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if err := s.store.MarkSessionCommentsSent(r.Context(), sess.ID); err != nil {
		s.log.Warn("mark review comments sent", "session", sess.ID, "err", err)
	}
	writeJSON(w, http.StatusCreated, run)
}

func (s *Server) enqueueReviewRun(r *http.Request, sess *domain.AgentSession, prompt string) (*domain.Run, error) {
	agentID := sess.AgentID
	if agentID == "" {
		agentID = "claude"
	}
	run := &domain.Run{
		ID:          "run_" + ulid.Make().String(),
		WorkspaceID: sess.WorkspaceID,
		Work:        domain.WorkRef{Provider: "orca", Instance: "session", Type: "session", ID: sess.ID},
		Snapshot:    domain.Snapshot{Title: sess.Title},
		AgentID:     agentID,
		Status:      domain.RunQueued,
		Repo:        sess.Repo,
		Base:        sess.Base,
		Model:       sess.Model,
		Prompt:      prompt,
		TaskID:      sess.TaskID,
		Role:        "review",
		SessionID:   sess.ID,
		CreatedAt:   time.Now().UTC(),
	}
	if run.Base == "" {
		run.Base = "main"
	}
	if sess.WorktreeID != "" {
		if wt, err := s.store.GetWorktree(r.Context(), sess.WorktreeID); err == nil && wt != nil {
			run.Repo = wt.Repo
			run.Base = wt.BaseRef
			run.Branch = wt.Branch
		}
	}
	if run.Branch == "" && run.Repo != "" {
		short := sess.ID
		if len(short) > 8 {
			short = short[len(short)-8:]
		}
		run.Branch = "orca/session-" + short
	}
	if err := s.store.CreateRun(r.Context(), run); err != nil {
		return nil, err
	}
	_, _ = s.store.AppendSessionMessage(r.Context(), &domain.SessionMessage{
		SessionID: sess.ID, RunID: run.ID, Role: "user", Kind: "review", Text: prompt,
	})
	_, _ = s.store.AppendEvent(r.Context(), run.ID, domain.EventRunCreated, map[string]any{
		"sessionId": sess.ID, "role": "review",
	})
	_ = s.store.RecordSessionRun(r.Context(), sess.ID, "", run.ID)
	return run, nil
}

func (s *Server) resolveSessionComment(w http.ResponseWriter, r *http.Request) {
	comment, err := s.store.GetSessionComment(r.Context(), chi.URLParam(r, "id"))
	if err != nil || comment == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "comment not found")
		return
	}
	sess, err := s.store.GetAgentSession(r.Context(), comment.SessionID)
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	updated, err := s.store.ResolveSessionComment(r.Context(), comment.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if updated == nil {
		writeErr(w, http.StatusConflict, "E_STATE", "comment is not open")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

type updateSessionCommentRequest struct {
	Body string `json:"body"`
}

func (s *Server) updateSessionComment(w http.ResponseWriter, r *http.Request) {
	comment, sess, ok := s.sessionCommentContext(w, r)
	if !ok {
		return
	}
	_ = sess
	var req updateSessionCommentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		writeErr(w, http.StatusBadRequest, "E_COMMENT", "body is required")
		return
	}
	if updated, err := s.store.UpdateSessionComment(r.Context(), comment.ID, body); err != nil || !updated {
		writeErr(w, http.StatusInternalServerError, "E_DB", "could not update comment")
		return
	}
	comment.Body = body
	writeJSON(w, http.StatusOK, comment)
}

func (s *Server) deleteSessionComment(w http.ResponseWriter, r *http.Request) {
	comment, _, ok := s.sessionCommentContext(w, r)
	if !ok {
		return
	}
	if deleted, err := s.store.DeleteSessionComment(r.Context(), comment.ID); err != nil || !deleted {
		writeErr(w, http.StatusInternalServerError, "E_DB", "could not delete comment")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) sessionCommentContext(w http.ResponseWriter, r *http.Request) (*domain.SessionComment, *domain.AgentSession, bool) {
	comment, err := s.store.GetSessionComment(r.Context(), chi.URLParam(r, "id"))
	if err != nil || comment == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "comment not found")
		return nil, nil, false
	}
	sess, err := s.store.GetAgentSession(r.Context(), comment.SessionID)
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return nil, nil, false
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return nil, nil, false
	}
	return comment, sess, true
}

package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Server) getSessionWorktree(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	wt, reason := s.sessionWorktreeRecord(r, sess)
	if wt == nil {
		writeErr(w, http.StatusUnprocessableEntity, "E_WORKTREE", reason)
		return
	}
	cleanup, _ := s.store.LatestWorktreeCleanup(r.Context(), wt.ID)
	writeJSON(w, http.StatusOK, map[string]any{"worktree": wt, "cleanup": cleanup})
}

func (s *Server) archiveSessionWorktree(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	wt, reason := s.sessionWorktreeRecord(r, sess)
	if wt == nil {
		writeErr(w, http.StatusUnprocessableEntity, "E_WORKTREE", reason)
		return
	}
	updated, err := s.store.ArchiveWorktree(r.Context(), wt.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	s.enqueueWorktreeCleanup(r, wt)
	s.appendWorktreeMessage(r, sess, "Worktree archived: "+wt.Branch)
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) cleanupSessionWorktree(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	wt, reason := s.sessionWorktreeRecord(r, sess)
	if wt == nil {
		writeErr(w, http.StatusUnprocessableEntity, "E_WORKTREE", reason)
		return
	}
	run, _ := s.store.GetRun(r.Context(), wt.RunID)
	if run == nil || run.MachineID == "" {
		writeErr(w, http.StatusBadRequest, "E_MACHINE", "worktree has no execution machine")
		return
	}
	cleanup, err := s.store.EnqueueWorktreeCleanup(r.Context(), wt, run.MachineID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	s.appendWorktreeMessage(r, sess, "Worktree cleanup queued: "+wt.Branch)
	writeJSON(w, http.StatusAccepted, cleanup)
}

func (s *Server) appendWorktreeMessage(r *http.Request, sess *domain.AgentSession, text string) {
	_, _ = s.store.AppendSessionMessage(r.Context(), &domain.SessionMessage{
		SessionID: sess.ID, Role: "system", Kind: "worktree", Text: text,
	})
}

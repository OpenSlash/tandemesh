package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/gitops"
)

func (s *Server) sessionWorktreeRecord(r *http.Request, sess *domain.AgentSession) (*domain.Worktree, string) {
	if sess.WorktreeID == "" {
		return nil, "session has no worktree"
	}
	wt, err := s.store.GetWorktree(r.Context(), sess.WorktreeID)
	if err != nil || wt == nil {
		return nil, "session worktree not found"
	}
	return wt, ""
}

func localWorktreePath(wt *domain.Worktree) (string, bool) {
	path := strings.TrimSpace(wt.Path)
	if path == "" {
		return "", false
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", false
	}
	return path, true
}

func (s *Server) sessionWorktree(r *http.Request, sess *domain.AgentSession) (*domain.Worktree, string) {
	wt, reason := s.sessionWorktreeRecord(r, sess)
	if wt == nil {
		return nil, reason
	}
	if _, ok := localWorktreePath(wt); !ok {
		return nil, "worktree path is not available on this host"
	}
	return wt, ""
}

func (s *Server) getSessionGit(w http.ResponseWriter, r *http.Request) {
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
	path, local := localWorktreePath(wt)
	if !local {
		writeJSON(w, http.StatusOK, map[string]any{
			"worktreeId": wt.ID,
			"branch":     wt.Branch,
			"clean":      true,
			"changes":    []string{},
			"remote":     true,
		})
		return
	}
	status, err := gitops.New().Inspect(r.Context(), path)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "E_GIT", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"worktreeId": wt.ID,
		"branch":     status.Branch,
		"clean":      status.Clean,
		"changes":    status.Changes,
		"lastCommit": status.LastCommit,
		"commits":    status.Commits,
	})
}

func (s *Server) enqueueGitJob(r *http.Request, sess *domain.AgentSession, wt *domain.Worktree, action, message string) (*domain.GitJob, string) {
	run, err := s.store.GetRun(r.Context(), wt.RunID)
	if err != nil || run == nil || run.MachineID == "" {
		return nil, "worktree is not attached to an execution machine"
	}
	job, err := s.store.CreateGitJob(r.Context(), &domain.GitJob{
		SessionID:  sess.ID,
		WorktreeID: wt.ID,
		MachineID:  run.MachineID,
		Path:       wt.Path,
		Action:     action,
		Message:    message,
	})
	if err != nil {
		return nil, err.Error()
	}
	return job, ""
}

type shipSessionRequest struct {
	Message string `json:"message"`
}

func (s *Server) shipSession(w http.ResponseWriter, r *http.Request) {
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
	var req shipSessionRequest
	if r.ContentLength > 0 {
		if err := decodeJSON(w, r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
			return
		}
	}
	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = sess.Title
	}
	path, local := localWorktreePath(wt)
	if !local {
		job, reason := s.enqueueGitJob(r, sess, wt, "ship", message)
		if job == nil {
			writeErr(w, http.StatusUnprocessableEntity, "E_GIT", reason)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status": "queued", "jobId": job.ID, "machineId": job.MachineID,
		})
		return
	}
	status, err := gitops.New().Ship(r.Context(), path, message)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "E_GIT", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"branch":     status.Branch,
		"clean":      status.Clean,
		"lastCommit": status.LastCommit,
	})
}

func (s *Server) createSessionPR(w http.ResponseWriter, r *http.Request) {
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
	path, local := localWorktreePath(wt)
	if !local {
		job, reason := s.enqueueGitJob(r, sess, wt, "pr", "")
		if job == nil {
			writeErr(w, http.StatusUnprocessableEntity, "E_PR", reason)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status": "queued", "jobId": job.ID, "machineId": job.MachineID,
		})
		return
	}
	url, err := gitops.New().CreatePR(r.Context(), path)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "E_PR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

func (s *Server) getSessionGitJob(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	job, err := s.store.GetGitJob(r.Context(), chi.URLParam(r, "jobId"))
	if err != nil || job == nil || job.SessionID != sess.ID {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "git job not found")
		return
	}
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) agentdGitJobs(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.PendingGitJobs(r.Context(), r.URL.Query().Get("machineId"), 20)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type gitJobResult struct {
	Status string          `json:"status"`
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error"`
}

func (s *Server) agentdGitJobResult(w http.ResponseWriter, r *http.Request) {
	var req gitJobResult
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.Status == "" {
		req.Status = "succeeded"
	}
	result := ""
	if len(req.Result) > 0 {
		result = string(req.Result)
	}
	if err := s.store.FinishGitJob(r.Context(), chi.URLParam(r, "id"), req.Status, result, req.Error); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": req.Status})
}

func (s *Server) getSessionPRStatus(w http.ResponseWriter, r *http.Request) {
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
	path, local := localWorktreePath(wt)
	if !local {
		writeErr(w, http.StatusUnprocessableEntity, "E_REMOTE", "PR status is only available on the worktree host")
		return
	}
	state, err := gitops.New().PRStatus(r.Context(), path)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "E_PR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, state)
}

func (s *Server) refreshSessionPR(w http.ResponseWriter, r *http.Request) {
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
	path, local := localWorktreePath(wt)
	if !local {
		job, reason := s.enqueueGitJob(r, sess, wt, "pr_status", "")
		if job == nil {
			writeErr(w, http.StatusUnprocessableEntity, "E_PR", reason)
			return
		}
		writeJSON(w, http.StatusAccepted, map[string]any{"status": "queued", "jobId": job.ID, "machineId": job.MachineID})
		return
	}
	state, err := gitops.New().PRStatus(r.Context(), path)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, "E_PR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, state)
}

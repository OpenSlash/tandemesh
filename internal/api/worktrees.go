package api

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Server) listWorktrees(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspaceId")
	if workspaceID != "" && !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	items, err := s.store.ListWorktrees(r.Context(), workspaceID, r.URL.Query().Get("taskId"), r.URL.Query().Get("repo"), r.URL.Query().Get("status"), 200)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"worktrees": items})
}

func (s *Server) getWorktree(w http.ResponseWriter, r *http.Request) {
	wt, err := s.store.GetWorktree(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if wt == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "worktree not found")
		return
	}
	if !s.requireWorkspace(w, r, wt.WorkspaceID) {
		return
	}
	writeJSON(w, http.StatusOK, wt)
}

func (s *Server) getRunWorktree(w http.ResponseWriter, r *http.Request) {
	wt, err := s.store.GetWorktreeByRun(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if wt == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "worktree not found for run")
		return
	}
	if !s.requireWorkspace(w, r, wt.WorkspaceID) {
		return
	}
	writeJSON(w, http.StatusOK, wt)
}

type checkpointRequest struct {
	Comment         string `json:"comment"`
	WorkspaceStatus string `json:"workspaceStatus"`
}

func (s *Server) setWorktreeCheckpoint(w http.ResponseWriter, r *http.Request) {
	wt, err := s.store.GetWorktree(r.Context(), chi.URLParam(r, "id"))
	if err != nil || wt == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "worktree not found")
		return
	}
	s.applyWorktreeCheckpoint(w, r, wt)
}

func (s *Server) checkpointRunWorktree(w http.ResponseWriter, r *http.Request) {
	wt, err := s.store.GetWorktreeByRun(r.Context(), chi.URLParam(r, "id"))
	if err != nil || wt == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "worktree not found for run")
		return
	}
	s.applyWorktreeCheckpoint(w, r, wt)
}

func (s *Server) applyWorktreeCheckpoint(w http.ResponseWriter, r *http.Request, wt *domain.Worktree) {
	if !s.requireWorkspace(w, r, wt.WorkspaceID) {
		return
	}
	var req checkpointRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	updated, err := s.store.SetWorktreeCheckpoint(r.Context(), wt.ID, req.Comment, req.WorkspaceStatus)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	s.recordTaskEvent(r.Context(), wt.TaskID, "task.worktree.checkpoint", map[string]any{
		"worktreeId": wt.ID, "comment": req.Comment, "status": req.WorkspaceStatus,
	})
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) archiveWorktree(w http.ResponseWriter, r *http.Request) {
	wt, err := s.store.GetWorktree(r.Context(), chi.URLParam(r, "id"))
	if err != nil || wt == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "worktree not found")
		return
	}
	if !s.requireWorkspace(w, r, wt.WorkspaceID) {
		return
	}
	updated, err := s.store.ArchiveWorktree(r.Context(), wt.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	s.enqueueWorktreeCleanup(r, wt)
	s.recordTaskEvent(r.Context(), wt.TaskID, "task.worktree.archived", map[string]any{"worktreeId": wt.ID})
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) enqueueWorktreeCleanup(r *http.Request, wt *domain.Worktree) {
	run, _ := s.store.GetRun(r.Context(), wt.RunID)
	machineID := ""
	if run != nil {
		machineID = run.MachineID
	}
	if machineID == "" {
		return
	}
	if _, err := s.store.EnqueueWorktreeCleanup(r.Context(), wt, machineID); err != nil {
		s.log.Warn("enqueue worktree cleanup", "worktree", wt.ID, "err", err)
	}
}

func (s *Server) cleanupWorktree(w http.ResponseWriter, r *http.Request) {
	wt, err := s.store.GetWorktree(r.Context(), chi.URLParam(r, "id"))
	if err != nil || wt == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "worktree not found")
		return
	}
	if !s.requireWorkspace(w, r, wt.WorkspaceID) {
		return
	}
	run, _ := s.store.GetRun(r.Context(), wt.RunID)
	if run == nil || run.MachineID == "" {
		writeErr(w, http.StatusBadRequest, "E_MACHINE", "worktree has no execution machine")
		return
	}
	c, err := s.store.EnqueueWorktreeCleanup(r.Context(), wt, run.MachineID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, c)
}

type retentionRequest struct {
	WorkspaceID   string `json:"workspaceId"`
	OlderThanDays int    `json:"olderThanDays"`
}

func (s *Server) worktreeRetention(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspaceId")
	if workspaceID != "" && !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	days := 7
	if v := r.URL.Query().Get("olderThanDays"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			days = n
		}
	}
	cutoff := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour)
	items, err := s.store.ExpiredWorktrees(r.Context(), workspaceID, cutoff, 500)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"worktrees": items, "olderThanDays": days})
}

func (s *Server) applyWorktreeRetention(w http.ResponseWriter, r *http.Request) {
	var req retentionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.WorkspaceID != "" && !s.requireWorkspace(w, r, req.WorkspaceID) {
		return
	}
	if req.OlderThanDays <= 0 {
		req.OlderThanDays = 7
	}
	cutoff := time.Now().UTC().Add(-time.Duration(req.OlderThanDays) * 24 * time.Hour)
	items, err := s.store.ExpiredWorktrees(r.Context(), req.WorkspaceID, cutoff, 1000)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	archived, queued := 0, 0
	for _, wt := range items {
		if _, err := s.store.ArchiveWorktree(r.Context(), wt.ID); err != nil {
			continue
		}
		archived++
		run, _ := s.store.GetRun(r.Context(), wt.RunID)
		if run != nil && run.MachineID != "" {
			if _, err := s.store.EnqueueWorktreeCleanup(r.Context(), wt, run.MachineID); err == nil {
				queued++
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]int{"archived": archived, "queued": queued})
}

func (s *Server) agentdWorktreeCleanups(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.PendingWorktreeCleanups(r.Context(), r.URL.Query().Get("machineId"), 50)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if items == nil {
		items = []*domain.WorktreeCleanup{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type worktreeCleanupResult struct {
	Status string `json:"status"`
	Error  string `json:"error"`
}

func (s *Server) agentdWorktreeCleanupResult(w http.ResponseWriter, r *http.Request) {
	var req worktreeCleanupResult
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	status := req.Status
	if status != "done" && status != "failed" {
		writeErr(w, http.StatusBadRequest, "E_STATUS", "status must be done or failed")
		return
	}
	id := chi.URLParam(r, "id")
	if err := s.store.FinishWorktreeCleanup(r.Context(), id, status, req.Error); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if status == "done" {
		if c, _ := s.store.GetWorktreeCleanup(r.Context(), id); c != nil {
			_ = s.store.SetWorktreeStatus(r.Context(), c.WorktreeID, "removed")
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": status})
}

type repositorySummary struct {
	Repo          string `json:"repo"`
	BaseRef       string `json:"baseRef,omitempty"`
	Branch        string `json:"branch,omitempty"`
	Path          string `json:"path,omitempty"`
	WorktreeCount int    `json:"worktreeCount"`
}

func (s *Server) listRepositories(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspaceId")
	if workspaceID != "" && !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	items, err := s.store.ListWorktrees(r.Context(), workspaceID, "", "", "", 200)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	byRepo := map[string]*repositorySummary{}
	for _, wt := range items {
		if wt.ArchivedAt != nil || strings.TrimSpace(wt.Repo) == "" {
			continue
		}
		entry := byRepo[wt.Repo]
		if entry == nil {
			entry = &repositorySummary{Repo: wt.Repo, BaseRef: wt.BaseRef, Branch: wt.Branch, Path: wt.Path}
			byRepo[wt.Repo] = entry
		}
		entry.WorktreeCount++
	}
	out := make([]*repositorySummary, 0, len(byRepo))
	for _, entry := range byRepo {
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].WorktreeCount > out[j].WorktreeCount })
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

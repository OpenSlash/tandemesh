package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
)

type createSessionRequest struct {
	WorkspaceID string `json:"workspaceId"`
	TaskID      string `json:"taskId"`
	AgentID     string `json:"agentId"`
	Backend     string `json:"backend"`
	Model       string `json:"model"`
	Reasoning   string `json:"reasoning"`
	Mode        string `json:"mode"`
	Title       string `json:"title"`
	Repo        string `json:"repo"`
	Base        string `json:"base"`
	WorktreeID  string `json:"worktreeId"`
}

func (s *Server) listSessions(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspaceId")
	status := r.URL.Query().Get("status")
	taskID := r.URL.Query().Get("taskId")
	if workspaceID == "all" {
		workspaces, err := s.store.ListWorkspaces(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		items := []*domain.AgentSession{}
		for _, workspace := range workspaces {
			if !s.allowWorkspace(r.Context(), workspace.ID) {
				continue
			}
			batch, err := s.store.ListAgentSessions(r.Context(), workspace.ID, taskID, status, 200)
			if err != nil {
				writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
				return
			}
			items = append(items, batch...)
		}
		sort.SliceStable(items, func(i, j int) bool { return items[i].UpdatedAt.After(items[j].UpdatedAt) })
		if len(items) > 200 {
			items = items[:200]
		}
		writeJSON(w, http.StatusOK, map[string]any{"sessions": items})
		return
	}
	if workspaceID != "" && !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	items, err := s.store.ListAgentSessions(r.Context(), workspaceID, taskID, status, 200)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": items})
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var req createSessionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.WorkspaceID == "" {
		req.WorkspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, req.WorkspaceID) {
		return
	}
	if req.Mode == "" {
		req.Mode = "chat"
	}
	if req.Mode != "chat" && req.Mode != "terminal" {
		writeErr(w, http.StatusBadRequest, "E_MODE", "mode must be chat or terminal")
		return
	}
	req.Reasoning = strings.ToLower(strings.TrimSpace(req.Reasoning))
	if req.Reasoning != "" && req.Reasoning != "low" && req.Reasoning != "medium" && req.Reasoning != "high" {
		writeErr(w, http.StatusBadRequest, "E_REASONING", "reasoning must be low, medium, or high")
		return
	}
	if req.AgentID == "" {
		req.AgentID = "claude"
	}
	var task *domain.Task
	if req.TaskID != "" {
		task, _ = s.store.GetTask(r.Context(), req.TaskID)
		if task == nil {
			writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
			return
		}
		if req.WorkspaceID == "ws_default" {
			req.WorkspaceID = task.WorkspaceID
		}
		if req.Title == "" {
			req.Title = task.Title
		}
		if task.AssigneeAgentID != "" && req.AgentID == "claude" {
			req.AgentID = task.AssigneeAgentID
		}
		if task.WorkspaceID != req.WorkspaceID {
			writeErr(w, http.StatusBadRequest, "E_WORKSPACE", "task is not in this workspace")
			return
		}
	}
	if req.WorktreeID != "" {
		wt, err := s.store.GetWorktree(r.Context(), req.WorktreeID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		if wt == nil || wt.WorkspaceID != req.WorkspaceID || wt.ArchivedAt != nil {
			writeErr(w, http.StatusBadRequest, "E_WORKTREE", "worktree is not available in this workspace")
			return
		}
		req.Repo = wt.Repo
		if req.Base == "" {
			req.Base = wt.BaseRef
		}
	}
	if req.AgentID != "" {
		agent, err := s.store.GetAgent(r.Context(), req.AgentID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		if agent != nil {
			if !s.agentInWorkspace(agent, req.WorkspaceID) {
				writeErr(w, http.StatusBadRequest, "E_AGENT", "agent is not in this workspace")
				return
			}
			if req.Backend == "" {
				req.Backend = agent.Backend
			}
		}
	}
	if req.Title == "" {
		req.Title = "Agent session"
	}
	sess, err := s.store.CreateAgentSession(r.Context(), &domain.AgentSession{
		WorkspaceID: req.WorkspaceID,
		TaskID:      req.TaskID,
		AgentID:     req.AgentID,
		Backend:     req.Backend,
		Model:       req.Model,
		Reasoning:   req.Reasoning,
		CreatedBy:   userOf(r),
		Mode:        req.Mode,
		Title:       req.Title,
		Repo:        req.Repo,
		Base:        req.Base,
		WorktreeID:  req.WorktreeID,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if req.TaskID != "" {
		s.recordTaskEvent(r.Context(), req.TaskID, "task.session.created", map[string]any{
			"sessionId": sess.ID, "mode": sess.Mode, "agentId": sess.AgentID,
		})
	}
	writeJSON(w, http.StatusCreated, sess)
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	count, _ := s.store.CountSessionRuns(r.Context(), sess.ID)
	writeJSON(w, http.StatusOK, map[string]any{"session": sess, "runCount": count})
}

func (s *Server) closeSession(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	closed, err := s.store.SetAgentSessionStatus(r.Context(), sess.ID, "closed")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, closed)
}

func (s *Server) listSessionRuns(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	items, err := s.store.ListRunsBySession(r.Context(), sess.ID, 200)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type sessionRunRequest struct {
	Prompt          string `json:"prompt"`
	Repo            string `json:"repo"`
	Base            string `json:"base"`
	AgentID         string `json:"agentId"`
	SkipUserMessage bool   `json:"skipUserMessage"`
}

func (s *Server) createSessionRun(w http.ResponseWriter, r *http.Request) {
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
	var req sessionRunRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		writeErr(w, http.StatusBadRequest, "E_PROMPT", "prompt is required")
		return
	}
	agentID := req.AgentID
	if agentID == "" {
		agentID = sess.AgentID
	}
	if agentID == "" {
		agentID = "claude"
	}
	id := "run_" + ulid.Make().String()
	prompt := req.Prompt
	if sess.Context != "" {
		prompt = sess.Context + "\n\n" + prompt
	}
	work := domain.WorkRef{Provider: "orca", Instance: "session", Type: "session", ID: sess.ID}
	run := &domain.Run{
		ID:          id,
		WorkspaceID: sess.WorkspaceID,
		Work:        work,
		Snapshot:    domain.Snapshot{Title: sess.Title},
		AgentID:     agentID,
		Status:      domain.RunQueued,
		Repo:        req.Repo,
		Base:        req.Base,
		Model:       sess.Model,
		Prompt:      prompt,
		TaskID:      sess.TaskID,
		Role:        "chat",
		SessionID:   sess.ID,
		CreatedAt:   time.Now().UTC(),
	}
	if run.Repo == "" {
		run.Repo = sess.Repo
	}
	if run.Base == "" {
		run.Base = sess.Base
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
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if sess.Context != "" {
		_ = s.store.ClearAgentSessionContext(r.Context(), sess.ID)
	}
	if sess.WorktreeID == "" && run.Repo != "" {
		wt, err := s.store.CreateWorktree(r.Context(), &domain.Worktree{
			WorkspaceID: sess.WorkspaceID,
			TaskID:      sess.TaskID,
			RunID:       run.ID,
			Repo:        run.Repo,
			Branch:      run.Branch,
			BaseRef:     run.Base,
			StartRef:    run.Base,
			Status:      "creating",
		})
		if err != nil {
			s.log.Warn("create session worktree", "session", sess.ID, "err", err)
		} else if err := s.store.SetAgentSessionWorktree(r.Context(), sess.ID, wt.ID); err != nil {
			s.log.Warn("bind session worktree", "session", sess.ID, "err", err)
		}
	}
	if !req.SkipUserMessage {
		_, _ = s.store.AppendSessionMessage(r.Context(), &domain.SessionMessage{
			SessionID: sess.ID, RunID: run.ID, Role: "user", Kind: "text", Text: req.Prompt,
		})
	}
	if _, err := s.store.AppendEvent(r.Context(), run.ID, domain.EventRunCreated, map[string]any{
		"sessionId": sess.ID,
	}); err != nil {
		s.log.Warn("append session run event", "run", run.ID, "err", err)
	}
	if err := s.store.RecordSessionRun(r.Context(), sess.ID, "", run.ID); err != nil {
		s.log.Warn("record session run", "session", sess.ID, "err", err)
	}
	writeJSON(w, http.StatusCreated, run)
}

type sessionMessage struct {
	ID    string `json:"id,omitempty"`
	RunID string `json:"runId"`
	Role  string `json:"role"`
	Kind  string `json:"kind"`
	Text  string `json:"text"`
	TS    int64  `json:"ts"`
}

func (s *Server) recordSessionEvent(ctx context.Context, runID string, ev *domain.Event) {
	if ev == nil {
		return
	}
	if ev.Type == "agent.session.started" {
		backendID, _ := ev.Payload["sessionId"].(string)
		if backendID != "" {
			if run, err := s.store.GetRun(ctx, runID); err == nil && run != nil && run.SessionID != "" {
				if err := s.store.RecordSessionRun(ctx, run.SessionID, backendID, runID); err != nil {
					s.log.Warn("record backend session", "run", runID, "err", err)
				}
			}
		}
		return
	}
	if ev.Type == "agent.message.complete" {
		// Streaming chunks are already persisted; this is the final aggregate.
		return
	}
	run, err := s.store.GetRun(ctx, runID)
	if err != nil || run == nil || run.SessionID == "" {
		return
	}
	block := aggregateEvent(*ev)
	if block == nil || strings.TrimSpace(block.Text) == "" {
		return
	}
	role := "assistant"
	if block.Kind == "status" || block.Kind == "result" {
		role = "system"
	}
	if _, err := s.store.AppendSessionMessage(ctx, &domain.SessionMessage{
		SessionID: run.SessionID, RunID: runID, Role: role, Kind: block.Kind, Text: block.Text,
	}); err != nil {
		s.log.Warn("append session message", "run", runID, "err", err)
	}
}

func (s *Server) sessionDiff(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	runs, err := s.store.ListRunsBySession(r.Context(), sess.ID, 100)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if len(runs) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"diff": "", "truncated": false})
		return
	}
	for ri := len(runs) - 1; ri >= 0; ri-- {
		run := runs[ri]
		events, err := s.store.ListEvents(r.Context(), run.ID, 0)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		for i := len(events) - 1; i >= 0; i-- {
			if events[i].Type != "workspace.diff" {
				continue
			}
			diff, _ := events[i].Payload["diff"].(string)
			truncated, _ := events[i].Payload["truncated"].(bool)
			writeJSON(w, http.StatusOK, map[string]any{
				"runId": run.ID, "diff": diff, "truncated": truncated,
			})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"diff": "", "truncated": false})
}

func (s *Server) streamSessionEvents(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeErr(w, http.StatusInternalServerError, "E_STREAM", "streaming unsupported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	after := r.URL.Query().Get("after")
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.NewTimer(10 * time.Minute)
	defer deadline.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-deadline.C:
			fmt.Fprint(w, "event: timeout\ndata: {}\n\n")
			flusher.Flush()
			return
		case <-ticker.C:
			messages, err := s.store.ListSessionMessagesAfter(r.Context(), sess.ID, after, 100)
			if err != nil {
				return
			}
			for _, msg := range messages {
				raw, _ := json.Marshal(sessionMessage{
					ID: msg.ID, RunID: msg.RunID, Role: msg.Role, Kind: msg.Kind, Text: msg.Text, TS: msg.CreatedAt.UnixMilli(),
				})
				fmt.Fprintf(w, "id: %s\ndata: %s\n\n", msg.ID, raw)
				after = msg.ID
			}
			if len(messages) > 0 {
				flusher.Flush()
			}
			if sess.Status == "closed" && len(messages) == 0 {
				fmt.Fprint(w, "event: end\ndata: {}\n\n")
				flusher.Flush()
				return
			}
		}
	}
}

func (s *Server) sessionMessages(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	stored, err := s.store.ListSessionMessages(r.Context(), sess.ID, 500)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if len(stored) > 0 {
		out := make([]sessionMessage, 0, len(stored))
		for _, msg := range stored {
			out = append(out, sessionMessage{
				ID: msg.ID, RunID: msg.RunID, Role: msg.Role, Kind: msg.Kind, Text: msg.Text, TS: msg.CreatedAt.UnixMilli(),
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"messages": out})
		return
	}
	runs, err := s.store.ListRunsBySession(r.Context(), sess.ID, 200)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	out := make([]sessionMessage, 0)
	for _, run := range runs {
		if run.Prompt != "" {
			out = append(out, sessionMessage{RunID: run.ID, Role: "user", Kind: "text", Text: run.Prompt, TS: run.CreatedAt.UnixMilli()})
		}
		events, err := s.store.ListEvents(r.Context(), run.ID, 0)
		if err != nil {
			continue
		}
		for _, ev := range events {
			block := aggregateEvent(ev)
			if block == nil {
				continue
			}
			role := "assistant"
			if block.Kind == "status" || block.Kind == "result" {
				role = "system"
			}
			if block.Text == "" {
				continue
			}
			out = append(out, sessionMessage{RunID: run.ID, Role: role, Kind: block.Kind, Text: block.Text, TS: block.TS})
		}
		if run.ErrorMessage != "" {
			out = append(out, sessionMessage{RunID: run.ID, Role: "system", Kind: "error", Text: fmt.Sprintf("%s: %s", run.ErrorCode, run.ErrorMessage), TS: time.Now().UnixMilli()})
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": out})
}
func (s *Server) recordAgentQuestion(ctx context.Context, runID string, ev *domain.Event) {
	if ev == nil || ev.Type != "agent.question" {
		return
	}
	run, err := s.store.GetRun(ctx, runID)
	if err != nil || run == nil || run.SessionID == "" {
		return
	}
	text, _ := ev.Payload["question"].(string)
	if strings.TrimSpace(text) == "" {
		return
	}
	options := []string{}
	if raw, ok := ev.Payload["options"].([]any); ok {
		for _, value := range raw {
			if option, ok := value.(string); ok && strings.TrimSpace(option) != "" {
				options = append(options, option)
			}
		}
	}
	id, _ := ev.Payload["id"].(string)
	if _, err := s.store.CreateAgentQuestion(ctx, &domain.AgentQuestion{
		ID: id, SessionID: run.SessionID, RunID: run.ID, Question: text, Options: options,
	}); err != nil {
		s.log.Warn("create agent question", "run", runID, "err", err)
	}
}

type regenerateSessionRequest struct {
	RunID string `json:"runId"`
}

// regenerateSession queues a new run from the last prompt without adding a
// duplicate user message to the transcript.
func (s *Server) regenerateSession(w http.ResponseWriter, r *http.Request) {
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
	var req regenerateSessionRequest
	if r.ContentLength > 0 {
		if err := decodeJSON(w, r, &req); err != nil {
			writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
			return
		}
	}
	runID := req.RunID
	if runID == "" {
		runID = sess.LastRunID
	}
	if runID == "" {
		writeErr(w, http.StatusConflict, "E_STATE", "session has no runs")
		return
	}
	orig, err := s.store.GetRun(r.Context(), runID)
	if err != nil || orig == nil || orig.SessionID != sess.ID {
		writeErr(w, http.StatusNotFound, "E_RUN", "run not found in this session")
		return
	}
	if orig.Status == domain.RunQueued || orig.Status == domain.RunLeased || orig.Status == domain.RunRunning || orig.Status == domain.RunPreparing || orig.Status == domain.RunFinalizing {
		writeErr(w, http.StatusConflict, "E_STATE", "run is still active")
		return
	}
	clone := *orig
	clone.ID = "run_" + ulid.Make().String()
	clone.Status = domain.RunQueued
	clone.MachineID = ""
	clone.Fence = 0
	clone.Attempt = 0
	clone.ErrorCode = ""
	clone.ErrorMessage = ""
	clone.CreatedAt = time.Now().UTC()
	clone.StartedAt = nil
	clone.EndedAt = nil
	clone.LeaseExpiresAt = nil
	clone.VariantOf = orig.ID
	clone.VariantLabel = "regenerate"
	if err := s.store.CreateRun(r.Context(), &clone); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	_, _ = s.store.AppendEvent(r.Context(), clone.ID, domain.EventRunCreated, map[string]any{
		"sessionId": sess.ID, "regenerateOf": orig.ID,
	})
	_, _ = s.store.AppendSessionMessage(r.Context(), &domain.SessionMessage{
		SessionID: sess.ID, RunID: clone.ID, Role: "system", Kind: "regenerate", Text: "Regenerated response",
	})
	if err := s.store.RecordSessionRun(r.Context(), sess.ID, "", clone.ID); err != nil {
		s.log.Warn("record regenerated run", "session", sess.ID, "err", err)
	}
	writeJSON(w, http.StatusCreated, &clone)
}

// stopSession cancels every active or queued run in a session.
func (s *Server) stopSession(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	runs, err := s.store.ListRunsBySession(r.Context(), sess.ID, 100)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	stopped := 0
	for _, run := range runs {
		switch run.Status {
		case domain.RunQueued, domain.RunLeased, domain.RunPreparing, domain.RunRunning, domain.RunWaitingApproval, domain.RunFinalizing:
		default:
			continue
		}
		ok, err := s.store.CancelRun(r.Context(), run.ID)
		if err != nil || !ok {
			continue
		}
		if run.MachineID != "" {
			s.hub.Send(run.MachineID, hub.Message{Type: "run.cancel", RunID: run.ID, Fence: run.Fence})
		}
		_, _ = s.store.AppendEvent(r.Context(), run.ID, "run.cancelled", map[string]any{"by": "session-stop"})
		stopped++
	}
	if stopped > 0 {
		_, _ = s.store.AppendSessionMessage(r.Context(), &domain.SessionMessage{
			SessionID: sess.ID, Role: "system", Kind: "stop", Text: "Stopped run(s): " + strconv.Itoa(stopped),
		})
	}
	writeJSON(w, http.StatusOK, map[string]int{"stopped": stopped})
}

type renameSessionRequest struct {
	Title string `json:"title"`
}

func (s *Server) renameSession(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	var req renameSessionRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		writeErr(w, http.StatusBadRequest, "E_TITLE", "title is required")
		return
	}
	updated, err := s.store.RenameAgentSession(r.Context(), sess.ID, title)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) archiveSession(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	updated, err := s.store.ArchiveAgentSession(r.Context(), sess.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

type editSessionMessageRequest struct {
	Text string `json:"text"`
}

func (s *Server) editSessionMessage(w http.ResponseWriter, r *http.Request) {
	sess, err := s.store.GetAgentSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil || sess == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "session not found")
		return
	}
	if !s.requireWorkspace(w, r, sess.WorkspaceID) {
		return
	}
	var req editSessionMessageRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		writeErr(w, http.StatusBadRequest, "E_TEXT", "text is required")
		return
	}
	ok, err := s.store.UpdateSessionMessageText(r.Context(), sess.ID, chi.URLParam(r, "messageId"), text)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusNotFound, "E_MESSAGE", "user message not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

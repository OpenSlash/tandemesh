package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	workspaceID := r.URL.Query().Get("workspace")
	all := workspaceID == "all"
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !all && !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	items, err := s.store.ListTasks(r.Context(), store.TaskFilter{
		Workspace: func() string {
			if all {
				return ""
			}
			return workspaceID
		}(),
		Status:     r.URL.Query().Get("status"),
		Provider:   r.URL.Query().Get("provider"),
		Label:      r.URL.Query().Get("label"),
		Agent:      r.URL.Query().Get("agent"),
		Assignee:   r.URL.Query().Get("assignee"),
		Iteration:  r.URL.Query().Get("iteration"),
		Unassigned: r.URL.Query().Get("unassigned") == "1",
		Limit:      limit,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	s.enrichTaskAssignees(r.Context(), items)
	s.enrichTaskFields(r.Context(), items)
	if all {
		filtered := make([]*domain.Task, 0, len(items))
		for _, item := range items {
			if s.allowWorkspace(r.Context(), item.WorkspaceID) {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if err := s.applyTaskInbox(r.Context(), s.actingUser(r), items); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if inbox := r.URL.Query().Get("inbox"); inbox != "" {
		filtered := make([]*domain.Task, 0, len(items))
		for _, item := range items {
			if item.InboxState == inbox {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	if items == nil {
		items = []*domain.Task{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) setTaskAssignee(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireWorkspace(w, r, task.WorkspaceID) {
		return
	}
	var req struct {
		AgentID string `json:"agentId"`
		GroupID string `json:"groupId"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.GroupID != "" {
		group, err := s.store.GetGroup(r.Context(), req.GroupID)
		if err != nil || group == nil || group.WorkspaceID != task.WorkspaceID {
			writeErr(w, http.StatusBadRequest, "E_GROUP", "group not found in this workspace")
			return
		}
		if err := s.store.SetTaskAssigneeGroup(r.Context(), task.ID, req.GroupID); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		task.AssigneeGroupID = req.GroupID
		task.AssigneeAgentID = ""
		writeJSON(w, http.StatusOK, task)
		return
	}
	if req.AgentID != "" {
		agent, err := s.store.GetAgent(r.Context(), req.AgentID)
		if err != nil || agent == nil || !s.agentInWorkspace(agent, task.WorkspaceID) {
			writeErr(w, http.StatusBadRequest, "E_AGENT", "agent not found in this workspace")
			return
		}
	}
	if err := s.store.SetTaskAssignee(r.Context(), task.ID, req.AgentID); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	task.AssigneeAgentID = req.AgentID
	task.AssigneeGroupID = ""
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
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
	s.enrichTaskAssignees(r.Context(), []*domain.Task{task})
	s.enrichTaskFields(r.Context(), []*domain.Task{task})
	if err := s.applyTaskInbox(r.Context(), s.actingUser(r), []*domain.Task{task}); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	runs, err := s.store.ListRunsByTask(r.Context(), task.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if runs == nil {
		runs = []*domain.Run{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": task, "runs": runs})
}

type taskRunRequest struct {
	AgentID string `json:"agentId"`
	Repo    string `json:"repo"`
	Base    string `json:"base"`
	Role    string `json:"role"`
	Prompt  string `json:"prompt"`
}

// startTaskRun assigns another agent to a task. Runs are attempts, not the
// primary object: several agents can work the same task.
func (s *Server) startTaskRun(w http.ResponseWriter, r *http.Request) {
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
	if task.ActiveRuns > 0 {
		writeErr(w, http.StatusConflict, "E_ACTIVE", "task is already running")
		return
	}
	if task.ExecutionState == domain.TaskExecutionPending || task.ExecutionState == domain.TaskExecutionBlocked {
		writeErr(w, http.StatusConflict, "E_NOT_READY", "task is waiting on dependencies or a decision gate")
		return
	}
	if failures, _ := s.store.ConsecutiveDispatchFailures(r.Context(), task.ID); failures >= 3 {
		writeErr(w, http.StatusConflict, "E_CIRCUIT_BROKEN", "task failed three consecutive dispatches; clear it before retrying")
		return
	}
	var req taskRunRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.AgentID == "" {
		req.AgentID = "claude"
	}
	if req.Base == "" {
		req.Base = "main"
	}
	switch req.Role {
	case "", domain.RoleLead, domain.RoleImplementer, domain.RoleReviewer:
	default:
		writeErr(w, http.StatusBadRequest, "E_ROLE", "role must be lead, implementer or reviewer")
		return
	}
	prompt, err := s.taskPrompt(r.Context(), task, req.Prompt)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	run, err := s.createTaskRun(r.Context(), task, req.AgentID, req.Role, req.Repo, req.Base, prompt)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

type teamRoleRequest struct {
	Role    string `json:"role"`
	AgentID string `json:"agentId"`
}

type teamRequest struct {
	Repo    string            `json:"repo"`
	Base    string            `json:"base"`
	GroupID string            `json:"groupId"`
	Roles   []teamRoleRequest `json:"roles"`
}

// startTaskTeam assigns a standard crew to one task.
func (s *Server) startTaskTeam(w http.ResponseWriter, r *http.Request) {
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
	if task.ExecutionState == domain.TaskExecutionPending || task.ExecutionState == domain.TaskExecutionBlocked {
		writeErr(w, http.StatusConflict, "E_NOT_READY", "task is waiting on dependencies or a decision gate")
		return
	}
	if failures, _ := s.store.ConsecutiveDispatchFailures(r.Context(), task.ID); failures >= 3 {
		writeErr(w, http.StatusConflict, "E_CIRCUIT_BROKEN", "task failed three consecutive dispatches; clear it before retrying")
		return
	}
	var req teamRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if len(req.Roles) == 0 && req.GroupID == "" {
		writeErr(w, http.StatusBadRequest, "E_ROLES", "roles or groupId is required")
		return
	}
	if req.GroupID != "" {
		group, err := s.store.GetGroup(r.Context(), req.GroupID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		if group == nil || group.WorkspaceID != task.WorkspaceID {
			writeErr(w, http.StatusBadRequest, "E_GROUP", "group is not in this workspace")
			return
		}
		members, err := s.store.ListGroupMembers(r.Context(), group.ID)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		req.Roles = make([]teamRoleRequest, 0, len(members))
		for _, member := range members {
			role := member.Role
			if role == "" {
				role = domain.RoleImplementer
			}
			req.Roles = append(req.Roles, teamRoleRequest{Role: role, AgentID: member.AgentID})
		}
	}
	items, err := s.launchTeam(r.Context(), task, req.Repo, req.Base, req.Roles)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "E_TEAM", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"items": items})
}

// launchTeam creates one run per role, sharing the task prompt.
func (s *Server) launchTeam(ctx context.Context, task *domain.Task, repo, base string, roles []teamRoleRequest) ([]*domain.Run, error) {
	if len(roles) == 0 {
		return nil, fmt.Errorf("roles is required")
	}
	if base == "" {
		base = "main"
	}
	prompt, err := s.taskPrompt(ctx, task, "")
	if err != nil {
		return nil, err
	}
	items := make([]*domain.Run, 0, len(roles))
	for _, role := range roles {
		switch role.Role {
		case domain.RoleLead, domain.RoleImplementer, domain.RoleReviewer:
		default:
			return nil, fmt.Errorf("role must be lead, implementer or reviewer")
		}
		agent := role.AgentID
		if agent == "" {
			agent = "claude"
		}
		run, err := s.createTaskRun(ctx, task, agent, role.Role, repo, base, prompt)
		if err != nil {
			return nil, err
		}
		items = append(items, run)
	}
	return items, nil
}

// taskPrompt combines the task text with the team notes.
func (s *Server) taskPrompt(ctx context.Context, task *domain.Task, override string) (string, error) {
	prompt := task.Title
	if task.Body != "" {
		prompt += "\n\n" + task.Body
	}
	if override != "" {
		prompt += "\n\nAdditional instructions:\n" + override
	}
	notes, err := s.store.ListTaskNotes(ctx, task.ID)
	if err != nil {
		return "", err
	}
	if len(notes) > 0 {
		prompt += "\n\nTeam notes:"
		for _, note := range notes {
			prompt += "\n- [" + note.Author + "] " + note.Body
		}
	}
	return prompt, nil
}

// ensureTaskRunSession returns a reusable agent session when the agent enables
// task or agent level context. Empty means a fresh ACP session per run.
func (s *Server) ensureTaskRunSession(ctx context.Context, task *domain.Task, agentID, repo, base string) string {
	agent, _ := s.store.GetAgent(ctx, agentID)
	mode := "off"
	backend, model, agentName := "", "", ""
	if agent != nil {
		mode = agent.SessionMode
		if mode == "" {
			mode = "off"
		}
		backend, model, agentName = agent.Backend, agent.Model, agent.Name
	}
	taskID := task.ID
	if mode == "agent" {
		taskID = ""
	}
	if mode == "task" || mode == "agent" {
		if existing, _ := s.store.FindReusableAgentSession(ctx, task.WorkspaceID, taskID, agentID); existing != nil {
			return existing.ID
		}
	}
	title := task.Title
	if mode == "agent" && agentName != "" {
		title = agentName + " context"
	}
	sess, err := s.store.CreateAgentSession(ctx, &domain.AgentSession{
		WorkspaceID: task.WorkspaceID, TaskID: taskID, AgentID: agentID,
		Backend: backend, Model: model, Mode: "chat", Title: title,
		Repo: repo, Base: base, Status: "active", CreatedBy: "system",
	})
	if err != nil {
		return ""
	}
	return sess.ID
}

// createTaskRun persists one run under a task.
func (s *Server) createTaskRun(ctx context.Context, task *domain.Task, agent, role, repo, base, prompt string) (*domain.Run, error) {
	return s.createTaskRunModel(ctx, task, agent, role, repo, base, prompt, "")
}

func (s *Server) createTaskRunModel(ctx context.Context, task *domain.Task, agent, role, repo, base, prompt, model string) (*domain.Run, error) {
	if prefix := s.skillPrompt(ctx, task.WorkspaceID, agent); prefix != "" {
		prompt = prefix + "\n\n" + prompt
	}
	id := "run_" + ulid.Make().String()
	run := &domain.Run{
		ID:          id,
		WorkspaceID: task.WorkspaceID,
		Work: domain.WorkRef{
			Provider: task.Provider, Instance: task.Instance, Type: task.WorkType,
			ID: task.WorkID, URL: task.URL, Revision: task.Revision,
		},
		Snapshot: domain.Snapshot{
			Title: task.Title, Body: task.Body, Labels: task.Labels, Author: task.Assignee,
		},
		AgentID:   agent,
		Status:    domain.RunQueued,
		Repo:      repo,
		Base:      base,
		Branch:    "orca/" + id[len(id)-8:],
		Prompt:    prompt,
		Model:     model,
		SessionID: s.ensureTaskRunSession(ctx, task, agent, repo, base),
		TaskID:    task.ID,
		Role:      role,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.store.CreateRun(ctx, run); err != nil {
		return nil, err
	}
	if repo != "" {
		_, _ = s.store.CreateWorktree(ctx, &domain.Worktree{
			WorkspaceID: task.WorkspaceID,
			TaskID:      task.ID,
			RunID:       id,
			Repo:        repo,
			Branch:      run.Branch,
			BaseRef:     base,
			StartRef:    base,
			Status:      "creating",
		})
	}
	dispatch, err := s.store.CreateDispatch(ctx, task, id, agent, 1)
	if err != nil {
		return nil, err
	}
	_ = s.store.SetTaskStatus(ctx, task.ID, domain.TaskInProgress)
	_, _ = s.store.AppendEvent(ctx, id, domain.EventRunCreated, map[string]any{
		"taskId":     task.ID,
		"agent":      run.AgentID,
		"role":       run.Role,
		"dispatchId": dispatch.ID,
	})
	s.recordTaskEvent(ctx, task.ID, "task.dispatch", map[string]any{
		"runId": id, "dispatchId": dispatch.ID, "agent": run.AgentID, "role": run.Role,
	})
	return run, nil
}

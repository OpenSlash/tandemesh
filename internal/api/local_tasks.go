package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

type localTaskRequest struct {
	WorkspaceID string   `json:"workspaceId"`
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	Labels      []string `json:"labels"`
	AgentID     string   `json:"agentId"`
	MemberID    string   `json:"memberId"`
	Login       string   `json:"login"`
}

// createLocalTask creates a task that is not backed by an external platform.
func (s *Server) createLocalTask(w http.ResponseWriter, r *http.Request) {
	var req localTaskRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	if req.Title == "" {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "title is required")
		return
	}
	if req.WorkspaceID == "" {
		req.WorkspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, req.WorkspaceID) {
		return
	}
	if req.AgentID != "" {
		agent, err := s.store.GetAgent(r.Context(), req.AgentID)
		if err != nil || agent == nil || !s.agentInWorkspace(agent, req.WorkspaceID) {
			writeErr(w, http.StatusBadRequest, "E_AGENT", "agent not found in this workspace")
			return
		}
	}
	ref := domain.WorkRef{Provider: "local", Instance: req.WorkspaceID, Type: "task", ID: "tsk_" + ulid.Make().String()}
	task, err := s.store.EnsureTaskIn(r.Context(), req.WorkspaceID, ref, domain.Snapshot{Title: req.Title, Body: req.Body, Labels: req.Labels})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	_ = s.store.SetTaskOrigin(r.Context(), task.ID, "local")
	task.Origin = "local"
	if req.MemberID != "" && !s.workspaceMemberExists(r.Context(), req.WorkspaceID, req.MemberID) {
		writeErr(w, http.StatusBadRequest, "E_MEMBER", "member not found in this workspace")
		return
	}
	if req.MemberID != "" {
		login := req.Login
		if login == "" {
			login = req.MemberID
		}
		if !s.workspaceMemberExists(r.Context(), task.WorkspaceID, req.MemberID) {
			writeErr(w, http.StatusBadRequest, "E_MEMBER", "member not found in this workspace")
			return
		}
		if err := s.store.SetTaskMemberAssignee(r.Context(), task.ID, req.MemberID, login); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		task.AssigneeMemberID, task.Assignee = req.MemberID, login
		s.notifyTaskAssignment(r.Context(), task, req.MemberID, login)
	}
	if req.AgentID != "" {
		if err := s.store.SetTaskAssignee(r.Context(), task.ID, req.AgentID); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		task.AssigneeAgentID = req.AgentID
	}
	writeJSON(w, http.StatusCreated, task)
}

type localAssigneeRequest struct {
	MemberID     string `json:"memberId"`
	Login        string `json:"login"`
	AutoDispatch bool   `json:"autoDispatch"`
	AgentID      string `json:"agentId"`
	Repo         string `json:"repo"`
	Base         string `json:"base"`
}

func (s *Server) assignLocalMember(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireWorkspace(w, r, task.WorkspaceID) {
		return
	}
	var req localAssigneeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	req.MemberID = strings.TrimSpace(req.MemberID)
	if req.MemberID == "" {
		writeErr(w, http.StatusBadRequest, "E_FIELDS", "memberId is required")
		return
	}
	login := strings.TrimSpace(req.Login)
	if login == "" {
		login = req.MemberID
	}
	if !s.workspaceMemberExists(r.Context(), task.WorkspaceID, req.MemberID) {
		writeErr(w, http.StatusBadRequest, "E_MEMBER", "member not found in this workspace")
		return
	}
	if err := s.store.SetTaskMemberAssignee(r.Context(), task.ID, req.MemberID, login); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	task.AssigneeMemberID, task.Assignee, task.AssigneeAgentID, task.AssigneeGroupID = req.MemberID, login, "", ""
	s.notifyTaskAssignment(r.Context(), task, req.MemberID, login)
	if !req.AutoDispatch {
		writeJSON(w, http.StatusOK, task)
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
		writeErr(w, http.StatusConflict, "E_CIRCUIT_BROKEN", "task failed three consecutive dispatches")
		return
	}
	agent, err := s.pickAgent(r.Context(), task)
	if req.AgentID != "" {
		agent, err = s.store.GetAgent(r.Context(), req.AgentID)
	}
	if err != nil || agent == nil {
		writeErr(w, http.StatusBadRequest, "E_AGENT", "agent not found")
		return
	}
	prompt, err := s.taskPrompt(r.Context(), task, "")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	run, err := s.createTaskRun(r.Context(), task, agent.ID, "lead", req.Repo, req.Base, prompt)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_RUN", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"task": task, "run": run})
}

// workspaceMemberExists reports whether memberID is the single local account.
// The open-source core has no workspace member list; the local user owns every
// workspace.
func (s *Server) workspaceMemberExists(_ context.Context, _ string, memberID string) bool {
	local := s.cfg.User
	if local == "" {
		local = "me"
	}
	return memberID == local
}

func (s *Server) notifyTaskAssignment(ctx context.Context, task *domain.Task, memberID, login string) {
	_ = s.store.CreateNotification(ctx, &domain.Notification{
		ID: "notif_" + ulid.Make().String(), WorkspaceID: task.WorkspaceID, Type: "task_assigned",
		Title: "Task assigned: " + task.Title, Body: login + " was assigned", Link: "/tasks/" + task.ID,
	})
}

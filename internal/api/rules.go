package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Server) listAssignmentRules(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace")
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	items, err := s.store.ListAssignmentRules(r.Context(), workspaceID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type assignmentRuleRequest struct {
	WorkspaceID  string `json:"workspaceId"`
	Name         string `json:"name"`
	Provider     string `json:"provider"`
	Instance     string `json:"instance"`
	Label        string `json:"label"`
	Author       string `json:"author"`
	Assignee     string `json:"assignee"`
	AgentID      string `json:"agentId"`
	GroupID      string `json:"groupId"`
	Repo         string `json:"repo"`
	AutoDispatch bool   `json:"autoDispatch"`
	Policy       string `json:"policy"`
	Enabled      *bool  `json:"enabled"`
	Priority     int    `json:"priority"`
}

func (s *Server) createAssignmentRule(w http.ResponseWriter, r *http.Request) {
	var req assignmentRuleRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if req.AgentID == "" && req.GroupID == "" {
		writeErr(w, http.StatusBadRequest, "E_AGENT", "agentId or groupId is required")
		return
	}
	if req.WorkspaceID == "" {
		req.WorkspaceID = "ws_default"
	}
	if !s.requireWorkspace(w, r, req.WorkspaceID) {
		return
	}
	if req.GroupID != "" {
		group, err := s.store.GetGroup(r.Context(), req.GroupID)
		if err != nil || group == nil || group.WorkspaceID != req.WorkspaceID {
			writeErr(w, http.StatusBadRequest, "E_GROUP", "group not found in this workspace")
			return
		}
	} else {
		agent, err := s.store.GetAgent(r.Context(), req.AgentID)
		if err != nil || agent == nil || !s.agentInWorkspace(agent, req.WorkspaceID) {
			writeErr(w, http.StatusBadRequest, "E_AGENT", "agent not found in this workspace")
			return
		}
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	policy := req.Policy
	if policy == "" {
		if req.AutoDispatch {
			policy = "conditional"
		} else {
			policy = "manual"
		}
	}
	rule := &domain.AssignmentRule{
		ID: "rule_" + ulid.Make().String(), WorkspaceID: req.WorkspaceID,
		Name: req.Name, Provider: req.Provider, Instance: req.Instance,
		Label: req.Label, Author: req.Author, Assignee: req.Assignee,
		AgentID: req.AgentID, GroupID: req.GroupID, Repo: req.Repo, AutoDispatch: policy != "manual", Policy: policy,
		Enabled: enabled, Priority: req.Priority,
	}
	if err := s.store.UpsertAssignmentRule(r.Context(), rule); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (s *Server) deleteAssignmentRule(w http.ResponseWriter, r *http.Request) {
	ok, err := s.store.DeleteAssignmentRule(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "rule not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// applyAssignmentRules assigns a synced task and optionally dispatches it.
func (s *Server) applyAssignmentRules(ctx context.Context, task *domain.Task) *domain.Run {
	if task == nil {
		return nil
	}
	rules, err := s.store.ListAssignmentRules(ctx, task.WorkspaceID)
	if err != nil {
		return nil
	}
	if automations, err := s.store.ListAutomations(ctx, task.WorkspaceID); err == nil {
		for _, automation := range automations {
			if !automation.Enabled || automation.TriggerKind != "task.synced" || automation.Schedule != "" {
				continue
			}
			groupID, _ := automation.Writeback["groupId"].(string)
			policy, _ := automation.Writeback["policy"].(string)
			author, _ := automation.Writeback["author"].(string)
			assignee, _ := automation.Writeback["assignee"].(string)
			autoDispatch, _ := automation.Writeback["autoDispatch"].(bool)
			if policy == "" {
				policy = "conditional"
			}
			rules = append(rules, &domain.AssignmentRule{
				ID: automation.ID, WorkspaceID: automation.WorkspaceID, Name: automation.Name,
				Provider: automation.Provider, Instance: automation.Instance, Label: automation.TriggerValue,
				Author: author, Assignee: assignee, AgentID: automation.AgentID, GroupID: groupID,
				Repo: automation.Repo, AutoDispatch: autoDispatch, Policy: policy, Enabled: automation.Enabled,
				Priority: automation.Priority, CreatedAt: automation.CreatedAt,
			})
		}
	}
	mappedAuthor, mappedAssignee := "", ""
	if item, err := s.store.ResolveExternalIdentity(ctx, task.WorkspaceID, task.Provider, task.Author); err == nil && item != nil {
		mappedAuthor = item.MemberID
	}
	if item, err := s.store.ResolveExternalIdentity(ctx, task.WorkspaceID, task.Provider, task.Assignee); err == nil && item != nil {
		mappedAssignee = item.MemberID
	}
	for _, rule := range rules {
		if !rule.Enabled || (rule.AgentID == "" && rule.GroupID == "") {
			continue
		}
		if rule.Policy != "always" && rule.Provider != "" && rule.Provider != task.Provider {
			continue
		}
		if rule.Policy != "always" && rule.Instance != "" && rule.Instance != task.Instance {
			continue
		}
		if rule.Policy != "always" && rule.Label != "" && !containsFold(task.Labels, rule.Label) {
			continue
		}
		if rule.Policy != "always" && rule.Author != "" && !actorMatches(task.Author, mappedAuthor, rule.Author) {
			continue
		}
		if rule.Policy != "always" && rule.Assignee != "" && !actorMatches(task.Assignee, mappedAssignee, rule.Assignee) {
			continue
		}
		if rule.GroupID != "" {
			if err := s.store.SetTaskAssigneeGroup(ctx, task.ID, rule.GroupID); err != nil {
				return nil
			}
			task.AssigneeGroupID = rule.GroupID
			task.AssigneeAgentID = ""
			if !rule.AutoDispatch || task.AutoDispatchedAt != nil || task.ActiveRuns > 0 {
				return nil
			}
			members, err := s.store.ListGroupMembers(ctx, rule.GroupID)
			if err != nil {
				return nil
			}
			roles := make([]teamRoleRequest, 0, len(members))
			for _, member := range members {
				role := member.Role
				if role == "" {
					role = domain.RoleImplementer
				}
				roles = append(roles, teamRoleRequest{Role: role, AgentID: member.AgentID})
			}
			runs, err := s.launchTeam(ctx, task, rule.Repo, "", roles)
			if err != nil || len(runs) == 0 {
				return nil
			}
			now := time.Now().UTC()
			_ = s.store.SetTaskAutoDispatched(ctx, task.ID, now)
			task.AutoDispatchedAt = &now
			return runs[0]
		}
		if err := s.store.SetTaskAssignee(ctx, task.ID, rule.AgentID); err != nil {
			return nil
		}
		task.AssigneeAgentID = rule.AgentID
		if !rule.AutoDispatch || task.AutoDispatchedAt != nil || task.ActiveRuns > 0 {
			return nil
		}
		prompt, err := s.taskPrompt(ctx, task, "")
		if err != nil {
			return nil
		}
		run, err := s.createTaskRun(ctx, task, rule.AgentID, "lead", rule.Repo, "", prompt)
		if err != nil {
			return nil
		}
		now := time.Now().UTC()
		_ = s.store.SetTaskAutoDispatched(ctx, task.ID, now)
		task.AutoDispatchedAt = &now
		return run
	}
	return nil
}

func actorMatches(external, mapped, expected string) bool {
	if strings.Contains(strings.ToLower(external), strings.ToLower(expected)) {
		return true
	}
	return mapped != "" && strings.EqualFold(mapped, expected)
}

func containsFold(items []string, needle string) bool {
	for _, item := range items {
		if strings.EqualFold(item, needle) {
			return true
		}
	}
	return false
}

package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

type agentRequest struct {
	ID           string         `json:"id"`
	WorkspaceID  string         `json:"workspaceId"`
	WorkspaceIDs []string       `json:"workspaceIds"`
	MachineID    string         `json:"machineId"`
	Name         string         `json:"name"`
	Instructions string         `json:"instructions"`
	Backend      string         `json:"backend"`
	Role         string         `json:"role"`
	Capabilities []string       `json:"capabilities"`
	Model        string         `json:"model"`
	Skills       []string       `json:"skills"`
	Repos        []string       `json:"repos"`
	Tools        map[string]any `json:"tools"`
	SessionMode  string         `json:"sessionMode"`
}

func normalizeSessionMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "task", "agent":
		return strings.ToLower(strings.TrimSpace(mode))
	default:
		return "off"
	}
}

func agentWorkspaces(agent *domain.Agent) []string {
	if len(agent.WorkspaceIDs) > 0 {
		return agent.WorkspaceIDs
	}
	if agent.WorkspaceID != "" {
		return []string{agent.WorkspaceID}
	}
	return nil
}

func (s *Server) agentInWorkspace(agent *domain.Agent, workspaceID string) bool {
	for _, id := range agentWorkspaces(agent) {
		if id == workspaceID {
			return true
		}
	}
	return false
}

func (s *Server) allowAgent(ctx context.Context, agent *domain.Agent) bool {
	for _, workspaceID := range agentWorkspaces(agent) {
		if s.allowWorkspace(ctx, workspaceID) {
			return true
		}
	}
	return false
}

func (s *Server) listAgents(w http.ResponseWriter, r *http.Request) {
	workspaceID := r.URL.Query().Get("workspace")
	all := workspaceID == "all"
	if workspaceID == "" {
		workspaceID = "ws_default"
	}
	if !all && !s.requireWorkspace(w, r, workspaceID) {
		return
	}
	lookup := workspaceID
	if all {
		lookup = ""
	}
	agents, err := s.store.ListAgents(r.Context(), lookup)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if all {
		filtered := make([]*domain.Agent, 0, len(agents))
		for _, agent := range agents {
			if s.allowAgent(r.Context(), agent) {
				filtered = append(filtered, agent)
			}
		}
		agents = filtered
	}
	if agents == nil {
		agents = []*domain.Agent{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": agents})
}

func (s *Server) createAgent(w http.ResponseWriter, r *http.Request) {
	var req agentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeErr(w, http.StatusBadRequest, "E_NAME", "name is required")
		return
	}
	if req.WorkspaceID == "" && len(req.WorkspaceIDs) == 0 {
		req.WorkspaceID = "ws_default"
	}
	workspaceIDs := req.WorkspaceIDs
	if len(workspaceIDs) == 0 {
		workspaceIDs = []string{req.WorkspaceID}
	}
	seen := map[string]struct{}{}
	clean := make([]string, 0, len(workspaceIDs))
	for _, workspaceID := range workspaceIDs {
		if workspaceID == "" {
			continue
		}
		if _, ok := seen[workspaceID]; ok {
			continue
		}
		seen[workspaceID] = struct{}{}
		if !s.requireWorkspace(w, r, workspaceID) {
			return
		}
		clean = append(clean, workspaceID)
	}
	if len(clean) == 0 {
		writeErr(w, http.StatusBadRequest, "E_WORKSPACE", "at least one workspace is required")
		return
	}
	req.WorkspaceID = clean[0]
	if req.MachineID != "" {
		machines, err := s.store.ListMachines(r.Context())
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
		found := false
		for _, machine := range machines {
			if machine.ID == req.MachineID {
				found = true
				break
			}
		}
		if !found {
			writeErr(w, http.StatusBadRequest, "E_MACHINE", "machine not found")
			return
		}
	}
	agent := &domain.Agent{
		ID:           "agt_" + ulid.Make().String(),
		WorkspaceID:  req.WorkspaceID,
		WorkspaceIDs: clean,
		MachineID:    req.MachineID,
		Name:         req.Name,
		Instructions: req.Instructions,
		Backend:      req.Backend,
		Role:         req.Role,
		Capabilities: req.Capabilities,
		Model:        req.Model,
		Skills:       req.Skills,
		Repos:        req.Repos,
		Tools:        req.Tools,
		SessionMode:  normalizeSessionMode(req.SessionMode),
	}
	if agent.Backend == "" {
		agent.Backend = "claude"
	}
	if err := s.store.UpsertAgent(r.Context(), agent); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if err := s.store.SetAgentWorkspaces(r.Context(), agent.ID, clean); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, agent)
}

func (s *Server) getAgent(w http.ResponseWriter, r *http.Request) {
	agent, err := s.store.GetAgent(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if agent == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "agent not found")
		return
	}
	if !s.allowAgent(r.Context(), agent) {
		writeErr(w, http.StatusForbidden, "E_FORBIDDEN", "not a member of any assigned workspace")
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

func (s *Server) updateAgent(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	existing, err := s.store.GetAgent(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if existing == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "agent not found")
		return
	}
	if !s.allowAgent(r.Context(), existing) {
		writeErr(w, http.StatusForbidden, "E_FORBIDDEN", "not a member of any assigned workspace")
		return
	}
	var req agentRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	var assigned []string
	if len(req.WorkspaceIDs) > 0 || req.WorkspaceID != "" {
		ids := req.WorkspaceIDs
		if len(ids) == 0 {
			ids = []string{req.WorkspaceID}
		}
		seen := map[string]struct{}{}
		for _, workspaceID := range ids {
			if workspaceID == "" {
				continue
			}
			if _, ok := seen[workspaceID]; ok {
				continue
			}
			seen[workspaceID] = struct{}{}
			if !s.requireWorkspace(w, r, workspaceID) {
				return
			}
			assigned = append(assigned, workspaceID)
		}
		if len(assigned) == 0 {
			writeErr(w, http.StatusBadRequest, "E_WORKSPACE", "at least one workspace is required")
			return
		}
		existing.WorkspaceID = assigned[0]
		existing.WorkspaceIDs = assigned
	}
	if req.Name != "" {
		existing.Name = req.Name
	}
	existing.Instructions = req.Instructions
	if req.Backend != "" {
		existing.Backend = req.Backend
	}
	existing.Role = req.Role
	existing.Capabilities = req.Capabilities
	existing.Model = req.Model
	existing.Skills = req.Skills
	existing.Repos = req.Repos
	existing.Tools = req.Tools
	existing.SessionMode = normalizeSessionMode(req.SessionMode)
	existing.MachineID = req.MachineID
	if err := s.store.UpsertAgent(r.Context(), existing); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if len(assigned) > 0 {
		if err := s.store.SetAgentWorkspaces(r.Context(), existing.ID, assigned); err != nil {
			writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
			return
		}
	}
	writeJSON(w, http.StatusOK, existing)
}

func (s *Server) deleteAgent(w http.ResponseWriter, r *http.Request) {
	agent, err := s.store.GetAgent(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if agent == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "agent not found")
		return
	}
	if !s.allowAgent(r.Context(), agent) {
		writeErr(w, http.StatusForbidden, "E_FORBIDDEN", "not a member of any assigned workspace")
		return
	}
	ok, err := s.store.DeleteAgent(r.Context(), agent.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if !ok {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "agent not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type agentSkillsRequest struct {
	Skills []string `json:"skills"`
}

func (s *Server) setAgentSkills(w http.ResponseWriter, r *http.Request) {
	agent, err := s.store.GetAgent(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	if agent == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "agent not found")
		return
	}
	if !s.allowAgent(r.Context(), agent) {
		writeErr(w, http.StatusForbidden, "E_FORBIDDEN", "not a member of any assigned workspace")
		return
	}
	var req agentSkillsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeErr(w, http.StatusBadRequest, "E_BODY", err.Error())
		return
	}
	seen := map[string]struct{}{}
	skills := make([]string, 0, len(req.Skills))
	for _, skill := range req.Skills {
		skill = strings.TrimSpace(skill)
		if skill == "" {
			continue
		}
		if _, ok := seen[skill]; ok {
			continue
		}
		seen[skill] = struct{}{}
		skills = append(skills, skill)
	}
	agent.Skills = skills
	if err := s.store.UpsertAgent(r.Context(), agent); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, agent)
}

package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// summonAgent picks the best available agent for a task, assigns it and starts a run.
func (s *Server) summonAgent(w http.ResponseWriter, r *http.Request) {
	task, err := s.store.GetTask(r.Context(), chi.URLParam(r, "id"))
	if err != nil || task == nil {
		writeErr(w, http.StatusNotFound, "E_NOT_FOUND", "task not found")
		return
	}
	if !s.requireWorkspace(w, r, task.WorkspaceID) {
		return
	}
	if task.ActiveRuns > 0 {
		writeErr(w, http.StatusConflict, "E_ACTIVE", "task is already running")
		return
	}
	agent, err := s.pickAgent(r.Context(), task)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "E_AGENT", err.Error())
		return
	}
	if err := s.store.SetTaskAssignee(r.Context(), task.ID, agent.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	task.AssigneeAgentID = agent.ID
	task.AssigneeGroupID = ""
	prompt, err := s.taskPrompt(r.Context(), task, "")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_DB", err.Error())
		return
	}
	repo := task.Instance
	if len(agent.Repos) > 0 {
		repo = agent.Repos[0]
	}
	run, err := s.createTaskRun(r.Context(), task, agent.ID, "lead", repo, "", prompt)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "E_RUN", err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"agent": agent, "run": run})
}

// pickAgent scores workspace agents by task text, capabilities, repos and host availability.
func (s *Server) pickAgent(ctx context.Context, task *domain.Task) (*domain.Agent, error) {
	agents, err := s.store.ListAgents(ctx, task.WorkspaceID)
	if err != nil {
		return nil, err
	}
	machines, err := s.store.ListMachines(ctx)
	if err != nil {
		return nil, err
	}
	online := map[string]*domain.Machine{}
	for _, m := range machines {
		if time.Since(m.LastSeen) < 2*time.Minute {
			online[m.ID] = m
		}
	}
	text := strings.ToLower(task.Title + " " + task.Body + " " + strings.Join(task.Labels, " "))
	bestScore := -1
	var best *domain.Agent
	for _, agent := range agents {
		if !agentAvailable(agent, online) {
			continue
		}
		score := 0
		for _, cap := range agent.Capabilities {
			if cap != "" && strings.Contains(text, strings.ToLower(cap)) {
				score += 3
			}
		}
		for _, skill := range agent.Skills {
			if skill != "" && strings.Contains(text, strings.ToLower(skill)) {
				score += 2
			}
		}
		if agent.Role != "" && strings.Contains(text, strings.ToLower(agent.Role)) {
			score += 2
		}
		for _, repo := range agent.Repos {
			if repo != "" && strings.Contains(strings.ToLower(task.Instance), strings.ToLower(repo)) {
				score += 2
			}
		}
		if agent.MachineID != "" {
			score++
		}
		if score > bestScore || (score == bestScore && best != nil && agent.Name < best.Name) {
			bestScore = score
			best = agent
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no online agent is available")
	}
	return best, nil
}

func agentAvailable(agent *domain.Agent, online map[string]*domain.Machine) bool {
	if agent.MachineID != "" {
		m, ok := online[agent.MachineID]
		return ok && machineSupportsAgent(m, agent.Backend)
	}
	for _, m := range online {
		if machineSupportsAgent(m, agent.Backend) {
			return true
		}
	}
	return false
}

func machineSupportsAgent(m *domain.Machine, backend string) bool {
	if backend == "" {
		return true
	}
	for _, a := range m.Agents {
		if a == backend {
			return true
		}
	}
	if _, ok := m.Runtimes[backend]; ok {
		return true
	}
	return len(m.Agents) == 0 && len(m.Runtimes) == 0
}

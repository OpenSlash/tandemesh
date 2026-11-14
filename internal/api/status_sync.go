package api

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

// automationForTask finds the event automation whose connection matches the task instance.
func (s *Server) automationForTask(ctx context.Context, task *domain.Task) *domain.Automation {
	automations, err := s.store.ListAutomations(ctx, task.WorkspaceID)
	if err != nil {
		return nil
	}
	for _, automation := range automations {
		if automation.Schedule != "" || automation.Provider != task.Provider {
			continue
		}
		conn, err := s.store.GetConnection(ctx, automation.ConnectionID)
		if err == nil && conn != nil && conn.Instance == task.Instance {
			return automation
		}
	}
	return nil
}

// conflictPolicy reads the automation conflict strategy; platform is the default.
func conflictPolicy(b *domain.Automation) string {
	if b == nil || b.Writeback == nil {
		return "platform"
	}
	if value, ok := b.Writeback["conflict"].(string); ok {
		switch value {
		case "local", "manual":
			return value
		}
	}
	return "platform"
}

// targetStateName resolves the provider state name for a local status.
func targetStateName(b *domain.Automation, status string) string {
	if b == nil {
		return ""
	}
	key := ""
	switch status {
	case domain.TaskOpen:
		key = "stateOpen"
	case domain.TaskInProgress:
		key = "stateInProgress"
	case domain.TaskDone:
		key = "stateDone"
	}
	if key != "" && b.Writeback != nil {
		if value, ok := b.Writeback[key].(string); ok && value != "" {
			return value
		}
	}
	for name, mapped := range b.StateMap {
		if mapped == status {
			return name
		}
	}
	return ""
}

// enqueueStatusWriteback pushes a local status change back to the platform.
func (s *Server) enqueueStatusWriteback(ctx context.Context, task *domain.Task, status string) {
	if !s.cfg.WritebackState || task == nil {
		return
	}
	automation := s.automationForTask(ctx, task)
	if automation == nil || !boolOption(automation.Writeback, "status", false) {
		return
	}
	ad, ok := s.integrations.Get(task.Provider)
	if !ok || !hasCapability(ad, integration.CapTransition) {
		return
	}
	target := targetStateName(automation, status)
	if target == "" {
		return
	}
	item := &domain.Writeback{
		ID:           "wb_" + ulid.Make().String(),
		Provider:     task.Provider,
		ConnectionID: automation.ConnectionID,
		Action:       "transition",
		Work: domain.WorkRef{
			Provider: task.Provider, Instance: task.Instance, Type: task.WorkType,
			ID: task.WorkID, URL: task.URL, Revision: task.Revision,
		},
		URL:    target,
		Kind:   "transition",
		NextAt: time.Now().UTC(),
	}
	if err := s.store.EnqueueWriteback(ctx, item); err != nil {
		s.log.Warn("enqueue status writeback", "task", task.ID, "err", err)
	}
}

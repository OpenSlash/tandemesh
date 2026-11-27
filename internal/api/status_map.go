package api

import (
	"context"
	"strings"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// applyPlatformState records the external workflow state and maps it to a
// local execution status when the platform state group is recognizable.
func (s *Server) applyPlatformState(ctx context.Context, task *domain.Task, snap domain.Snapshot, stateMap map[string]string, conflict string) {
	if task == nil || (snap.Status == "" && snap.StatusGroup == "") {
		return
	}
	local := mapPlatformStatusWith(stateMap, snap.StatusGroup, snap.Status)
	if local == task.Status && snap.Status == task.PlatformState && snap.StatusGroup == task.PlatformStateGroup {
		return
	}
	if task.PlatformState != "" && task.Status != local {
		if conflict == "local" && task.PlatformState == snap.Status && task.PlatformStateGroup == snap.StatusGroup {
			return
		}
		if conflict == "manual" && (task.PlatformState != snap.Status || task.PlatformStateGroup != snap.StatusGroup) {
			s.recordTaskEvent(ctx, task.ID, "task.conflict", map[string]any{
				"platform": snap.Status, "local": task.Status, "mapped": local,
			})
			return
		}
	}
	_ = s.store.SetTaskPlatformState(ctx, task.ID, snap.Status, snap.StatusGroup, local)
	if domain.TaskClosed(local) {
		_ = s.store.SetTaskExecutionState(ctx, task.ID, domain.TaskExecutionCompleted)
		_, _ = s.store.PromoteDependents(ctx, task.ID)
	} else if local == domain.TaskOpen {
		_ = s.store.EvaluateTaskReadiness(ctx, task.ID)
	}
}

// mapPlatformStatusWith applies a per-binding override before the default mapping.
func mapPlatformStatusWith(stateMap map[string]string, group, state string) string {
	for _, key := range []string{state, group, strings.ToLower(state), strings.ToLower(group)} {
		if key == "" {
			continue
		}
		if value, ok := stateMap[key]; ok {
			switch value {
			case domain.TaskOpen, domain.TaskInProgress, domain.TaskDone, domain.TaskCancelled:
				return value
			}
		}
	}
	return mapPlatformStatus(group, state)
}

// mapPlatformStatus maps common platform state groups/names onto Orca statuses.
func mapPlatformStatus(group, state string) string {
	switch strings.ToLower(group) {
	case "completed":
		return domain.TaskDone
	case "started":
		return domain.TaskInProgress
	case "unstarted", "backlog":
		return domain.TaskOpen
	case "cancelled", "canceled":
		return domain.TaskCancelled
	}
	switch strings.ToLower(state) {
	case "done", "closed", "completed", "resolved":
		return domain.TaskDone
	case "in progress", "in_progress", "started", "doing":
		return domain.TaskInProgress
	case "cancelled", "canceled":
		return domain.TaskCancelled
	}
	return ""
}

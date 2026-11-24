package store

import (
	"context"
	"errors"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// ErrTaskCycle is returned when a parent or dependency would form a cycle.
var ErrTaskCycle = errors.New("task relation would create a cycle")

// SetTaskParent links a task to a parent (empty clears it).
func (s *Store) SetTaskParent(ctx context.Context, id, parentID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tasks SET parent_id = ? WHERE id = ?`, parentID, id)
	return err
}

// ListTaskChildren returns direct subtasks.
func (s *Store) ListTaskChildren(ctx context.Context, parentID string) ([]*domain.Task, error) {
	return s.ListTasks(ctx, TaskFilter{Parent: parentID, Limit: 200})
}

// ListTaskDependencyIDs returns ids the task depends on.
func (s *Store) ListTaskDependencyIDs(ctx context.Context, taskID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT depends_on FROM task_dependencies WHERE task_id = ? ORDER BY depends_on`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// AddTaskDependency links taskID to dependsOn, rejecting cycles.
func (s *Store) AddTaskDependency(ctx context.Context, taskID, dependsOn string) error {
	if taskID == dependsOn {
		return ErrTaskCycle
	}
	seen := map[string]bool{}
	stack := []string{dependsOn}
	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current == taskID {
			return ErrTaskCycle
		}
		if seen[current] {
			continue
		}
		seen[current] = true
		deps, err := s.ListTaskDependencyIDs(ctx, current)
		if err != nil {
			return err
		}
		stack = append(stack, deps...)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO task_dependencies (task_id, depends_on) VALUES (?,?)
		ON CONFLICT DO NOTHING`, taskID, dependsOn)
	return err
}

// RemoveTaskDependency deletes a dependency link.
func (s *Store) RemoveTaskDependency(ctx context.Context, taskID, dependsOn string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM task_dependencies WHERE task_id = ? AND depends_on = ?`, taskID, dependsOn)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// EvaluateTaskReadiness recomputes whether an unstarted task is blocked by dependencies.
func (s *Store) EvaluateTaskReadiness(ctx context.Context, taskID string) error {
	var task domain.Task
	var state string
	err := s.db.QueryRowContext(ctx, `SELECT status, execution_state FROM tasks WHERE id = ?`, taskID).Scan(&task.Status, &state)
	if err != nil {
		return err
	}
	if task.Status == domain.TaskDone || state == domain.TaskExecutionDispatched || state == domain.TaskExecutionCompleted || state == domain.TaskExecutionBlocked {
		return nil
	}
	var unmet int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM task_dependencies d JOIN tasks dep ON dep.id = d.depends_on
		WHERE d.task_id = ? AND dep.status != ?`, taskID, domain.TaskDone).Scan(&unmet); err != nil {
		return err
	}
	next := domain.TaskExecutionReady
	if unmet > 0 {
		next = domain.TaskExecutionPending
	}
	_, err = s.db.ExecContext(ctx, `UPDATE tasks SET execution_state = ? WHERE id = ?`, next, taskID)
	return err
}

// PromoteDependents moves pending dependents to ready once a task is done.
func (s *Store) PromoteDependents(ctx context.Context, completedTaskID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT d.task_id FROM task_dependencies d JOIN tasks t ON t.id = d.task_id
		WHERE d.depends_on = ? AND t.status != ? AND t.execution_state = ?
		AND NOT EXISTS (SELECT 1 FROM task_dependencies d2 JOIN tasks dep ON dep.id = d2.depends_on
			WHERE d2.task_id = d.task_id AND dep.status != ?)`,
		completedTaskID, domain.TaskDone, domain.TaskExecutionPending, domain.TaskDone)
	if err != nil {
		return nil, err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	now := ms(time.Now())
	for _, id := range ids {
		if _, err := s.db.ExecContext(ctx, `UPDATE tasks SET execution_state = ?, updated_at_ms = ? WHERE id = ?`,
			domain.TaskExecutionReady, now, id); err != nil {
			return nil, err
		}
	}
	return ids, nil
}

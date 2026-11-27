package store

import (
	"context"
	"fmt"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// taskMirrorTables hold data copied from the platform for one task; they go
// with the task when it is pruned.
var taskMirrorTables = []string{
	"task_inbox", "task_dependencies", "task_comments", "task_notes", "task_events",
	"external_comments", "task_attachments", "task_fields", "task_pull_requests",
	"comment_threads", "task_commits", "comment_deliveries", "import_items",
}

// PruneUnboundTask removes a synced task its workspace no longer binds. A
// task with execution history (runs, sessions, dispatches or workflow runs)
// is kept and marked unbound instead, so that history stays reachable.
func (s *Store) PruneUnboundTask(ctx context.Context, id string) (deleted bool, err error) {
	var executed bool
	err = s.db.QueryRowContext(ctx, `SELECT
		EXISTS (SELECT 1 FROM runs WHERE task_id = ?) OR
		EXISTS (SELECT 1 FROM agent_sessions WHERE task_id = ?) OR
		EXISTS (SELECT 1 FROM dispatches WHERE task_id = ?) OR
		EXISTS (SELECT 1 FROM workflow_runs WHERE task_id = ?)`, id, id, id, id).Scan(&executed)
	if err != nil {
		return false, fmt.Errorf("prune task %s: %w", id, err)
	}
	if executed {
		_, err := s.db.ExecContext(ctx, `UPDATE tasks SET platform_state = ?, updated_at_ms = ? WHERE id = ?`,
			domain.PlatformStateUnbound, ms(time.Now()), id)
		return false, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	for _, table := range taskMirrorTables {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE task_id = ?`, id); err != nil {
			return false, fmt.Errorf("prune task %s from %s: %w", id, table, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

// ListSyncedTaskScopes returns the id and instance of every task synced from
// provider into a workspace, including unbound ones.
func (s *Store) ListSyncedTaskScopes(ctx context.Context, workspaceID, provider string) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, instance FROM tasks WHERE workspace_id = ? AND provider = ?`, workspaceID, provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, instance string
		if err := rows.Scan(&id, &instance); err != nil {
			return nil, err
		}
		out[id] = instance
	}
	return out, rows.Err()
}

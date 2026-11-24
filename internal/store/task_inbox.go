package store

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// TaskInboxStates returns per-user inbox rows for the given task IDs.
func (s *Store) TaskInboxStates(ctx context.Context, userID string, taskIDs []string) (map[string]*domain.TaskInbox, error) {
	out := map[string]*domain.TaskInbox{}
	if userID == "" || len(taskIDs) == 0 {
		return out, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(taskIDs)), ",")
	args := make([]any, 0, len(taskIDs)+1)
	args = append(args, userID)
	for _, id := range taskIDs {
		args = append(args, id)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT task_id, user_id, state, read_at_ms, updated_at_ms
		FROM task_inbox WHERE user_id = ? AND task_id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var (
			inbox   domain.TaskInbox
			readAt  sql.NullInt64
			updated int64
		)
		if err := rows.Scan(&inbox.TaskID, &inbox.UserID, &inbox.State, &readAt, &updated); err != nil {
			return nil, err
		}
		inbox.ReadAt = fromNull(readAt)
		inbox.UpdatedAt = fromMS(updated)
		out[inbox.TaskID] = &inbox
	}
	return out, rows.Err()
}

// SetTaskInboxState upserts one user's triage state for a task.
func (s *Store) SetTaskInboxState(ctx context.Context, taskID, userID, state string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `INSERT INTO task_inbox (task_id, user_id, state, read_at_ms, updated_at_ms)
		VALUES (?,?,?,NULL,?)
		ON CONFLICT(task_id, user_id) DO UPDATE SET state = excluded.state, updated_at_ms = excluded.updated_at_ms`,
		taskID, userID, state, ms(now))
	return err
}

// MarkTaskRead records the first read time for one user.
func (s *Store) MarkTaskRead(ctx context.Context, taskID, userID string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `INSERT INTO task_inbox (task_id, user_id, state, read_at_ms, updated_at_ms)
		VALUES (?,?,'inbox',?,?)
		ON CONFLICT(task_id, user_id) DO UPDATE SET
			read_at_ms = COALESCE(task_inbox.read_at_ms, excluded.read_at_ms),
			updated_at_ms = excluded.updated_at_ms`,
		taskID, userID, ms(now), ms(now))
	return err
}

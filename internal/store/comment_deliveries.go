package store

import (
	"context"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// UpsertCommentDelivery records the per-target outcome of an outbound comment.
func (s *Store) UpsertCommentDelivery(ctx context.Context, d *domain.CommentDelivery) error {
	if d.UpdatedAt.IsZero() {
		d.UpdatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO comment_deliveries
		(id, workspace_id, task_id, comment_id, provider, status, external_id, error, attempts, updated_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status, external_id = excluded.external_id,
			error = excluded.error, attempts = excluded.attempts, updated_at_ms = excluded.updated_at_ms`,
		d.ID, d.WorkspaceID, d.TaskID, d.CommentID, d.Provider, d.Status, d.ExternalID, d.Error, d.Attempts, ms(d.UpdatedAt))
	return err
}

// ListCommentDeliveries returns deliveries for a task.
func (s *Store) ListCommentDeliveries(ctx context.Context, taskID string) ([]*domain.CommentDelivery, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, task_id, comment_id, provider, status, external_id, error, attempts, updated_at_ms
		FROM comment_deliveries WHERE task_id = ? ORDER BY updated_at_ms ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.CommentDelivery
	for rows.Next() {
		var (
			d       domain.CommentDelivery
			updated int64
		)
		if err := rows.Scan(&d.ID, &d.WorkspaceID, &d.TaskID, &d.CommentID, &d.Provider, &d.Status, &d.ExternalID, &d.Error, &d.Attempts, &updated); err != nil {
			return nil, err
		}
		d.UpdatedAt = fromMS(updated)
		out = append(out, &d)
	}
	return out, rows.Err()
}

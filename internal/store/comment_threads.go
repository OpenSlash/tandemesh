package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// EnsureCommentThread creates or returns the external discussion thread.
func (s *Store) EnsureCommentThread(ctx context.Context, thread *domain.CommentThread) (*domain.CommentThread, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, workspace_id, task_id, provider, external_id, source, resolved, created_at_ms, updated_at_ms
		FROM comment_threads WHERE task_id = ? AND provider = ? AND external_id = ?`,
		thread.TaskID, thread.Provider, thread.ExternalID)
	var (
		existing domain.CommentThread
		resolved int
		created  int64
		updated  int64
	)
	err := row.Scan(&existing.ID, &existing.WorkspaceID, &existing.TaskID, &existing.Provider,
		&existing.ExternalID, &existing.Source, &resolved, &created, &updated)
	if err == nil {
		existing.Resolved = resolved == 1
		existing.CreatedAt = fromMS(created)
		existing.UpdatedAt = fromMS(updated)
		return &existing, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}
	now := time.Now().UTC()
	thread.CreatedAt = now
	thread.UpdatedAt = now
	_, err = s.db.ExecContext(ctx, `INSERT INTO comment_threads
		(id, workspace_id, task_id, provider, external_id, source, resolved, created_at_ms, updated_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		thread.ID, thread.WorkspaceID, thread.TaskID, thread.Provider, thread.ExternalID,
		thread.Source, 0, ms(now), ms(now))
	if err != nil {
		return nil, err
	}
	return thread, nil
}

// ListCommentThreads returns threads for a task.
func (s *Store) ListCommentThreads(ctx context.Context, taskID string) ([]*domain.CommentThread, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, task_id, provider, external_id, source, resolved, created_at_ms, updated_at_ms
		FROM comment_threads WHERE task_id = ? ORDER BY created_at_ms ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.CommentThread
	for rows.Next() {
		var (
			item     domain.CommentThread
			resolved int
			created  int64
			updated  int64
		)
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.TaskID, &item.Provider, &item.ExternalID,
			&item.Source, &resolved, &created, &updated); err != nil {
			return nil, err
		}
		item.Resolved = resolved == 1
		item.CreatedAt = fromMS(created)
		item.UpdatedAt = fromMS(updated)
		out = append(out, &item)
	}
	return out, rows.Err()
}

// SetCommentThreadResolved marks a thread resolved or reopened.
func (s *Store) SetCommentThreadResolved(ctx context.Context, id string, resolved bool) error {
	value := 0
	if resolved {
		value = 1
	}
	_, err := s.db.ExecContext(ctx, `UPDATE comment_threads SET resolved = ?, updated_at_ms = ? WHERE id = ?`,
		value, ms(time.Now()), id)
	return err
}

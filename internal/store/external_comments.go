package store

import (
	"context"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// UpsertExternalComment stores one platform comment.
func (s *Store) UpsertExternalComment(ctx context.Context, c *domain.ExternalComment) error {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO external_comments
		(id, task_id, provider, external_id, author, body, thread_id, parent_external_id,
		 author_external_id, author_member_id, direction, created_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			author = excluded.author, body = excluded.body, thread_id = excluded.thread_id,
			author_member_id = excluded.author_member_id, direction = excluded.direction`,
		c.ID, c.TaskID, c.Provider, c.ExternalID, c.Author, c.Body, c.ThreadID, c.ParentExternalID,
		c.AuthorExternalID, c.AuthorMemberID, c.Direction, ms(c.CreatedAt))
	return err
}

// ListExternalComments returns platform comments for a task.
func (s *Store) ListExternalComments(ctx context.Context, taskID string, limit int) ([]*domain.ExternalComment, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, task_id, provider, external_id, author, body,
		thread_id, parent_external_id, author_external_id, author_member_id, direction, created_at_ms
		FROM external_comments WHERE task_id = ? ORDER BY created_at_ms ASC LIMIT ?`, taskID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.ExternalComment
	for rows.Next() {
		var (
			c       domain.ExternalComment
			created int64
		)
		if err := rows.Scan(&c.ID, &c.TaskID, &c.Provider, &c.ExternalID, &c.Author, &c.Body,
			&c.ThreadID, &c.ParentExternalID, &c.AuthorExternalID, &c.AuthorMemberID, &c.Direction, &created); err != nil {
			return nil, err
		}
		c.CreatedAt = fromMS(created)
		out = append(out, &c)
	}
	return out, rows.Err()
}

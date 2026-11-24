package store

import (
	"context"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// AddTaskComment appends a discussion comment.
func (s *Store) AddTaskComment(ctx context.Context, c *domain.TaskComment) error {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO task_comments (id, task_id, author, body, created_at_ms)
		VALUES (?,?,?,?,?)`, c.ID, c.TaskID, c.Author, c.Body, ms(c.CreatedAt))
	return err
}

// ListTaskComments returns comments in creation order.
func (s *Store) ListTaskComments(ctx context.Context, taskID string) ([]*domain.TaskComment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, task_id, author, body, created_at_ms
		FROM task_comments WHERE task_id = ? ORDER BY created_at_ms ASC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.TaskComment
	for rows.Next() {
		var (
			c       domain.TaskComment
			created int64
		)
		if err := rows.Scan(&c.ID, &c.TaskID, &c.Author, &c.Body, &created); err != nil {
			return nil, err
		}
		c.CreatedAt = fromMS(created)
		out = append(out, &c)
	}
	return out, rows.Err()
}

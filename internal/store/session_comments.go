package store

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// CreateSessionComment adds a diff review comment.
func (s *Store) CreateSessionComment(ctx context.Context, comment *domain.SessionComment) (*domain.SessionComment, error) {
	if comment.ID == "" {
		comment.ID = "scm_" + ulid.Make().String()
	}
	if comment.Status == "" {
		comment.Status = "open"
	}
	if comment.Side == "" {
		comment.Side = "new"
	}
	if comment.CreatedAt.IsZero() {
		comment.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO session_comments
		(id, session_id, file, line, side, body, status, created_at_ms)
		VALUES (?,?,?,?,?,?,?,?)`,
		comment.ID, comment.SessionID, comment.File, comment.Line, comment.Side, comment.Body, comment.Status, ms(comment.CreatedAt))
	if err != nil {
		return nil, err
	}
	return comment, nil
}

// ListSessionComments returns comments in creation order.
func (s *Store) ListSessionComments(ctx context.Context, sessionID, status string) ([]*domain.SessionComment, error) {
	q := `SELECT id, session_id, file, line, side, body, status, created_at_ms
		FROM session_comments WHERE session_id = ?`
	args := []any{sessionID}
	if status != "" {
		q += " AND status = ?"
		args = append(args, status)
	}
	q += " ORDER BY created_at_ms ASC, id ASC"
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.SessionComment{}
	for rows.Next() {
		var comment domain.SessionComment
		var created int64
		if err := rows.Scan(&comment.ID, &comment.SessionID, &comment.File, &comment.Line, &comment.Side, &comment.Body, &comment.Status, &created); err != nil {
			return nil, err
		}
		comment.CreatedAt = fromMS(created)
		out = append(out, &comment)
	}
	return out, rows.Err()
}

// MarkSessionCommentsSent closes review comments after they are handed to the agent.
func (s *Store) MarkSessionCommentsSent(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE session_comments SET status = 'sent' WHERE session_id = ? AND status = 'open'`, sessionID)
	return err
}

// ResolveSessionComment marks a review comment resolved.
func (s *Store) ResolveSessionComment(ctx context.Context, id string) (*domain.SessionComment, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE session_comments SET status = 'resolved' WHERE id = ? AND status = 'open'`, id)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, nil
	}
	row := s.db.QueryRowContext(ctx, `SELECT id, session_id, file, line, side, body, status, created_at_ms FROM session_comments WHERE id = ?`, id)
	var (
		comment domain.SessionComment
		created int64
	)
	if err := row.Scan(&comment.ID, &comment.SessionID, &comment.File, &comment.Line, &comment.Side, &comment.Body, &comment.Status, &created); err != nil {
		return nil, err
	}
	comment.CreatedAt = fromMS(created)
	return &comment, nil
}

// GetSessionComment returns one review comment.
func (s *Store) GetSessionComment(ctx context.Context, id string) (*domain.SessionComment, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, session_id, file, line, side, body, status, created_at_ms FROM session_comments WHERE id = ?`, id)
	var (
		comment domain.SessionComment
		created int64
	)
	if err := row.Scan(&comment.ID, &comment.SessionID, &comment.File, &comment.Line, &comment.Side, &comment.Body, &comment.Status, &created); err != nil {
		return nil, err
	}
	comment.CreatedAt = fromMS(created)
	return &comment, nil
}

// UpdateSessionComment edits a review comment body.
func (s *Store) UpdateSessionComment(ctx context.Context, id, body string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE session_comments SET body = ? WHERE id = ?`, body, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// DeleteSessionComment removes a review comment.
func (s *Store) DeleteSessionComment(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM session_comments WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

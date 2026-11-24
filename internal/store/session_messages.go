package store

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// AppendSessionMessage persists one durable conversation message.
func (s *Store) AppendSessionMessage(ctx context.Context, msg *domain.SessionMessage) (*domain.SessionMessage, error) {
	if msg.ID == "" {
		msg.ID = "smsg_" + ulid.Make().String()
	}
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO session_messages
		(id, session_id, run_id, role, kind, text, created_at_ms)
		VALUES (?,?,?,?,?,?,?)`,
		msg.ID, msg.SessionID, msg.RunID, msg.Role, msg.Kind, msg.Text, ms(msg.CreatedAt))
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// ListSessionMessages returns messages in chronological order.
func (s *Store) ListSessionMessages(ctx context.Context, sessionID string, limit int) ([]*domain.SessionMessage, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, session_id, run_id, role, kind, text, created_at_ms
		FROM session_messages WHERE session_id = ? ORDER BY created_at_ms ASC, id ASC LIMIT ?`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.SessionMessage{}
	for rows.Next() {
		var msg domain.SessionMessage
		var created int64
		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.RunID, &msg.Role, &msg.Kind, &msg.Text, &created); err != nil {
			return nil, err
		}
		msg.CreatedAt = fromMS(created)
		out = append(out, &msg)
	}
	return out, rows.Err()
}

// ListSessionMessagesAfter returns messages after a message id.
func (s *Store) ListSessionMessagesAfter(ctx context.Context, sessionID, after string, limit int) ([]*domain.SessionMessage, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, session_id, run_id, role, kind, text, created_at_ms
		FROM session_messages WHERE session_id = ? AND id > ? ORDER BY id ASC LIMIT ?`, sessionID, after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.SessionMessage{}
	for rows.Next() {
		var msg domain.SessionMessage
		var created int64
		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.RunID, &msg.Role, &msg.Kind, &msg.Text, &created); err != nil {
			return nil, err
		}
		msg.CreatedAt = fromMS(created)
		out = append(out, &msg)
	}
	return out, rows.Err()
}

// UpdateSessionMessageText edits a user message in place.
func (s *Store) UpdateSessionMessageText(ctx context.Context, sessionID, messageID, text string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE session_messages SET text = ? WHERE id = ? AND session_id = ? AND role = 'user'`,
		text, messageID, sessionID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

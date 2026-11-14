package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// SendAgentMessage stores one already-addressed mailbox message.
func (s *Store) SendAgentMessage(ctx context.Context, m *domain.AgentMessage) error {
	if m.ID == "" {
		m.ID = "msg_" + ulid.Make().String()
	}
	if m.CreatedAt.IsZero() {
		m.CreatedAt = time.Now().UTC()
	}
	if m.Status == "" {
		m.Status = "unread"
	}
	if m.Type == "" {
		m.Type = "status"
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO agent_messages
		(id, workspace_id, task_id, run_id, dispatch_id, sender, recipient, type, subject, body, status, created_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		m.ID, m.WorkspaceID, m.TaskID, m.RunID, m.DispatchID, m.Sender, m.Recipient, m.Type, m.Subject, m.Body, m.Status, ms(m.CreatedAt))
	return err
}

// ListAgentMessages lists mailbox rows without consuming them.
func (s *Store) ListAgentMessages(ctx context.Context, taskID, runID, status string, limit int) ([]*domain.AgentMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := `SELECT id, workspace_id, task_id, run_id, dispatch_id, sender, recipient, type, subject, body, status, created_at_ms, handled_at_ms
		FROM agent_messages WHERE 1=1`
	var args []any
	if taskID != "" {
		q += " AND task_id = ?"
		args = append(args, taskID)
	}
	if runID != "" {
		q += " AND (run_id = ? OR recipient = ?)"
		args = append(args, runID, runID)
	}
	if status != "" {
		q += " AND status = ?"
		args = append(args, status)
	}
	q += " ORDER BY created_at_ms ASC LIMIT ?"
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.AgentMessage
	for rows.Next() {
		m, err := scanAgentMessage(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// CheckAgentMessage claims the oldest unread message for a run and marks it handled.
func (s *Store) CheckAgentMessage(ctx context.Context, runID string) (*domain.AgentMessage, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, workspace_id, task_id, run_id, dispatch_id, sender, recipient, type, subject, body, status, created_at_ms, handled_at_ms
		FROM agent_messages WHERE (run_id = ? OR recipient = ?) AND status = 'unread'
		ORDER BY created_at_ms ASC LIMIT 1`, runID, runID)
	m, err := scanAgentMessage(row)
	if err == sql.ErrNoRows || m == nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	if _, err := s.db.ExecContext(ctx, `UPDATE agent_messages SET status = 'handled', handled_at_ms = ? WHERE id = ? AND status = 'unread'`,
		ms(now), m.ID); err != nil {
		return nil, err
	}
	m.Status = "handled"
	m.HandledAt = &now
	return m, nil
}

// AckAgentMessage marks a specific message handled.
func (s *Store) AckAgentMessage(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE agent_messages SET status = 'handled', handled_at_ms = ? WHERE id = ? AND status = 'unread'`,
		ms(time.Now()), id)
	return err
}

func scanAgentMessage(sc interface{ Scan(...any) error }) (*domain.AgentMessage, error) {
	var (
		m       domain.AgentMessage
		created int64
		handled sql.NullInt64
	)
	err := sc.Scan(&m.ID, &m.WorkspaceID, &m.TaskID, &m.RunID, &m.DispatchID, &m.Sender, &m.Recipient,
		&m.Type, &m.Subject, &m.Body, &m.Status, &created, &handled)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	m.CreatedAt = fromMS(created)
	if handled.Valid {
		t := fromMS(handled.Int64)
		m.HandledAt = &t
	}
	return &m, nil
}

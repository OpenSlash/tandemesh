package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

const terminalColumns = `id, workspace_id, agent_session_id, machine_id, shell, cwd, cols, rows, status, exit_code, created_at_ms, last_seen_ms`

// CreateTerminalSession stores a pending terminal session.
func (s *Store) CreateTerminalSession(ctx context.Context, t *domain.TerminalSession) error {
	if t.ID == "" {
		t.ID = "term_" + ulid.Make().String()
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	if t.Cols == 0 {
		t.Cols = 120
	}
	if t.Rows == 0 {
		t.Rows = 30
	}
	if t.Status == "" {
		t.Status = "pending"
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO terminal_sessions
		(id, workspace_id, agent_session_id, machine_id, shell, cwd, cols, rows, status, exit_code, created_at_ms, last_seen_ms)
		VALUES (?,?,?,?,?,?,?,?,?,NULL,?,NULL)`,
		t.ID, t.WorkspaceID, t.AgentSessionID, t.MachineID, t.Shell, t.Cwd, t.Cols, t.Rows, t.Status, ms(t.CreatedAt))
	return err
}

// GetTerminalSession returns one terminal session or nil.
func (s *Store) GetTerminalSession(ctx context.Context, id string) (*domain.TerminalSession, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+terminalColumns+` FROM terminal_sessions WHERE id = ?`, id)
	return scanTerminalSession(row)
}

// ListTerminalSessions returns recent terminal sessions in a workspace.
func (s *Store) ListTerminalSessions(ctx context.Context, workspaceID string, limit int) ([]*domain.TerminalSession, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	query := `SELECT ` + terminalColumns + ` FROM terminal_sessions`
	args := []any{}
	if workspaceID != "" {
		query += ` WHERE workspace_id = ?`
		args = append(args, workspaceID)
	}
	query += ` ORDER BY created_at_ms DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.TerminalSession
	for rows.Next() {
		t, err := scanTerminalSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// PendingTerminalSessions returns terminals waiting for a daemon.
func (s *Store) PendingTerminalSessions(ctx context.Context, machineID string) ([]*domain.TerminalSession, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+terminalColumns+` FROM terminal_sessions
		WHERE status = 'pending' AND machine_id = ? ORDER BY created_at_ms LIMIT 20`, machineID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.TerminalSession
	for rows.Next() {
		t, err := scanTerminalSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// SetTerminalStatus updates status and last seen time.
func (s *Store) SetTerminalStatus(ctx context.Context, id, status string, exitCode *int) error {
	var code any
	if exitCode != nil {
		code = *exitCode
	}
	_, err := s.db.ExecContext(ctx, `UPDATE terminal_sessions SET status = ?, exit_code = ?, last_seen_ms = ? WHERE id = ?`,
		status, code, ms(time.Now().UTC()), id)
	return err
}

// AppendTerminalOutput stores one output chunk and returns its sequence.
func (s *Store) AppendTerminalOutput(ctx context.Context, terminalID, data string) (int, error) {
	var seq int
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq),0)+1 FROM terminal_events WHERE terminal_id = ?`, terminalID).Scan(&seq); err != nil {
		return 0, err
	}
	id := "tev_" + ulid.Make().String()
	_, err := s.db.ExecContext(ctx, `INSERT INTO terminal_events (id, terminal_id, seq, data, created_at_ms) VALUES (?,?,?,?,?)`,
		id, terminalID, seq, data, ms(time.Now().UTC()))
	return seq, err
}

// TerminalOutputSince returns output chunks after a sequence.
func (s *Store) TerminalOutputSince(ctx context.Context, terminalID string, after int) ([]domain.TerminalEvent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT seq, data, created_at_ms FROM terminal_events WHERE terminal_id = ? AND seq > ? ORDER BY seq`, terminalID, after)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.TerminalEvent, 0)
	for rows.Next() {
		var ev domain.TerminalEvent
		var created int64
		if err := rows.Scan(&ev.Seq, &ev.Data, &created); err != nil {
			return nil, err
		}
		ev.TerminalID = terminalID
		ev.CreatedAt = fromMS(created)
		out = append(out, ev)
	}
	return out, rows.Err()
}

// AppendTerminalInput queues terminal keystrokes for the daemon.
func (s *Store) AppendTerminalInput(ctx context.Context, terminalID, data string) error {
	var seq int
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(seq),0)+1 FROM terminal_inputs WHERE terminal_id = ?`, terminalID).Scan(&seq); err != nil {
		return err
	}
	id := "tin_" + ulid.Make().String()
	_, err := s.db.ExecContext(ctx, `INSERT INTO terminal_inputs (id, terminal_id, seq, data, consumed, created_at_ms) VALUES (?,?,?,?,0,?)`,
		id, terminalID, seq, data, ms(time.Now().UTC()))
	return err
}

// TakeTerminalInputs marks and returns pending input chunks.
func (s *Store) TakeTerminalInputs(ctx context.Context, terminalID string) ([]domain.TerminalInput, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, seq, data, created_at_ms FROM terminal_inputs
		WHERE terminal_id = ? AND consumed = 0 ORDER BY seq LIMIT 100`, terminalID)
	if err != nil {
		return nil, err
	}
	var out []domain.TerminalInput
	maxSeq := 0
	for rows.Next() {
		var in domain.TerminalInput
		var created int64
		if err := rows.Scan(&in.ID, &in.Seq, &in.Data, &created); err != nil {
			rows.Close()
			return nil, err
		}
		in.TerminalID = terminalID
		in.CreatedAt = fromMS(created)
		out = append(out, in)
		if in.Seq > maxSeq {
			maxSeq = in.Seq
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if maxSeq > 0 {
		_, err = s.db.ExecContext(ctx, `UPDATE terminal_inputs SET consumed = 1 WHERE terminal_id = ? AND consumed = 0 AND seq <= ?`, terminalID, maxSeq)
	}
	return out, err
}

func scanTerminalSession(sc interface{ Scan(...any) error }) (*domain.TerminalSession, error) {
	var (
		t        domain.TerminalSession
		exitCode sql.NullInt64
		created  int64
		lastSeen sql.NullInt64
	)
	if err := sc.Scan(&t.ID, &t.WorkspaceID, &t.AgentSessionID, &t.MachineID, &t.Shell, &t.Cwd, &t.Cols, &t.Rows, &t.Status, &exitCode, &created, &lastSeen); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if exitCode.Valid {
		code := int(exitCode.Int64)
		t.ExitCode = &code
	}
	t.CreatedAt = fromMS(created)
	if lastSeen.Valid {
		ts := fromMS(lastSeen.Int64)
		t.LastSeenAt = &ts
	}
	return &t, nil
}

package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// CreateAgentSession inserts a persistent agent conversation.
func (s *Store) CreateAgentSession(ctx context.Context, sess *domain.AgentSession) (*domain.AgentSession, error) {
	if sess.ID == "" {
		sess.ID = "ses_" + ulid.Make().String()
	}
	if sess.Mode == "" {
		sess.Mode = "chat"
	}
	if sess.Status == "" {
		sess.Status = "active"
	}
	now := time.Now().UTC()
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = now
	}
	sess.UpdatedAt = now
	_, err := s.db.ExecContext(ctx, `INSERT INTO agent_sessions
		(id, workspace_id, task_id, agent_id, backend, model, reasoning, mode, title, repo, base, worktree_id, status, backend_session_id, last_run_id, created_by, created_at_ms, updated_at_ms, seed_context)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		sess.ID, sess.WorkspaceID, sess.TaskID, sess.AgentID, sess.Backend, sess.Model, sess.Reasoning, sess.Mode, sess.Title,
		sess.Repo, sess.Base, sess.WorktreeID, sess.Status, sess.BackendSessionID, sess.LastRunID, sess.CreatedBy, ms(sess.CreatedAt), ms(sess.UpdatedAt), sess.Context)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

// GetAgentSession returns a session or nil.
func (s *Store) GetAgentSession(ctx context.Context, id string) (*domain.AgentSession, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, workspace_id, task_id, agent_id, backend, model, reasoning, mode, title, repo, base, worktree_id, status,
		(SELECT status FROM runs r WHERE r.session_id = agent_sessions.id ORDER BY r.created_at_ms DESC LIMIT 1),
		backend_session_id, last_run_id, created_by, created_at_ms, updated_at_ms, seed_context, closed_at_ms FROM agent_sessions WHERE id = ?`, id)
	return scanAgentSession(row)
}

// ListAgentSessions returns sessions filtered by workspace, task or status.
func (s *Store) ListAgentSessions(ctx context.Context, workspaceID, taskID, status string, limit int) ([]*domain.AgentSession, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	q := `SELECT id, workspace_id, task_id, agent_id, backend, model, reasoning, mode, title, repo, base, worktree_id, status,
		(SELECT status FROM runs r WHERE r.session_id = agent_sessions.id ORDER BY r.created_at_ms DESC LIMIT 1),
		backend_session_id, last_run_id, created_by, created_at_ms, updated_at_ms, seed_context, closed_at_ms FROM agent_sessions WHERE 1=1`
	var args []any
	if workspaceID != "" {
		q += " AND workspace_id = ?"
		args = append(args, workspaceID)
	}
	if taskID != "" {
		q += " AND task_id = ?"
		args = append(args, taskID)
	}
	if status != "" {
		q += " AND status = ?"
		args = append(args, status)
	} else {
		q += " AND status != 'archived'"
	}
	q += " ORDER BY updated_at_ms DESC LIMIT ?"
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.AgentSession
	for rows.Next() {
		sess, err := scanAgentSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}

// SetAgentSessionWorktree binds a session to its persistent worktree.
func (s *Store) SetAgentSessionWorktree(ctx context.Context, id, worktreeID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE agent_sessions SET worktree_id = ?, updated_at_ms = ? WHERE id = ?`,
		worktreeID, ms(time.Now()), id)
	return err
}

// ClearAgentSessionContext removes the one-shot transcript seed.
func (s *Store) ClearAgentSessionContext(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE agent_sessions SET seed_context = '', updated_at_ms = ? WHERE id = ?`,
		ms(time.Now()), id)
	return err
}

// RecordSessionRun links a run to its session and stores the backend session id.
func (s *Store) RecordSessionRun(ctx context.Context, id, backendSessionID, runID string) error {
	if backendSessionID == "" {
		_, err := s.db.ExecContext(ctx, `UPDATE agent_sessions SET last_run_id = ?, status = 'active', updated_at_ms = ? WHERE id = ?`,
			runID, ms(time.Now()), id)
		return err
	}
	_, err := s.db.ExecContext(ctx, `UPDATE agent_sessions SET backend_session_id = ?, last_run_id = ?, status = 'active', updated_at_ms = ? WHERE id = ?`,
		backendSessionID, runID, ms(time.Now()), id)
	return err
}

// SetAgentSessionStatus updates lifecycle state.
func (s *Store) SetAgentSessionStatus(ctx context.Context, id, status string) (*domain.AgentSession, error) {
	now := time.Now().UTC()
	q := `UPDATE agent_sessions SET status = ?, updated_at_ms = ? WHERE id = ?`
	args := []any{status, ms(now), id}
	if status == "closed" {
		q = `UPDATE agent_sessions SET status = ?, closed_at_ms = ?, updated_at_ms = ? WHERE id = ?`
		args = []any{status, ms(now), ms(now), id}
	}
	if _, err := s.db.ExecContext(ctx, q, args...); err != nil {
		return nil, err
	}
	return s.GetAgentSession(ctx, id)
}

// ListRunsBySession returns runs attached to a session, oldest first.
func (s *Store) ListRunsBySession(ctx context.Context, sessionID string, limit int) ([]*domain.Run, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+runColumns+` FROM runs WHERE session_id = ? ORDER BY created_at_ms ASC LIMIT ?`, sessionID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Run
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CountSessionRuns returns the number of runs in a session.
func (s *Store) CountSessionRuns(ctx context.Context, sessionID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM runs WHERE session_id = ?`, sessionID).Scan(&n)
	return n, err
}

func scanAgentSession(sc interface{ Scan(...any) error }) (*domain.AgentSession, error) {
	var (
		sess    domain.AgentSession
		last    sql.NullString
		created int64
		updated int64
		closed  sql.NullInt64
	)
	err := sc.Scan(&sess.ID, &sess.WorkspaceID, &sess.TaskID, &sess.AgentID, &sess.Backend, &sess.Model, &sess.Reasoning, &sess.Mode, &sess.Title,
		&sess.Repo, &sess.Base, &sess.WorktreeID, &sess.Status, &last, &sess.BackendSessionID, &sess.LastRunID, &sess.CreatedBy, &created, &updated, &sess.Context, &closed)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if last.Valid {
		sess.LastRunStatus = last.String
	}
	sess.CreatedAt = fromMS(created)
	sess.UpdatedAt = fromMS(updated)
	if closed.Valid {
		t := fromMS(closed.Int64)
		sess.ClosedAt = &t
	}
	return &sess, nil
}

// RenameAgentSession updates a session title.
func (s *Store) RenameAgentSession(ctx context.Context, id, title string) (*domain.AgentSession, error) {
	if _, err := s.db.ExecContext(ctx, `UPDATE agent_sessions SET title = ?, updated_at_ms = ? WHERE id = ?`,
		title, ms(time.Now()), id); err != nil {
		return nil, err
	}
	return s.GetAgentSession(ctx, id)
}

// ArchiveAgentSession hides a session without deleting its history.
func (s *Store) ArchiveAgentSession(ctx context.Context, id string) (*domain.AgentSession, error) {
	if _, err := s.db.ExecContext(ctx, `UPDATE agent_sessions SET status = 'archived', updated_at_ms = ? WHERE id = ?`,
		ms(time.Now()), id); err != nil {
		return nil, err
	}
	return s.GetAgentSession(ctx, id)
}

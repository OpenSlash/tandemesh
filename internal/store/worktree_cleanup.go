package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// ExpiredWorktrees lists worktrees older than cutoff that are not yet archived.
func (s *Store) ExpiredWorktrees(ctx context.Context, workspaceID string, cutoff time.Time, limit int) ([]*domain.Worktree, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	query := `SELECT id, workspace_id, task_id, run_id, repo, path, branch, base_ref, start_ref, status,
		checkpoint_comment, workspace_status, created_at_ms, updated_at_ms, archived_at_ms
		FROM worktrees WHERE status NOT IN ('archived','removed') AND created_at_ms <= ?`
	args := []any{ms(cutoff)}
	if workspaceID != "" {
		query += ` AND workspace_id = ?`
		args = append(args, workspaceID)
	}
	query += ` ORDER BY created_at_ms LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.Worktree{}
	for rows.Next() {
		wt, err := scanWorktree(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, wt)
	}
	return out, rows.Err()
}

// EnqueueWorktreeCleanup creates a pending physical cleanup request.
func (s *Store) EnqueueWorktreeCleanup(ctx context.Context, wt *domain.Worktree, machineID string) (*domain.WorktreeCleanup, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, worktree_id, run_id, path, repo, machine_id, status, attempts, error, created_at_ms, completed_at_ms
		FROM worktree_cleanups WHERE worktree_id = ? AND status = 'pending' LIMIT 1`, wt.ID)
	if existing, err := scanWorktreeCleanup(row); err == nil && existing != nil {
		return existing, nil
	}
	c := &domain.WorktreeCleanup{
		ID:         "wtc_" + ulid.Make().String(),
		WorktreeID: wt.ID,
		RunID:      wt.RunID,
		Path:       wt.Path,
		Repo:       wt.Repo,
		MachineID:  machineID,
		Status:     "pending",
		CreatedAt:  time.Now().UTC(),
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO worktree_cleanups
		(id, worktree_id, run_id, path, repo, machine_id, status, attempts, error, created_at_ms, completed_at_ms)
		VALUES (?,?,?,?,?,?,?,0,'',?,NULL)`,
		c.ID, c.WorktreeID, c.RunID, c.Path, c.Repo, c.MachineID, c.Status, ms(c.CreatedAt))
	if err != nil {
		return nil, err
	}
	return c, nil
}

// GetWorktreeCleanup returns one cleanup request or nil.
func (s *Store) GetWorktreeCleanup(ctx context.Context, id string) (*domain.WorktreeCleanup, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, worktree_id, run_id, path, repo, machine_id, status, attempts, error, created_at_ms, completed_at_ms
		FROM worktree_cleanups WHERE id = ?`, id)
	return scanWorktreeCleanup(row)
}

// PendingWorktreeCleanups returns cleanup work for a machine.
func (s *Store) PendingWorktreeCleanups(ctx context.Context, machineID string, limit int) ([]*domain.WorktreeCleanup, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, worktree_id, run_id, path, repo, machine_id, status, attempts, error, created_at_ms, completed_at_ms
		FROM worktree_cleanups WHERE status = 'pending' AND machine_id = ? ORDER BY created_at_ms LIMIT ?`, machineID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.WorktreeCleanup
	for rows.Next() {
		c, err := scanWorktreeCleanup(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// FinishWorktreeCleanup records the result of a physical cleanup.
func (s *Store) FinishWorktreeCleanup(ctx context.Context, id, status, errMsg string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE worktree_cleanups SET status = ?, attempts = attempts + 1, error = ?, completed_at_ms = ? WHERE id = ?`,
		status, errMsg, ms(time.Now().UTC()), id)
	return err
}

func scanWorktreeCleanup(sc interface{ Scan(...any) error }) (*domain.WorktreeCleanup, error) {
	var (
		c         domain.WorktreeCleanup
		created   int64
		completed sql.NullInt64
	)
	if err := sc.Scan(&c.ID, &c.WorktreeID, &c.RunID, &c.Path, &c.Repo, &c.MachineID, &c.Status, &c.Attempts, &c.Error, &created, &completed); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	c.CreatedAt = fromMS(created)
	if completed.Valid {
		ts := fromMS(completed.Int64)
		c.CompletedAt = &ts
	}
	return &c, nil
}

// LatestWorktreeCleanup returns the most recent cleanup for a worktree.
func (s *Store) LatestWorktreeCleanup(ctx context.Context, worktreeID string) (*domain.WorktreeCleanup, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, worktree_id, run_id, path, repo, machine_id, status, attempts, error, created_at_ms, completed_at_ms
		FROM worktree_cleanups WHERE worktree_id = ? ORDER BY created_at_ms DESC LIMIT 1`, worktreeID)
	return scanWorktreeCleanup(row)
}

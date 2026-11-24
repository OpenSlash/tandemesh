package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// CreateWorktree records the intent to prepare a worktree for a run.
func (s *Store) CreateWorktree(ctx context.Context, wt *domain.Worktree) (*domain.Worktree, error) {
	if wt.ID == "" {
		wt.ID = "wt_" + ulid.Make().String()
	}
	now := time.Now().UTC()
	if wt.CreatedAt.IsZero() {
		wt.CreatedAt = now
	}
	wt.UpdatedAt = now
	if wt.Status == "" {
		wt.Status = "creating"
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO worktrees
		(id, workspace_id, task_id, run_id, repo, path, branch, base_ref, start_ref, status, checkpoint_comment, workspace_status, created_at_ms, updated_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		wt.ID, wt.WorkspaceID, wt.TaskID, wt.RunID, wt.Repo, wt.Path, wt.Branch, wt.BaseRef, wt.StartRef,
		wt.Status, wt.CheckpointComment, wt.WorkspaceStatus, ms(wt.CreatedAt), ms(wt.UpdatedAt))
	if err != nil {
		return nil, err
	}
	return wt, nil
}

// MarkWorktreeReady upserts the worktree report emitted by an agent.
func (s *Store) MarkWorktreeReady(ctx context.Context, runID, path, branch string) error {
	run, err := s.GetRun(ctx, runID)
	if err != nil {
		return err
	}
	if run == nil {
		return sql.ErrNoRows
	}
	now := time.Now().UTC()
	id := "wt_" + ulid.Make().String()
	_, err = s.db.ExecContext(ctx, `INSERT INTO worktrees
		(id, workspace_id, task_id, run_id, repo, path, branch, base_ref, start_ref, status, created_at_ms, updated_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(run_id) DO UPDATE SET path = excluded.path, branch = excluded.branch, status = 'ready', updated_at_ms = excluded.updated_at_ms`,
		id, run.WorkspaceID, run.TaskID, runID, run.Repo, path, branch, run.Base, run.Base, "ready", ms(now), ms(now))
	return err
}

// GetWorktree returns one worktree or nil.
func (s *Store) GetWorktree(ctx context.Context, id string) (*domain.Worktree, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, workspace_id, task_id, run_id, repo, path, branch, base_ref, start_ref,
		status, checkpoint_comment, workspace_status, created_at_ms, updated_at_ms, archived_at_ms FROM worktrees WHERE id = ?`, id)
	return scanWorktree(row)
}

// GetWorktreeByRun returns the worktree for a run or nil.
func (s *Store) GetWorktreeByRun(ctx context.Context, runID string) (*domain.Worktree, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, workspace_id, task_id, run_id, repo, path, branch, base_ref, start_ref,
		status, checkpoint_comment, workspace_status, created_at_ms, updated_at_ms, archived_at_ms FROM worktrees WHERE run_id = ?`, runID)
	return scanWorktree(row)
}

// ListWorktrees returns worktrees filtered by workspace, task or status.
func (s *Store) ListWorktrees(ctx context.Context, workspaceID, taskID, repo, status string, limit int) ([]*domain.Worktree, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	q := `SELECT id, workspace_id, task_id, run_id, repo, path, branch, base_ref, start_ref,
		status, checkpoint_comment, workspace_status, created_at_ms, updated_at_ms, archived_at_ms FROM worktrees WHERE 1=1`
	var args []any
	if workspaceID != "" {
		q += " AND workspace_id = ?"
		args = append(args, workspaceID)
	}
	if taskID != "" {
		q += " AND task_id = ?"
		args = append(args, taskID)
	}
	if repo != "" {
		q += " AND repo = ?"
		args = append(args, repo)
	}
	if status != "" {
		q += " AND status = ?"
		args = append(args, status)
	}
	q += " ORDER BY created_at_ms DESC LIMIT ?"
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Worktree
	for rows.Next() {
		wt, err := scanWorktree(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, wt)
	}
	return out, rows.Err()
}

// SetWorktreeCheckpoint records an agent's free-text progress snapshot.
func (s *Store) SetWorktreeCheckpoint(ctx context.Context, id, comment, workspaceStatus string) (*domain.Worktree, error) {
	_, err := s.db.ExecContext(ctx, `UPDATE worktrees SET checkpoint_comment = ?, workspace_status = ?, updated_at_ms = ? WHERE id = ?`,
		comment, workspaceStatus, ms(time.Now()), id)
	if err != nil {
		return nil, err
	}
	return s.GetWorktree(ctx, id)
}

// SetWorktreeStatus updates lifecycle state.
func (s *Store) SetWorktreeStatus(ctx context.Context, id, status string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE worktrees SET status = ?, updated_at_ms = ? WHERE id = ?`,
		status, ms(time.Now()), id)
	return err
}

// ArchiveWorktree marks a worktree archived.
func (s *Store) ArchiveWorktree(ctx context.Context, id string) (*domain.Worktree, error) {
	now := time.Now().UTC()
	if _, err := s.db.ExecContext(ctx, `UPDATE worktrees SET status = 'archived', archived_at_ms = ?, updated_at_ms = ? WHERE id = ?`,
		ms(now), ms(now), id); err != nil {
		return nil, err
	}
	return s.GetWorktree(ctx, id)
}

func scanWorktree(sc interface{ Scan(...any) error }) (*domain.Worktree, error) {
	var (
		wt       domain.Worktree
		created  int64
		updated  int64
		archived sql.NullInt64
	)
	err := sc.Scan(&wt.ID, &wt.WorkspaceID, &wt.TaskID, &wt.RunID, &wt.Repo, &wt.Path, &wt.Branch, &wt.BaseRef, &wt.StartRef,
		&wt.Status, &wt.CheckpointComment, &wt.WorkspaceStatus, &created, &updated, &archived)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	wt.CreatedAt = fromMS(created)
	wt.UpdatedAt = fromMS(updated)
	if archived.Valid {
		t := fromMS(archived.Int64)
		wt.ArchivedAt = &t
	}
	return &wt, nil
}

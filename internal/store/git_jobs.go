package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// CreateGitJob queues a remote git operation for a session worktree.
func (s *Store) CreateGitJob(ctx context.Context, job *domain.GitJob) (*domain.GitJob, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, session_id, worktree_id, machine_id, path, action, message, status, result_json, error, created_at_ms, updated_at_ms, completed_at_ms
		FROM git_jobs WHERE session_id = ? AND status = 'pending' LIMIT 1`, job.SessionID)
	if existing, err := scanGitJob(row); err == nil && existing != nil {
		return existing, nil
	}
	if job.ID == "" {
		job.ID = "git_" + ulid.Make().String()
	}
	now := time.Now().UTC()
	job.Status = "pending"
	job.CreatedAt, job.UpdatedAt = now, now
	_, err := s.db.ExecContext(ctx, `INSERT INTO git_jobs
		(id, session_id, worktree_id, machine_id, path, action, message, status, result_json, error, created_at_ms, updated_at_ms, completed_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,NULL)`,
		job.ID, job.SessionID, job.WorktreeID, job.MachineID, job.Path, job.Action, job.Message,
		job.Status, "", "", ms(job.CreatedAt), ms(job.UpdatedAt))
	if err != nil {
		return nil, err
	}
	return job, nil
}

// GetGitJob returns one git job or nil.
func (s *Store) GetGitJob(ctx context.Context, id string) (*domain.GitJob, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, session_id, worktree_id, machine_id, path, action, message, status, result_json, error, created_at_ms, updated_at_ms, completed_at_ms
		FROM git_jobs WHERE id = ?`, id)
	return scanGitJob(row)
}

// PendingGitJobs returns queued git work for a machine.
func (s *Store) PendingGitJobs(ctx context.Context, machineID string, limit int) ([]*domain.GitJob, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, session_id, worktree_id, machine_id, path, action, message, status, result_json, error, created_at_ms, updated_at_ms, completed_at_ms
		FROM git_jobs WHERE status = 'pending' AND machine_id = ? ORDER BY created_at_ms LIMIT ?`, machineID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.GitJob{}
	for rows.Next() {
		job, err := scanGitJob(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, job)
	}
	return out, rows.Err()
}

// FinishGitJob records the outcome of a remote git operation.
func (s *Store) FinishGitJob(ctx context.Context, id, status, resultJSON, errMsg string) error {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `UPDATE git_jobs SET status = ?, result_json = ?, error = ?, updated_at_ms = ?, completed_at_ms = ? WHERE id = ?`,
		status, resultJSON, errMsg, ms(now), ms(now), id)
	return err
}

func scanGitJob(sc interface{ Scan(...any) error }) (*domain.GitJob, error) {
	var (
		job       domain.GitJob
		created   int64
		updated   int64
		completed sql.NullInt64
	)
	if err := sc.Scan(&job.ID, &job.SessionID, &job.WorktreeID, &job.MachineID, &job.Path, &job.Action,
		&job.Message, &job.Status, &job.ResultJSON, &job.Error, &created, &updated, &completed); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	job.CreatedAt = fromMS(created)
	job.UpdatedAt = fromMS(updated)
	if completed.Valid {
		ts := fromMS(completed.Int64)
		job.CompletedAt = &ts
	}
	return &job, nil
}

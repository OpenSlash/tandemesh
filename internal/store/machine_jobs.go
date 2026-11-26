package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

const machineJobColumns = `id, machine_id, kind, payload_json, status, result_json, error, requested_at_ms, started_at_ms, completed_at_ms`

// CreateMachineJob queues work for one host.
func (s *Store) CreateMachineJob(ctx context.Context, machineID, kind string, payload any) (*domain.MachineJob, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode machine job payload: %w", err)
	}
	now := time.Now().UTC()
	job := &domain.MachineJob{
		ID: "mjb_" + ulid.Make().String(), MachineID: machineID, Kind: kind,
		Payload: raw, Status: "pending", RequestedAt: now,
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO machine_jobs (id, machine_id, kind, payload_json, status, requested_at_ms)
		VALUES (?,?,?,?, 'pending', ?)`, job.ID, machineID, kind, string(raw), ms(now)); err != nil {
		return nil, fmt.Errorf("create machine job: %w", err)
	}
	return job, nil
}

// ClaimMachineJobs marks up to limit pending jobs of a host as running and
// returns them. A job is handed out once even if the host polls concurrently.
func (s *Store) ClaimMachineJobs(ctx context.Context, machineID string, limit int) ([]*domain.MachineJob, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+machineJobColumns+` FROM machine_jobs
		WHERE machine_id = ? AND status = 'pending' ORDER BY requested_at_ms LIMIT ?`, machineID, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending machine jobs: %w", err)
	}
	var pending []*domain.MachineJob
	for rows.Next() {
		job, err := scanMachineJob(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		pending = append(pending, job)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	out := []*domain.MachineJob{}
	for _, job := range pending {
		res, err := s.db.ExecContext(ctx, `UPDATE machine_jobs SET status = 'running', started_at_ms = ? WHERE id = ? AND status = 'pending'`, ms(now), job.ID)
		if err != nil {
			return nil, fmt.Errorf("claim machine job: %w", err)
		}
		if n, _ := res.RowsAffected(); n == 1 {
			job.Status, job.StartedAt = "running", &now
			out = append(out, job)
		}
	}
	return out, nil
}

// FinishMachineJob records a host's result for a job it claimed.
func (s *Store) FinishMachineJob(ctx context.Context, id, machineID, status string, result json.RawMessage, errMsg string) error {
	if len(result) == 0 {
		result = json.RawMessage("{}")
	}
	res, err := s.db.ExecContext(ctx, `UPDATE machine_jobs SET status = ?, result_json = ?, error = ?, completed_at_ms = ?
		WHERE id = ? AND machine_id = ? AND status = 'running'`, status, string(result), errMsg, ms(time.Now()), id, machineID)
	if err != nil {
		return fmt.Errorf("finish machine job: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetMachineJob returns one job or nil.
func (s *Store) GetMachineJob(ctx context.Context, id string) (*domain.MachineJob, error) {
	job, err := scanMachineJob(s.db.QueryRowContext(ctx, `SELECT `+machineJobColumns+` FROM machine_jobs WHERE id = ?`, id))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return job, err
}

func scanMachineJob(row rowScanner) (*domain.MachineJob, error) {
	var (
		job                domain.MachineJob
		payload, result    string
		requested          int64
		started, completed sql.NullInt64
	)
	if err := row.Scan(&job.ID, &job.MachineID, &job.Kind, &payload, &job.Status, &result, &job.Error, &requested, &started, &completed); err != nil {
		return nil, err
	}
	job.Payload, job.Result = json.RawMessage(payload), json.RawMessage(result)
	job.RequestedAt = fromMS(requested)
	if started.Valid {
		t := fromMS(started.Int64)
		job.StartedAt = &t
	}
	if completed.Valid {
		t := fromMS(completed.Int64)
		job.CompletedAt = &t
	}
	return &job, nil
}

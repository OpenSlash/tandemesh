package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// ErrOpenGates is returned when a dispatch cannot complete while a decision
// gate is still open.
var ErrOpenGates = errors.New("decision gates are still open")

// ErrDispatchDone is returned when a dispatch has already reached a terminal state.
var ErrDispatchDone = errors.New("dispatch already completed")

// CreateDispatch records a new execution attempt for a task.
func (s *Store) CreateDispatch(ctx context.Context, task *domain.Task, runID, agentID string, attempt int) (*domain.Dispatch, error) {
	if attempt <= 0 {
		attempt = 1
	}
	now := time.Now().UTC()
	failures, err := s.ConsecutiveDispatchFailures(ctx, task.ID)
	if err != nil {
		return nil, err
	}
	d := &domain.Dispatch{
		ID:           "dsp_" + ulid.Make().String(),
		TaskID:       task.ID,
		WorkspaceID:  task.WorkspaceID,
		RunID:        runID,
		AgentID:      agentID,
		Attempt:      attempt,
		Status:       domain.DispatchPending,
		FailureCount: failures,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO dispatches
		(id, task_id, workspace_id, run_id, agent_id, attempt, status, failure_count, created_at_ms, updated_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		d.ID, d.TaskID, d.WorkspaceID, d.RunID, d.AgentID, d.Attempt, d.Status, d.FailureCount, ms(now), ms(now))
	if err != nil {
		return nil, err
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE tasks SET execution_state = ?, updated_at_ms = ? WHERE id = ?`,
		domain.TaskExecutionDispatched, ms(now), task.ID)
	return d, nil
}

// ConsecutiveDispatchFailures counts failed dispatches back to the last success.
func (s *Store) ConsecutiveDispatchFailures(ctx context.Context, taskID string) (int, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT status FROM dispatches WHERE task_id = ? ORDER BY created_at_ms DESC LIMIT 20`, taskID)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			return 0, err
		}
		switch status {
		case domain.DispatchFailed, domain.DispatchCircuitBroken:
			count++
		default:
			return count, nil
		}
	}
	return count, rows.Err()
}

// GetDispatch returns one dispatch or nil.
func (s *Store) GetDispatch(ctx context.Context, id string) (*domain.Dispatch, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, task_id, workspace_id, run_id, agent_id, attempt, status,
		outcome, result, files_modified, failure_count, created_at_ms, updated_at_ms, ended_at_ms
		FROM dispatches WHERE id = ?`, id)
	return scanDispatch(row)
}

// ListDispatchesByTask returns attempts newest first.
func (s *Store) ListDispatchesByTask(ctx context.Context, taskID string) ([]*domain.Dispatch, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, task_id, workspace_id, run_id, agent_id, attempt, status,
		outcome, result, files_modified, failure_count, created_at_ms, updated_at_ms, ended_at_ms
		FROM dispatches WHERE task_id = ? ORDER BY created_at_ms DESC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Dispatch
	for rows.Next() {
		d, err := scanDispatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ActiveDispatchByRun returns the newest non-terminal dispatch for a run.
func (s *Store) ActiveDispatchByRun(ctx context.Context, runID string) (*domain.Dispatch, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, task_id, workspace_id, run_id, agent_id, attempt, status,
		outcome, result, files_modified, failure_count, created_at_ms, updated_at_ms, ended_at_ms
		FROM dispatches WHERE run_id = ? AND status IN ('pending','running') ORDER BY created_at_ms DESC LIMIT 1`, runID)
	return scanDispatch(row)
}

func scanDispatch(sc interface{ Scan(...any) error }) (*domain.Dispatch, error) {
	var (
		d       domain.Dispatch
		files   string
		created int64
		updated int64
		ended   sql.NullInt64
	)
	err := sc.Scan(&d.ID, &d.TaskID, &d.WorkspaceID, &d.RunID, &d.AgentID, &d.Attempt, &d.Status,
		&d.Outcome, &d.Result, &files, &d.FailureCount, &created, &updated, &ended)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	d.CreatedAt = fromMS(created)
	d.UpdatedAt = fromMS(updated)
	if ended.Valid {
		t := fromMS(ended.Int64)
		d.EndedAt = &t
	}
	_ = json.Unmarshal([]byte(files), &d.FilesModified)
	return &d, nil
}

// StartDispatch marks a pending dispatch as running.
func (s *Store) StartDispatch(ctx context.Context, id string) error {
	now := ms(time.Now())
	_, err := s.db.ExecContext(ctx, `UPDATE dispatches SET status = ?, updated_at_ms = ? WHERE id = ? AND status = ?`,
		domain.DispatchRunning, now, id, domain.DispatchPending)
	return err
}

// CompleteDispatch applies an outcome atomically to the dispatch and its task,
// then promotes dependents that are now unblocked.
func (s *Store) CompleteDispatch(ctx context.Context, id, taskID, outcome, result string, files []string) (*domain.Task, []string, error) {
	d, err := s.GetDispatch(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if d == nil {
		return nil, nil, sql.ErrNoRows
	}
	if taskID != "" && d.TaskID != taskID {
		return nil, nil, fmt.Errorf("dispatch %s does not belong to task %s", id, taskID)
	}
	if d.Status == domain.DispatchSucceeded || d.Status == domain.DispatchFailed || d.Status == domain.DispatchCircuitBroken {
		t, _ := s.GetTask(ctx, d.TaskID)
		return t, nil, ErrDispatchDone
	}
	gates, err := s.OpenGateCount(ctx, d.TaskID)
	if err != nil {
		return nil, nil, err
	}
	if gates > 0 {
		return nil, nil, ErrOpenGates
	}

	now := time.Now().UTC()
	status := domain.DispatchSucceeded
	taskStatus := domain.TaskDone
	execState := domain.TaskExecutionCompleted
	if strings.EqualFold(outcome, "failed") {
		status = domain.DispatchFailed
		taskStatus = domain.TaskOpen
		execState = domain.TaskExecutionFailed
	}
	filesJSON, _ := json.Marshal(files)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, nil, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE dispatches SET status = ?, outcome = ?, result = ?, files_modified = ?,
		updated_at_ms = ?, ended_at_ms = ? WHERE id = ?`, status, strings.ToLower(outcome), result, string(filesJSON), ms(now), ms(now), id); err != nil {
		return nil, nil, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE tasks SET status = ?, execution_state = ?, done_at_ms = ?, updated_at_ms = ? WHERE id = ?`,
		taskStatus, execState, func() any {
			if taskStatus == domain.TaskDone {
				return ms(now)
			}
			return nil
		}(), ms(now), d.TaskID); err != nil {
		return nil, nil, err
	}
	var promoted []string
	if taskStatus == domain.TaskDone {
		rows, err := tx.QueryContext(ctx, `SELECT d.task_id FROM task_dependencies d JOIN tasks t ON t.id = d.task_id
			WHERE d.depends_on = ? AND t.status != ? AND t.execution_state = ?
			AND NOT EXISTS (SELECT 1 FROM task_dependencies d2 JOIN tasks dep ON dep.id = d2.depends_on
				WHERE d2.task_id = d.task_id AND dep.status != ?)`,
			d.TaskID, domain.TaskDone, domain.TaskExecutionPending, domain.TaskDone)
		if err != nil {
			return nil, nil, err
		}
		for rows.Next() {
			var pid string
			if err := rows.Scan(&pid); err != nil {
				rows.Close()
				return nil, nil, err
			}
			promoted = append(promoted, pid)
		}
		rows.Close()
		for _, pid := range promoted {
			if _, err := tx.ExecContext(ctx, `UPDATE tasks SET execution_state = ?, updated_at_ms = ? WHERE id = ?`,
				domain.TaskExecutionReady, ms(now), pid); err != nil {
				return nil, nil, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, nil, err
	}
	task, _ := s.GetTask(ctx, d.TaskID)
	return task, promoted, nil
}

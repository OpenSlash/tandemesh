package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// CreateGate opens a decision gate that blocks task completion.
func (s *Store) CreateGate(ctx context.Context, taskID, dispatchID, question string, options []string) (*domain.DecisionGate, error) {
	now := time.Now().UTC()
	opts, _ := json.Marshal(options)
	g := &domain.DecisionGate{
		ID:         "gate_" + ulid.Make().String(),
		TaskID:     taskID,
		DispatchID: dispatchID,
		Question:   question,
		Options:    options,
		Status:     "open",
		CreatedAt:  now,
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO decision_gates
		(id, task_id, dispatch_id, question, options, status, created_at_ms) VALUES (?,?,?,?,?,?,?)`,
		g.ID, g.TaskID, g.DispatchID, g.Question, string(opts), g.Status, ms(now))
	if err != nil {
		return nil, err
	}
	return g, nil
}

// ResolveGate records a gate resolution.
func (s *Store) ResolveGate(ctx context.Context, id, resolution string) (*domain.DecisionGate, error) {
	now := time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `UPDATE decision_gates SET resolution = ?, status = 'resolved', resolved_at_ms = ? WHERE id = ? AND status = 'open'`,
		resolution, ms(now), id)
	if err != nil {
		return nil, err
	}
	return s.GetGate(ctx, id)
}

// GetGate returns one gate or nil.
func (s *Store) GetGate(ctx context.Context, id string) (*domain.DecisionGate, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, task_id, dispatch_id, question, options, resolution, status, created_at_ms, resolved_at_ms
		FROM decision_gates WHERE id = ?`, id)
	return scanGate(row)
}

// ListGates returns gates for a task, open first.
func (s *Store) ListGates(ctx context.Context, taskID string) ([]*domain.DecisionGate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, task_id, dispatch_id, question, options, resolution, status, created_at_ms, resolved_at_ms
		FROM decision_gates WHERE task_id = ? ORDER BY status = 'open' DESC, created_at_ms DESC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.DecisionGate
	for rows.Next() {
		g, err := scanGate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// OpenGateCount counts unresolved gates for a task.
func (s *Store) OpenGateCount(ctx context.Context, taskID string) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM decision_gates WHERE task_id = ? AND status = 'open'`, taskID).Scan(&n)
	return n, err
}

func scanGate(sc interface{ Scan(...any) error }) (*domain.DecisionGate, error) {
	var (
		g        domain.DecisionGate
		opts     string
		created  int64
		resolved sql.NullInt64
	)
	err := sc.Scan(&g.ID, &g.TaskID, &g.DispatchID, &g.Question, &opts, &g.Resolution, &g.Status, &created, &resolved)
	if err != nil {
		return nil, err
	}
	g.CreatedAt = fromMS(created)
	if resolved.Valid {
		t := fromMS(resolved.Int64)
		g.ResolvedAt = &t
	}
	_ = json.Unmarshal([]byte(opts), &g.Options)
	return &g, nil
}

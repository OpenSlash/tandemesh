package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// AddDiffAnnotation stores one line-anchored review note.
func (s *Store) AddDiffAnnotation(ctx context.Context, a *domain.DiffAnnotation) error {
	if a.ID == "" {
		a.ID = "ann_" + ulid.Make().String()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	if a.Status == "" {
		a.Status = "open"
	}
	if a.Side == "" {
		a.Side = "new"
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO diff_annotations
		(id, workspace_id, task_id, run_id, file_path, line, side, body, author, status, created_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		a.ID, a.WorkspaceID, a.TaskID, a.RunID, a.FilePath, a.Line, a.Side, a.Body, a.Author, a.Status, ms(a.CreatedAt))
	return err
}

// ListDiffAnnotations returns annotations, optionally filtered.
func (s *Store) ListDiffAnnotations(ctx context.Context, taskID, runID, status string) ([]*domain.DiffAnnotation, error) {
	q := `SELECT id, workspace_id, task_id, run_id, file_path, line, side, body, author, status, batch_id,
		created_at_ms, sent_at_ms, resolved_at_ms FROM diff_annotations WHERE 1=1`
	var args []any
	if taskID != "" {
		q += " AND task_id = ?"
		args = append(args, taskID)
	}
	if runID != "" {
		q += " AND run_id = ?"
		args = append(args, runID)
	}
	if status != "" {
		q += " AND status = ?"
		args = append(args, status)
	}
	q += " ORDER BY file_path, line, created_at_ms"
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.DiffAnnotation
	for rows.Next() {
		a, err := scanDiffAnnotation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ResolveDiffAnnotation closes one annotation.
func (s *Store) ResolveDiffAnnotation(ctx context.Context, id string) (*domain.DiffAnnotation, error) {
	now := time.Now().UTC()
	if _, err := s.db.ExecContext(ctx, `UPDATE diff_annotations SET status = 'resolved', resolved_at_ms = ? WHERE id = ?`,
		ms(now), id); err != nil {
		return nil, err
	}
	return s.GetDiffAnnotation(ctx, id)
}

// GetDiffAnnotation returns one annotation or nil.
func (s *Store) GetDiffAnnotation(ctx context.Context, id string) (*domain.DiffAnnotation, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, workspace_id, task_id, run_id, file_path, line, side, body, author, status, batch_id,
		created_at_ms, sent_at_ms, resolved_at_ms FROM diff_annotations WHERE id = ?`, id)
	return scanDiffAnnotation(row)
}

// MarkDiffAnnotationsSent groups open notes into one review batch.
func (s *Store) MarkDiffAnnotationsSent(ctx context.Context, ids []string, batchID string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	now := ms(time.Now())
	n := 0
	for _, id := range ids {
		res, err := s.db.ExecContext(ctx, `UPDATE diff_annotations SET batch_id = ?, sent_at_ms = ? WHERE id = ? AND status = 'open'`,
			batchID, now, id)
		if err != nil {
			return n, err
		}
		affected, _ := res.RowsAffected()
		n += int(affected)
	}
	return n, nil
}

func scanDiffAnnotation(sc interface{ Scan(...any) error }) (*domain.DiffAnnotation, error) {
	var (
		a        domain.DiffAnnotation
		created  int64
		sent     sql.NullInt64
		resolved sql.NullInt64
	)
	err := sc.Scan(&a.ID, &a.WorkspaceID, &a.TaskID, &a.RunID, &a.FilePath, &a.Line, &a.Side, &a.Body,
		&a.Author, &a.Status, &a.BatchID, &created, &sent, &resolved)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a.CreatedAt = fromMS(created)
	if sent.Valid {
		t := fromMS(sent.Int64)
		a.SentAt = &t
	}
	if resolved.Valid {
		t := fromMS(resolved.Int64)
		a.ResolvedAt = &t
	}
	return &a, nil
}

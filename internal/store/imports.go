package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// CreateImportBatch records a new import run.
func (s *Store) CreateImportBatch(ctx context.Context, batch *domain.ImportBatch) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO import_batches
		(id, workspace_id, provider, connection_id, query, status, imported, next_start_at, limit_count, failed, error, created_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		batch.ID, batch.WorkspaceID, batch.Provider, batch.ConnectionID, batch.Query,
		batch.Status, batch.Imported, batch.NextStartAt, batch.Limit, batch.Failed, batch.Error, ms(batch.CreatedAt))
	return err
}

// FinishImportBatch updates the final import counters.
func (s *Store) FinishImportBatch(ctx context.Context, batch *domain.ImportBatch) error {
	batch.FinishedAt = ptrTime(time.Now().UTC())
	_, err := s.db.ExecContext(ctx, `UPDATE import_batches SET status = ?, imported = ?, next_start_at = ?, failed = ?,
		error = ?, finished_at_ms = ? WHERE id = ?`,
		batch.Status, batch.Imported, batch.NextStartAt, batch.Failed, batch.Error, ms(*batch.FinishedAt), batch.ID)
	return err
}

// ListImportBatches returns import runs for a workspace.
func (s *Store) ListImportBatches(ctx context.Context, workspaceID string, limit int) ([]*domain.ImportBatch, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, provider, connection_id, query, status,
		imported, next_start_at, limit_count, failed, error, created_at_ms, finished_at_ms FROM import_batches
		WHERE workspace_id = ? ORDER BY created_at_ms DESC LIMIT ?`, workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.ImportBatch
	for rows.Next() {
		var (
			batch    domain.ImportBatch
			created  int64
			finished sql.NullInt64
		)
		if err := rows.Scan(&batch.ID, &batch.WorkspaceID, &batch.Provider, &batch.ConnectionID,
			&batch.Query, &batch.Status, &batch.Imported, &batch.NextStartAt, &batch.Limit, &batch.Failed, &batch.Error, &created, &finished); err != nil {
			return nil, err
		}
		batch.CreatedAt = fromMS(created)
		batch.FinishedAt = fromNull(finished)
		out = append(out, &batch)
	}
	return out, rows.Err()
}

// GetImportBatch returns one batch or nil.
func (s *Store) GetImportBatch(ctx context.Context, id string) (*domain.ImportBatch, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, workspace_id, provider, connection_id, query, status,
		imported, next_start_at, limit_count, failed, error, created_at_ms, finished_at_ms FROM import_batches WHERE id = ?`, id)
	var (
		batch    domain.ImportBatch
		created  int64
		finished sql.NullInt64
	)
	err := row.Scan(&batch.ID, &batch.WorkspaceID, &batch.Provider, &batch.ConnectionID,
		&batch.Query, &batch.Status, &batch.Imported, &batch.NextStartAt, &batch.Limit, &batch.Failed, &batch.Error, &created, &finished)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	batch.CreatedAt = fromMS(created)
	batch.FinishedAt = fromNull(finished)
	return &batch, nil
}

// AddImportItem records a task created by an import batch.
func (s *Store) AddImportItem(ctx context.Context, batchID, taskID, workID string) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO import_items(batch_id, task_id, work_id)
		VALUES (?,?,?)`, batchID, taskID, workID)
	return err
}

// ListImportItems returns task ids created by a batch.
func (s *Store) ListImportItems(ctx context.Context, batchID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT task_id FROM import_items WHERE batch_id = ?`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// DeleteImportedTask removes a task only when no runs reference it.
func (s *Store) DeleteImportedTask(ctx context.Context, taskID string) (bool, error) {
	var runs int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM runs WHERE task_id = ?`, taskID).Scan(&runs); err != nil {
		return false, err
	}
	if runs > 0 {
		return false, nil
	}
	for _, table := range []string{"task_comments", "task_events", "task_relations", "task_inbox",
		"task_pull_requests", "task_commits", "task_attachments", "external_comments"} {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM `+table+` WHERE task_id = ?`, taskID); err != nil {
			return false, err
		}
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, taskID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func ptrTime(t time.Time) *time.Time { return &t }

// SetImportProgress stores the next page cursor and counters.
func (s *Store) SetImportProgress(ctx context.Context, id string, nextStartAt, imported, failed int) error {
	_, err := s.db.ExecContext(ctx, `UPDATE import_batches SET next_start_at = ?, imported = ?, failed = ?,
		status = 'running' WHERE id = ?`, nextStartAt, imported, failed, id)
	return err
}

// SetImportStatus updates only the batch status.
func (s *Store) SetImportStatus(ctx context.Context, id, status, errText string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE import_batches SET status = ?, error = ? WHERE id = ?`, status, errText, id)
	return err
}

// ListResumableImports returns batches that should continue after a restart.
func (s *Store) ListResumableImports(ctx context.Context) ([]*domain.ImportBatch, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, provider, connection_id, query, status,
		imported, next_start_at, limit_count, failed, error, created_at_ms, finished_at_ms FROM import_batches
		WHERE status IN ('queued', 'running') ORDER BY created_at_ms ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.ImportBatch
	for rows.Next() {
		var (
			batch    domain.ImportBatch
			created  int64
			finished sql.NullInt64
		)
		if err := rows.Scan(&batch.ID, &batch.WorkspaceID, &batch.Provider, &batch.ConnectionID,
			&batch.Query, &batch.Status, &batch.Imported, &batch.NextStartAt, &batch.Limit,
			&batch.Failed, &batch.Error, &created, &finished); err != nil {
			return nil, err
		}
		batch.CreatedAt = fromMS(created)
		batch.FinishedAt = fromNull(finished)
		out = append(out, &batch)
	}
	return out, rows.Err()
}

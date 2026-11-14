package store

import (
	"context"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// ReplaceTaskFields replaces the structured field projection for a task.
func (s *Store) ReplaceTaskFields(ctx context.Context, taskID, provider string, fields map[string][]string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM task_fields WHERE task_id = ?`, taskID); err != nil {
		return err
	}
	for key, values := range fields {
		for _, value := range values {
			if _, err := tx.ExecContext(ctx, `INSERT INTO task_fields(task_id, provider, key, value)
				VALUES (?,?,?,?)`, taskID, provider, key, value); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// ListTaskFields returns the structured fields for a task.
func (s *Store) ListTaskFields(ctx context.Context, taskID string) ([]*domain.TaskField, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT task_id, provider, key, value FROM task_fields
		WHERE task_id = ? ORDER BY key, value`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.TaskField
	for rows.Next() {
		var item domain.TaskField
		if err := rows.Scan(&item.TaskID, &item.Provider, &item.Key, &item.Value); err != nil {
			return nil, err
		}
		out = append(out, &item)
	}
	return out, rows.Err()
}

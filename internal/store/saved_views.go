package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// CreateSavedView inserts a named filter.
func (s *Store) CreateSavedView(ctx context.Context, v *domain.SavedView) error {
	filters, _ := json.Marshal(v.Filters)
	_, err := s.db.ExecContext(ctx, `INSERT INTO saved_views (id, workspace_id, name, filters, created_at_ms)
		VALUES (?,?,?,?,?)`, v.ID, v.WorkspaceID, v.Name, string(filters), ms(time.Now()))
	return err
}

// ListSavedViews returns views for a workspace.
func (s *Store) ListSavedViews(ctx context.Context, workspaceID string) ([]*domain.SavedView, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, name, filters, created_at_ms
		FROM saved_views WHERE workspace_id = ? ORDER BY created_at_ms`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.SavedView
	for rows.Next() {
		var (
			v       domain.SavedView
			filters string
			created int64
		)
		if err := rows.Scan(&v.ID, &v.WorkspaceID, &v.Name, &filters, &created); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(filters), &v.Filters)
		v.CreatedAt = fromMS(created)
		out = append(out, &v)
	}
	return out, rows.Err()
}

// RenameSavedView updates a view name.
func (s *Store) RenameSavedView(ctx context.Context, id, name string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE saved_views SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// DeleteSavedView removes a view.
func (s *Store) DeleteSavedView(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM saved_views WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

package store

import (
	"context"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// SearchGroups matches groups by name or description.
func (s *Store) SearchGroups(ctx context.Context, workspaceID, query string, limit int) ([]*domain.Group, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	like := "%" + query + "%"
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, name, description, created_at_ms
		FROM groups
		WHERE workspace_id = ? AND (name LIKE ? OR description LIKE ?)
		ORDER BY created_at_ms DESC LIMIT ?`, workspaceID, like, like, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Group
	for rows.Next() {
		var (
			g       domain.Group
			created int64
		)
		if err := rows.Scan(&g.ID, &g.WorkspaceID, &g.Name, &g.Description, &created); err != nil {
			return nil, err
		}
		g.CreatedAt = fromMS(created)
		out = append(out, &g)
	}
	return out, rows.Err()
}

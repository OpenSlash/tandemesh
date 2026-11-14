package store

import (
	"context"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Store) CreateAuditLog(ctx context.Context, a *domain.AuditLog) error {
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO audit_logs(id, workspace_id, actor, action, resource, detail, created_at_ms)
		VALUES(?,?,?,?,?,?,?)`, a.ID, a.WorkspaceID, a.Actor, a.Action, a.Resource, a.Detail, ms(a.CreatedAt))
	return err
}

func (s *Store) ListAuditLogs(ctx context.Context, workspaceID string, limit int) ([]*domain.AuditLog, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, actor, action, resource, detail, created_at_ms
		FROM audit_logs WHERE workspace_id = ? ORDER BY created_at_ms DESC LIMIT ?`, workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.AuditLog
	for rows.Next() {
		var (
			a       domain.AuditLog
			created int64
		)
		if err := rows.Scan(&a.ID, &a.WorkspaceID, &a.Actor, &a.Action, &a.Resource, &a.Detail, &created); err != nil {
			return nil, err
		}
		a.CreatedAt = fromMS(created)
		out = append(out, &a)
	}
	return out, rows.Err()
}

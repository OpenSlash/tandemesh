package store

import (
	"context"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func (s *Store) CreateNotification(ctx context.Context, n *domain.Notification) error {
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO notifications(id, workspace_id, type, title, body, link, read_at_ms, created_at_ms)
		VALUES(?,?,?,?,?,?,?,?)`,
		n.ID, n.WorkspaceID, n.Type, n.Title, n.Body, n.Link, 0, ms(n.CreatedAt))
	return err
}

func (s *Store) ListNotifications(ctx context.Context, workspaceID string, unreadOnly bool, limit int) ([]*domain.Notification, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := `SELECT id, workspace_id, type, title, body, link, read_at_ms, created_at_ms FROM notifications WHERE workspace_id = ?`
	if unreadOnly {
		query += ` AND read_at_ms = 0`
	}
	query += ` ORDER BY created_at_ms DESC LIMIT ?`
	rows, err := s.db.QueryContext(ctx, query, workspaceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Notification
	for rows.Next() {
		var (
			n       domain.Notification
			readAt  int64
			created int64
		)
		if err := rows.Scan(&n.ID, &n.WorkspaceID, &n.Type, &n.Title, &n.Body, &n.Link, &readAt, &created); err != nil {
			return nil, err
		}
		n.CreatedAt = fromMS(created)
		if readAt > 0 {
			at := fromMS(readAt)
			n.ReadAt = &at
		}
		out = append(out, &n)
	}
	return out, rows.Err()
}

func (s *Store) MarkNotificationRead(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE notifications SET read_at_ms = ? WHERE id = ? AND read_at_ms = 0`, ms(time.Now()), id)
	return err
}

func (s *Store) MarkAllNotificationsRead(ctx context.Context, workspaceID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE notifications SET read_at_ms = ? WHERE workspace_id = ? AND read_at_ms = 0`, ms(time.Now()), workspaceID)
	return err
}

package store

import (
	"context"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// ListRetryDeliveries returns failed deliveries whose backoff has elapsed.
func (s *Store) ListRetryDeliveries(ctx context.Context, maxAttempts, limit int) ([]*domain.WebhookDelivery, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if maxAttempts <= 0 {
		maxAttempts = 5
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, provider, automation_id, delivery, kind, status, reason, run_id, body, attempts, created_at_ms
		FROM webhook_deliveries WHERE status = 'error' AND attempts < ? AND retry_after_ms <= ? ORDER BY created_at_ms ASC LIMIT ?`,
		maxAttempts, ms(time.Now()), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.WebhookDelivery
	for rows.Next() {
		var (
			d       domain.WebhookDelivery
			created int64
		)
		if err := rows.Scan(&d.ID, &d.Provider, &d.AutomationID, &d.Delivery, &d.Kind, &d.Status, &d.Reason, &d.RunID, &d.Body, &d.Attempts, &created); err != nil {
			return nil, err
		}
		d.CreatedAt = fromMS(created)
		out = append(out, &d)
	}
	return out, rows.Err()
}

// MarkWebhookDeliveryStatus updates only the status column.
func (s *Store) MarkWebhookDeliveryStatus(ctx context.Context, id, status string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE webhook_deliveries SET status = ? WHERE id = ?`, status, id)
	return err
}

// BumpDeliveryRetry records a failed attempt and schedules exponential backoff.
func (s *Store) BumpDeliveryRetry(ctx context.Context, id string) error {
	attempts := 0
	_ = s.db.QueryRowContext(ctx, `SELECT attempts FROM webhook_deliveries WHERE id = ?`, id).Scan(&attempts)
	next := attempts + 1
	delay := time.Duration(1<<uint(next)) * time.Minute
	if delay > time.Hour {
		delay = time.Hour
	}
	_, err := s.db.ExecContext(ctx, `UPDATE webhook_deliveries SET attempts = ?, retry_after_ms = ? WHERE id = ?`,
		next, ms(time.Now().Add(delay)), id)
	return err
}

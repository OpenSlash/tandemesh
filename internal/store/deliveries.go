package store

import (
	"context"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// CreateWebhookDelivery records an inbound webhook before processing.
func (s *Store) CreateWebhookDelivery(ctx context.Context, d *domain.WebhookDelivery) error {
	if d.CreatedAt.IsZero() {
		d.CreatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO webhook_deliveries
		(id, provider, automation_id, delivery, kind, status, reason, run_id, body, created_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		d.ID, d.Provider, d.AutomationID, d.Delivery, d.Kind, d.Status, d.Reason, d.RunID, d.Body, ms(d.CreatedAt))
	return err
}

// GetWebhookDelivery returns one recorded delivery.
func (s *Store) GetWebhookDelivery(ctx context.Context, id string) (*domain.WebhookDelivery, error) {
	var (
		d       domain.WebhookDelivery
		created int64
	)
	err := s.db.QueryRowContext(ctx, `SELECT id, provider, automation_id, delivery, kind, status, reason, run_id, body, created_at_ms FROM webhook_deliveries WHERE id = ?`, id).
		Scan(&d.ID, &d.Provider, &d.AutomationID, &d.Delivery, &d.Kind, &d.Status, &d.Reason, &d.RunID, &d.Body, &created)
	if err != nil {
		return nil, err
	}
	d.CreatedAt = fromMS(created)
	return &d, nil
}

// UpdateWebhookDelivery records the processing outcome.
func (s *Store) UpdateWebhookDelivery(ctx context.Context, id, status, reason, runID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE webhook_deliveries SET status = ?, reason = ?, run_id = ? WHERE id = ?`,
		status, reason, runID, id)
	return err
}

// ListWebhookDeliveries returns recent deliveries for a binding.
func (s *Store) ListWebhookDeliveries(ctx context.Context, automationID string, limit int) ([]*domain.WebhookDelivery, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	q := `SELECT id, provider, automation_id, delivery, kind, status, reason, run_id, body, created_at_ms FROM webhook_deliveries`
	args := []any{}
	if automationID != "" {
		q += ` WHERE automation_id = ?`
		args = append(args, automationID)
	}
	q += ` ORDER BY created_at_ms DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, q, args...)
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
		if err := rows.Scan(&d.ID, &d.Provider, &d.AutomationID, &d.Delivery, &d.Kind, &d.Status, &d.Reason, &d.RunID, &d.Body, &created); err != nil {
			return nil, err
		}
		d.CreatedAt = fromMS(created)
		out = append(out, &d)
	}
	return out, rows.Err()
}

// LatestWebhookDelivery returns the newest delivery for a provider and optional automation.
func (s *Store) LatestWebhookDelivery(ctx context.Context, provider, automationID string) (*domain.WebhookDelivery, error) {
	q := `SELECT id, provider, automation_id, delivery, kind, status, reason, run_id, body, created_at_ms
		FROM webhook_deliveries WHERE provider = ?`
	args := []any{provider}
	if automationID != "" {
		q += ` AND (automation_id = ? OR automation_id = '')`
		args = append(args, automationID)
	}
	q += ` ORDER BY created_at_ms DESC LIMIT 1`
	var (
		d       domain.WebhookDelivery
		created int64
	)
	err := s.db.QueryRowContext(ctx, q, args...).
		Scan(&d.ID, &d.Provider, &d.AutomationID, &d.Delivery, &d.Kind, &d.Status, &d.Reason, &d.RunID, &d.Body, &created)
	if err != nil {
		return nil, err
	}
	d.CreatedAt = fromMS(created)
	return &d, nil
}

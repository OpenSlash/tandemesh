package store

import (
	"context"
	"database/sql"
	"time"
)

// GetHistoryCache returns cached history JSON and its scan time.
func (s *Store) GetHistoryCache(ctx context.Context, machineID string) (string, time.Time, error) {
	var (
		raw     string
		scanned int64
	)
	err := s.db.QueryRowContext(ctx, `SELECT items_json, scanned_at_ms FROM history_cache WHERE machine_id = ?`, machineID).
		Scan(&raw, &scanned)
	if err == sql.ErrNoRows {
		return "", time.Time{}, nil
	}
	if err != nil {
		return "", time.Time{}, err
	}
	return raw, fromMS(scanned), nil
}

// SetHistoryCache stores a machine's scanned history.
func (s *Store) SetHistoryCache(ctx context.Context, machineID, itemsJSON string, scannedAt time.Time) error {
	if itemsJSON == "" {
		itemsJSON = "[]"
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO history_cache (machine_id, items_json, scanned_at_ms)
		VALUES (?,?,?)
		ON CONFLICT(machine_id) DO UPDATE SET items_json = excluded.items_json, scanned_at_ms = excluded.scanned_at_ms`,
		machineID, itemsJSON, ms(scannedAt))
	return err
}

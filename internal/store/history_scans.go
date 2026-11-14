package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// CreateHistoryScans queues a scan for each machine.
func (s *Store) CreateHistoryScans(ctx context.Context, machineIDs []string) (int, error) {
	now := time.Now().UTC()
	count := 0
	for _, machineID := range machineIDs {
		if machineID == "" {
			continue
		}
		row := s.db.QueryRowContext(ctx, `SELECT id FROM history_scans WHERE machine_id = ? AND status = 'pending' LIMIT 1`, machineID)
		var existing string
		if err := row.Scan(&existing); err == nil {
			continue
		}
		id := "hsc_" + ulid.Make().String()
		if _, err := s.db.ExecContext(ctx, `INSERT INTO history_scans (id, machine_id, status, error, requested_at_ms, completed_at_ms)
			VALUES (?,?, 'pending', '', ?, NULL)`, id, machineID, ms(now)); err != nil {
			return count, err
		}
		count++
	}
	return count, nil
}

// PendingHistoryScans returns queued scans for a machine.
func (s *Store) PendingHistoryScans(ctx context.Context, machineID string, limit int) ([]*domain.HistoryScan, error) {
	if limit <= 0 || limit > 20 {
		limit = 5
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, machine_id, status, error, requested_at_ms, completed_at_ms
		FROM history_scans WHERE machine_id = ? AND status = 'pending' ORDER BY requested_at_ms LIMIT ?`, machineID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*domain.HistoryScan{}
	for rows.Next() {
		scan, err := scanHistoryScan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, scan)
	}
	return out, rows.Err()
}

// FinishHistoryScan stores a machine's scan result.
func (s *Store) FinishHistoryScan(ctx context.Context, id, machineID, status, itemsJSON, errMsg string) error {
	now := time.Now().UTC()
	if _, err := s.db.ExecContext(ctx, `UPDATE history_scans SET status = ?, error = ?, completed_at_ms = ? WHERE id = ?`,
		status, errMsg, ms(now), id); err != nil {
		return err
	}
	if status != "done" {
		return nil
	}
	if itemsJSON == "" {
		itemsJSON = "[]"
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO machine_history (machine_id, items_json, scanned_at_ms)
		VALUES (?,?,?)
		ON CONFLICT(machine_id) DO UPDATE SET items_json = excluded.items_json, scanned_at_ms = excluded.scanned_at_ms`,
		machineID, itemsJSON, ms(now))
	return err
}

// MachineHistory returns cached scans by machine.
func (s *Store) MachineHistory(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT machine_id, items_json FROM machine_history`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var machineID, items string
		if rows.Scan(&machineID, &items) == nil {
			out[machineID] = items
		}
	}
	return out, rows.Err()
}

func scanHistoryScan(sc interface{ Scan(...any) error }) (*domain.HistoryScan, error) {
	var (
		scan      domain.HistoryScan
		requested int64
		completed sql.NullInt64
	)
	if err := sc.Scan(&scan.ID, &scan.MachineID, &scan.Status, &scan.Error, &requested, &completed); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	scan.RequestedAt = fromMS(requested)
	if completed.Valid {
		ts := fromMS(completed.Int64)
		scan.CompletedAt = &ts
	}
	return &scan, nil
}

package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// GetSyncCursor returns the stored cursor, or a zero cursor when absent.
func (s *Store) GetSyncCursor(ctx context.Context, connectionID string) (*domain.SyncCursor, error) {
	var (
		cursor, synced, full int64
		errText              string
	)
	err := s.db.QueryRowContext(ctx, `SELECT cursor_ms, last_synced_at_ms, last_full_ms, last_error FROM sync_cursors WHERE connection_id = ?`, connectionID).
		Scan(&cursor, &synced, &full, &errText)
	if err == sql.ErrNoRows {
		return &domain.SyncCursor{ConnectionID: connectionID}, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.SyncCursor{
		ConnectionID: connectionID,
		Cursor:       fromMS(cursor),
		LastSyncedAt: fromMS(synced),
		LastFullAt:   fromMS(full),
		LastError:    errText,
	}, nil
}

// SaveSyncCursor upserts the sync progress for one connection.
func (s *Store) SaveSyncCursor(ctx context.Context, c *domain.SyncCursor) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO sync_cursors
		(connection_id, cursor_ms, last_synced_at_ms, last_full_ms, last_error)
		VALUES (?,?,?,?,?)
		ON CONFLICT(connection_id) DO UPDATE SET
			cursor_ms = excluded.cursor_ms,
			last_synced_at_ms = excluded.last_synced_at_ms,
			last_full_ms = excluded.last_full_ms,
			last_error = excluded.last_error`,
		c.ConnectionID, ms(c.Cursor), ms(c.LastSyncedAt), ms(c.LastFullAt), c.LastError)
	return err
}

// ListTaskWorkIDs maps work IDs to task IDs for one provider/instance.
func (s *Store) ListTaskWorkIDs(ctx context.Context, provider, instance string) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, work_id FROM tasks WHERE provider = ? AND instance = ?`, provider, instance)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var id, workID string
		if err := rows.Scan(&id, &workID); err != nil {
			return nil, err
		}
		out[workID] = id
	}
	return out, rows.Err()
}

// TouchSyncCursor records a successful sync timestamp.
func (s *Store) TouchSyncCursor(ctx context.Context, connectionID string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO sync_cursors (connection_id, cursor_ms, last_synced_at_ms, last_full_ms, last_error)
		VALUES (?,0,?,0,'') ON CONFLICT(connection_id) DO UPDATE SET last_synced_at_ms = excluded.last_synced_at_ms, last_error = ''`,
		connectionID, ms(at))
	return err
}

// GetBindingSyncCursor returns the cursor for one binding.
func (s *Store) GetBindingSyncCursor(ctx context.Context, connectionID, bindingID string) (*domain.SyncCursor, error) {
	var cursor, synced, full int64
	var errText string
	err := s.db.QueryRowContext(ctx, `SELECT cursor_ms, last_synced_at_ms, last_full_ms, last_error
		FROM binding_sync_cursors WHERE connection_id = ? AND binding_id = ?`, connectionID, bindingID).
		Scan(&cursor, &synced, &full, &errText)
	if err == sql.ErrNoRows {
		return &domain.SyncCursor{ConnectionID: connectionID, BindingID: bindingID}, nil
	}
	if err != nil {
		return nil, err
	}
	return &domain.SyncCursor{
		ConnectionID: connectionID, BindingID: bindingID,
		Cursor: fromMS(cursor), LastSyncedAt: fromMS(synced),
		LastFullAt: fromMS(full), LastError: errText,
	}, nil
}

// SaveBindingSyncCursor upserts the cursor for one binding.
func (s *Store) SaveBindingSyncCursor(ctx context.Context, c *domain.SyncCursor) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO binding_sync_cursors
		(connection_id, binding_id, cursor_ms, last_synced_at_ms, last_full_ms, last_error)
		VALUES (?,?,?,?,?,?)
		ON CONFLICT(connection_id, binding_id) DO UPDATE SET
			cursor_ms = excluded.cursor_ms, last_synced_at_ms = excluded.last_synced_at_ms,
			last_full_ms = excluded.last_full_ms, last_error = excluded.last_error`,
		c.ConnectionID, c.BindingID, ms(c.Cursor), ms(c.LastSyncedAt), ms(c.LastFullAt), c.LastError)
	return err
}

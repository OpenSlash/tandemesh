package store

import (
	"context"
	"time"
)

// ClaimDebounce reports whether the key fired inside the window and refreshes it.
func (s *Store) ClaimDebounce(ctx context.Context, key string, window time.Duration) (bool, error) {
	now := ms(time.Now())
	cutoff := ms(time.Now().Add(-window))
	res, err := s.db.ExecContext(ctx, `INSERT INTO webhook_debounce(key, seen_at_ms) VALUES(?, ?) ON CONFLICT(key) DO NOTHING`, key, now)
	if err != nil {
		return false, err
	}
	if affected, _ := res.RowsAffected(); affected == 1 {
		return false, nil
	}
	res, err = s.db.ExecContext(ctx, `UPDATE webhook_debounce SET seen_at_ms = ? WHERE key = ? AND seen_at_ms > ?`, now, key, cutoff)
	if err != nil {
		return false, err
	}
	if affected, _ := res.RowsAffected(); affected == 1 {
		return true, nil
	}
	_, err = s.db.ExecContext(ctx, `UPDATE webhook_debounce SET seen_at_ms = ? WHERE key = ?`, now, key)
	return false, err
}

// AcquireSyncLock takes a named lease shared across control-plane instances.
func (s *Store) AcquireSyncLock(ctx context.Context, name, owner string, ttl time.Duration) (bool, error) {
	now := ms(time.Now())
	res, err := s.db.ExecContext(ctx, `INSERT INTO sync_locks(name, owner, expires_at_ms) VALUES(?,?,?) ON CONFLICT(name) DO NOTHING`, name, owner, ms(time.Now().Add(ttl)))
	if err != nil {
		return false, err
	}
	if affected, _ := res.RowsAffected(); affected == 1 {
		return true, nil
	}
	res, err = s.db.ExecContext(ctx, `UPDATE sync_locks SET owner = ?, expires_at_ms = ? WHERE name = ? AND expires_at_ms < ?`, owner, ms(time.Now().Add(ttl)), name, now)
	if err != nil {
		return false, err
	}
	affected, _ := res.RowsAffected()
	return affected == 1, nil
}

// ReleaseSyncLock drops a lock only when the caller still owns it.
func (s *Store) ReleaseSyncLock(ctx context.Context, name, owner string) {
	_, _ = s.db.ExecContext(ctx, `DELETE FROM sync_locks WHERE name = ? AND owner = ?`, name, owner)
}

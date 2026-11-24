package store

import (
	"context"
	"time"
)

// CreateMachineJoinToken stores a single-use machine registration token hash.
func (s *Store) CreateMachineJoinToken(ctx context.Context, id, tokenHash, createdBy string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO machine_join_tokens
		(id, token_hash, expires_at_ms, created_by, created_at_ms) VALUES (?,?,?,?,?)`,
		id, tokenHash, ms(expiresAt), createdBy, ms(time.Now()))
	return err
}

// ConsumeMachineJoinToken marks a valid unused token as used.
func (s *Store) ConsumeMachineJoinToken(ctx context.Context, tokenHash string) (bool, error) {
	now := time.Now()
	res, err := s.db.ExecContext(ctx, `UPDATE machine_join_tokens SET used_at_ms = ?
		WHERE token_hash = ? AND used_at_ms IS NULL AND expires_at_ms > ?`,
		ms(now), tokenHash, ms(now))
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

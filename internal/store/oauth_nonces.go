package store

import (
	"context"
	"time"
)

// CreateOAuthNonce stores a one-time OAuth state nonce.
func (s *Store) CreateOAuthNonce(ctx context.Context, nonce string, expires time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO oauth_nonces(nonce, expires_at_ms, used)
		VALUES (?,?,0)`, nonce, ms(expires))
	return err
}

// CleanupOAuthNonces removes expired nonces.
func (s *Store) CleanupOAuthNonces(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM oauth_nonces WHERE expires_at_ms < ?`, ms(time.Now()))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ConsumeOAuthNonce marks a nonce used, returning false for unknown, expired or reused ones.
func (s *Store) ConsumeOAuthNonce(ctx context.Context, nonce string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `UPDATE oauth_nonces SET used = 1
		WHERE nonce = ? AND used = 0 AND expires_at_ms > ?`, nonce, ms(time.Now()))
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

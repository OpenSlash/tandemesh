package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// CreateSessionShare creates a read-only capability token for a session.
func (s *Store) CreateSessionShare(ctx context.Context, sessionID, createdBy string, ttl time.Duration) (*domain.SessionShare, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	share := &domain.SessionShare{
		Token:     hex.EncodeToString(buf),
		SessionID: sessionID,
		CreatedBy: createdBy,
		CreatedAt: now,
	}
	if ttl > 0 {
		expires := now.Add(ttl)
		share.ExpiresAt = &expires
	}
	var expires any
	if share.ExpiresAt != nil {
		expires = ms(*share.ExpiresAt)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO session_shares (token, session_id, created_by, created_at_ms, expires_at_ms)
		VALUES (?,?,?,?,?)`, share.Token, share.SessionID, share.CreatedBy, ms(share.CreatedAt), expires)
	if err != nil {
		return nil, err
	}
	return share, nil
}

// GetSessionShare returns a live share token, or nil when expired/missing.
func (s *Store) GetSessionShare(ctx context.Context, token string) (*domain.SessionShare, error) {
	var (
		share     domain.SessionShare
		created   int64
		expiresAt sql.NullInt64
	)
	err := s.db.QueryRowContext(ctx, `SELECT token, session_id, created_by, created_at_ms, expires_at_ms
		FROM session_shares WHERE token = ?`, token).
		Scan(&share.Token, &share.SessionID, &share.CreatedBy, &created, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	share.CreatedAt = fromMS(created)
	if expiresAt.Valid {
		expires := fromMS(expiresAt.Int64)
		if time.Now().After(expires) {
			return nil, nil
		}
		share.ExpiresAt = &expires
	}
	return &share, nil
}

// RevokeSessionShare removes a share token.
func (s *Store) RevokeSessionShare(ctx context.Context, sessionID, token string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM session_shares WHERE token = ? AND session_id = ?`, token, sessionID)
	return err
}

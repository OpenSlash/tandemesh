package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// ErrNotFound is returned when a pairing or device does not exist.
var ErrNotFound = errors.New("not found")

// CreateMobilePairing stores a single-use pairing code hash.
func (s *Store) CreateMobilePairing(ctx context.Context, p *domain.MobilePairing) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO mobile_pairings
		(id, code_hash, user_id, workspace_id, expires_at_ms, created_at_ms)
		VALUES (?, ?, ?, ?, ?, ?)`,
		p.ID, p.CodeHash, p.UserID, p.WorkspaceID, p.ExpiresAtMS, p.CreatedAtMS)
	if err != nil {
		return fmt.Errorf("create mobile pairing: %w", err)
	}
	return nil
}

// GetMobilePairing returns a pairing by id.
func (s *Store) GetMobilePairing(ctx context.Context, id string) (*domain.MobilePairing, error) {
	var p domain.MobilePairing
	var used sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT id, code_hash, user_id, workspace_id,
		expires_at_ms, used_at_ms, created_at_ms FROM mobile_pairings WHERE id = ?`, id).
		Scan(&p.ID, &p.CodeHash, &p.UserID, &p.WorkspaceID, &p.ExpiresAtMS, &used, &p.CreatedAtMS)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get mobile pairing: %w", err)
	}
	if used.Valid {
		p.UsedAtMS = used.Int64
	}
	return &p, nil
}

// ConsumeMobilePairing marks a code used and creates the device token.
func (s *Store) ConsumeMobilePairing(ctx context.Context, codeHash, tokenID, tokenHash, deviceName, platform string, nowMS int64) (*domain.MobileToken, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("consume pairing: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var pairingID, userID string
	var expiresAt int64
	var used sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT id, user_id, expires_at_ms, used_at_ms
		FROM mobile_pairings WHERE code_hash = ?`, codeHash).
		Scan(&pairingID, &userID, &expiresAt, &used)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("consume pairing: %w", err)
	}
	if used.Valid || nowMS > expiresAt {
		return nil, ErrNotFound
	}
	result, err := tx.ExecContext(ctx, `UPDATE mobile_pairings SET used_at_ms = ?
		WHERE id = ? AND used_at_ms IS NULL`, nowMS, pairingID)
	if err != nil {
		return nil, fmt.Errorf("consume pairing: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("consume pairing: %w", err)
	}
	if affected == 0 {
		return nil, ErrNotFound
	}
	token := &domain.MobileToken{
		ID:          tokenID,
		UserID:      userID,
		TokenHash:   tokenHash,
		DeviceName:  deviceName,
		Platform:    platform,
		CreatedAtMS: nowMS,
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO mobile_tokens
		(id, user_id, token_hash, device_name, platform, created_at_ms)
		VALUES (?, ?, ?, ?, ?, ?)`,
		token.ID, token.UserID, token.TokenHash, token.DeviceName, token.Platform, token.CreatedAtMS); err != nil {
		return nil, fmt.Errorf("create mobile token: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("consume pairing: %w", err)
	}
	return token, nil
}

// FindMobileToken returns the active device for a token hash and refreshes its
// last-used timestamp. Revoked and unknown tokens return ErrNotFound.
func (s *Store) FindMobileToken(ctx context.Context, tokenHash string, nowMS int64) (*domain.MobileToken, error) {
	var t domain.MobileToken
	var lastUsed, revoked sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT id, user_id, token_hash, device_name, platform,
		created_at_ms, last_used_ms, revoked_at_ms FROM mobile_tokens WHERE token_hash = ?`, tokenHash).
		Scan(&t.ID, &t.UserID, &t.TokenHash, &t.DeviceName, &t.Platform, &t.CreatedAtMS, &lastUsed, &revoked)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find mobile token: %w", err)
	}
	if revoked.Valid {
		return nil, ErrNotFound
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE mobile_tokens SET last_used_ms = ? WHERE id = ?`, nowMS, t.ID); err != nil {
		return nil, fmt.Errorf("touch mobile token: %w", err)
	}
	t.LastUsedMS = nowMS
	return &t, nil
}

// ListMobileTokens lists active and revoked devices for a user.
func (s *Store) ListMobileTokens(ctx context.Context, userID string) ([]*domain.MobileToken, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, user_id, token_hash, device_name, platform,
		created_at_ms, last_used_ms, revoked_at_ms FROM mobile_tokens
		WHERE user_id = ? ORDER BY created_at_ms DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list mobile tokens: %w", err)
	}
	defer rows.Close()
	var out []*domain.MobileToken
	for rows.Next() {
		var t domain.MobileToken
		var lastUsed, revoked sql.NullInt64
		if err := rows.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.DeviceName, &t.Platform,
			&t.CreatedAtMS, &lastUsed, &revoked); err != nil {
			return nil, fmt.Errorf("list mobile tokens: %w", err)
		}
		if lastUsed.Valid {
			t.LastUsedMS = lastUsed.Int64
		}
		if revoked.Valid {
			t.RevokedAtMS = revoked.Int64
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

// RevokeMobileToken revokes a device scoped to its owner.
func (s *Store) RevokeMobileToken(ctx context.Context, id, userID string, nowMS int64) error {
	result, err := s.db.ExecContext(ctx, `UPDATE mobile_tokens SET revoked_at_ms = ?
		WHERE id = ? AND user_id = ? AND revoked_at_ms IS NULL`, nowMS, id, userID)
	if err != nil {
		return fmt.Errorf("revoke mobile token: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke mobile token: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteExpiredMobilePairings removes expired pairing codes.
func (s *Store) DeleteExpiredMobilePairings(ctx context.Context, beforeMS int64) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM mobile_pairings WHERE expires_at_ms < ?`, beforeMS); err != nil {
		return fmt.Errorf("delete expired mobile pairings: %w", err)
	}
	return nil
}

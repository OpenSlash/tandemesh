package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// UpsertMemberIdentity stores one member's provider identity.
func (s *Store) UpsertMemberIdentity(ctx context.Context, identity *domain.MemberIdentity) error {
	if identity.WorkspaceID == "" {
		identity.WorkspaceID = "ws_default"
	}
	now := time.Now().UTC()
	if identity.CreatedAt.IsZero() {
		identity.CreatedAt = now
	}
	identity.UpdatedAt = now
	_, err := s.db.ExecContext(ctx, `INSERT INTO member_identities
		(id, workspace_id, member_id, provider, external_id, login, email, credential_subject, created_at_ms, updated_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(workspace_id, provider, external_id) DO UPDATE SET
			member_id = excluded.member_id, login = excluded.login, email = excluded.email,
			credential_subject = excluded.credential_subject, updated_at_ms = excluded.updated_at_ms`,
		identity.ID, identity.WorkspaceID, identity.MemberID, identity.Provider, identity.ExternalID,
		identity.Login, identity.Email, identity.CredentialSubject, ms(identity.CreatedAt), ms(identity.UpdatedAt))
	return err
}

// ListMemberIdentities returns member identities, optionally filtered by member.
func (s *Store) ListMemberIdentities(ctx context.Context, workspaceID, memberID string) ([]*domain.MemberIdentity, error) {
	q := identityMemberColumns + ` FROM member_identities WHERE workspace_id = ?`
	args := []any{workspaceID}
	if memberID != "" {
		q += ` AND member_id = ?`
		args = append(args, memberID)
	}
	q += ` ORDER BY created_at_ms ASC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.MemberIdentity
	for rows.Next() {
		item, err := scanMemberIdentity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// GetMemberIdentity returns one member identity or nil.
func (s *Store) GetMemberIdentity(ctx context.Context, id string) (*domain.MemberIdentity, error) {
	row := s.db.QueryRowContext(ctx, identityMemberColumns+` FROM member_identities WHERE id = ?`, id)
	item, err := scanMemberIdentity(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return item, err
}

// DeleteMemberIdentity removes one member identity.
func (s *Store) DeleteMemberIdentity(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM member_identities WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

const identityMemberColumns = `SELECT id, workspace_id, member_id, provider, external_id,
	login, email, credential_subject, created_at_ms, updated_at_ms`

func scanMemberIdentity(sc interface{ Scan(...any) error }) (*domain.MemberIdentity, error) {
	var (
		item    domain.MemberIdentity
		created int64
		updated int64
	)
	if err := sc.Scan(&item.ID, &item.WorkspaceID, &item.MemberID, &item.Provider, &item.ExternalID,
		&item.Login, &item.Email, &item.CredentialSubject, &created, &updated); err != nil {
		return nil, err
	}
	item.CreatedAt = fromMS(created)
	item.UpdatedAt = fromMS(updated)
	return &item, nil
}

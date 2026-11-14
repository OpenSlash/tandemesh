package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/oklog/ulid/v2"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// UpsertExternalIdentity maps one provider user to an Orca member.
func (s *Store) UpsertExternalIdentity(ctx context.Context, identity *domain.ExternalIdentity) error {
	if identity.WorkspaceID == "" {
		identity.WorkspaceID = "ws_default"
	}
	now := time.Now().UTC()
	if identity.CreatedAt.IsZero() {
		identity.CreatedAt = now
	}
	identity.UpdatedAt = now
	_, err := s.db.ExecContext(ctx, `INSERT INTO external_identities
		(id, workspace_id, provider, external_id, login, email, member_id, created_at_ms, updated_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?)
		ON CONFLICT(workspace_id, provider, external_id) DO UPDATE SET
			login = excluded.login, email = excluded.email, member_id = excluded.member_id,
			updated_at_ms = excluded.updated_at_ms`,
		identity.ID, identity.WorkspaceID, identity.Provider, identity.ExternalID,
		identity.Login, identity.Email, identity.MemberID, ms(identity.CreatedAt), ms(identity.UpdatedAt))
	return err
}

// ListExternalIdentities returns provider identities in a workspace.
func (s *Store) ListExternalIdentities(ctx context.Context, workspaceID, provider string) ([]*domain.ExternalIdentity, error) {
	q := identityColumns + ` FROM external_identities WHERE workspace_id = ?`
	args := []any{workspaceID}
	if provider != "" {
		q += ` AND provider = ?`
		args = append(args, provider)
	}
	q += ` ORDER BY created_at_ms ASC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.ExternalIdentity
	for rows.Next() {
		item, err := scanIdentity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// GetExternalIdentity returns one mapping or nil.
func (s *Store) GetExternalIdentity(ctx context.Context, id string) (*domain.ExternalIdentity, error) {
	row := s.db.QueryRowContext(ctx, identityColumns+` FROM external_identities WHERE id = ?`, id)
	item, err := scanIdentity(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return item, err
}

// ResolveExternalIdentity maps provider/external id to a workspace member.
func (s *Store) ResolveExternalIdentity(ctx context.Context, workspaceID, provider, externalID string) (*domain.ExternalIdentity, error) {
	if externalID == "" {
		return nil, nil
	}
	row := s.db.QueryRowContext(ctx, identityColumns+`
		FROM external_identities WHERE workspace_id = ? AND provider = ?
		AND (external_id = ? OR login = ?) LIMIT 1`, workspaceID, provider, externalID, externalID)
	item, err := scanIdentity(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return item, err
}

// DeleteExternalIdentity removes one mapping.
func (s *Store) DeleteExternalIdentity(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM external_identities WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

const identityColumns = `SELECT id, workspace_id, provider, external_id, login, email, member_id, created_at_ms, updated_at_ms`

func scanIdentity(sc interface{ Scan(...any) error }) (*domain.ExternalIdentity, error) {
	var (
		item    domain.ExternalIdentity
		created int64
		updated int64
	)
	if err := sc.Scan(&item.ID, &item.WorkspaceID, &item.Provider, &item.ExternalID,
		&item.Login, &item.Email, &item.MemberID, &created, &updated); err != nil {
		return nil, err
	}
	item.CreatedAt = fromMS(created)
	item.UpdatedAt = fromMS(updated)
	return &item, nil
}

// EnsureExternalIdentity creates an external identity for an unknown actor.
func (s *Store) EnsureExternalIdentity(ctx context.Context, workspaceID, provider, externalID, login, email string) (*domain.ExternalIdentity, error) {
	if externalID == "" {
		externalID = login
	}
	if externalID == "" {
		return nil, nil
	}
	if existing, err := s.ResolveExternalIdentity(ctx, workspaceID, provider, externalID); err == nil && existing != nil {
		return existing, nil
	}
	identity := &domain.ExternalIdentity{
		ID: "ext_" + ulid.Make().String(), WorkspaceID: workspaceID, Provider: provider,
		ExternalID: externalID, Login: login, Email: email,
	}
	if identity.Login == "" {
		identity.Login = externalID
	}
	if err := s.UpsertExternalIdentity(ctx, identity); err != nil {
		return nil, err
	}
	return identity, nil
}

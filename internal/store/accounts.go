package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// UpsertConnectionAccount stores one identity for a connection.
func (s *Store) UpsertConnectionAccount(ctx context.Context, account *domain.ConnectionAccount) error {
	if account.WorkspaceID == "" {
		account.WorkspaceID = "ws_default"
	}
	if account.Kind == "" {
		account.Kind = "user"
	}
	if account.CredentialSubject == "" {
		account.CredentialSubject = account.ID
	}
	now := time.Now().UTC()
	if account.CreatedAt.IsZero() {
		account.CreatedAt = now
	}
	account.UpdatedAt = now
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if account.IsDefault {
		if _, err := tx.ExecContext(ctx, `UPDATE connection_accounts SET is_default = 0 WHERE connection_id = ?`, account.ConnectionID); err != nil {
			return err
		}
	}
	enabled := 0
	if account.IsDefault {
		enabled = 1
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO connection_accounts
		(id, workspace_id, connection_id, provider, kind, member_id, login, email, external_id, credential_subject, is_default, created_at_ms, updated_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			kind = excluded.kind, member_id = excluded.member_id, login = excluded.login, email = excluded.email,
			external_id = excluded.external_id, credential_subject = excluded.credential_subject,
			is_default = excluded.is_default, updated_at_ms = excluded.updated_at_ms`,
		account.ID, account.WorkspaceID, account.ConnectionID, account.Provider, account.Kind, account.MemberID,
		account.Login, account.Email, account.ExternalID, account.CredentialSubject, enabled,
		ms(account.CreatedAt), ms(account.UpdatedAt)); err != nil {
		return err
	}
	return tx.Commit()
}

// ListConnectionAccounts returns accounts for one connection.
func (s *Store) ListConnectionAccounts(ctx context.Context, connectionID string) ([]*domain.ConnectionAccount, error) {
	rows, err := s.db.QueryContext(ctx, accountColumns+` FROM connection_accounts WHERE connection_id = ? ORDER BY is_default DESC, created_at_ms ASC`, connectionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.ConnectionAccount
	for rows.Next() {
		item, err := scanAccount(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// GetConnectionAccount returns one account or nil.
func (s *Store) GetConnectionAccount(ctx context.Context, id string) (*domain.ConnectionAccount, error) {
	row := s.db.QueryRowContext(ctx, accountColumns+` FROM connection_accounts WHERE id = ?`, id)
	item, err := scanAccount(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return item, err
}

// SetDefaultConnectionAccount marks one account as default.
func (s *Store) SetDefaultConnectionAccount(ctx context.Context, id string) (*domain.ConnectionAccount, error) {
	account, err := s.GetConnectionAccount(ctx, id)
	if err != nil || account == nil {
		return nil, err
	}
	account.IsDefault = true
	if err := s.UpsertConnectionAccount(ctx, account); err != nil {
		return nil, err
	}
	return account, nil
}

// DeleteConnectionAccount removes one account.
func (s *Store) DeleteConnectionAccount(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM connection_accounts WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

const accountColumns = `SELECT id, workspace_id, connection_id, provider, kind, member_id, login, email,
	external_id, credential_subject, is_default, created_at_ms, updated_at_ms`

func scanAccount(sc interface{ Scan(...any) error }) (*domain.ConnectionAccount, error) {
	var (
		item      domain.ConnectionAccount
		isDefault int
		created   int64
		updated   int64
	)
	if err := sc.Scan(&item.ID, &item.WorkspaceID, &item.ConnectionID, &item.Provider, &item.Kind, &item.MemberID,
		&item.Login, &item.Email, &item.ExternalID, &item.CredentialSubject, &isDefault,
		&created, &updated); err != nil {
		return nil, err
	}
	item.IsDefault = isDefault == 1
	item.CreatedAt = fromMS(created)
	item.UpdatedAt = fromMS(updated)
	return &item, nil
}

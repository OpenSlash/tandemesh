package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// UpsertBinding stores one resource binding for a connection.
func (s *Store) UpsertBinding(ctx context.Context, b *domain.ResourceBinding) error {
	if b.WorkspaceID == "" {
		b.WorkspaceID = "ws_default"
	}
	if b.Direction == "" {
		b.Direction = "read_write"
	}
	if b.Kind == "" {
		b.Kind = "resource"
	}
	now := time.Now().UTC()
	if b.CreatedAt.IsZero() {
		b.CreatedAt = now
	}
	b.UpdatedAt = now
	settings, _ := json.Marshal(b.Settings)
	if b.Settings == nil {
		settings = []byte("{}")
	}
	enabled := 0
	if b.Enabled {
		enabled = 1
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO resource_bindings
		(id, workspace_id, connection_id, provider, external_id, name, kind, direction, account_id, settings_json, enabled, created_at_ms, updated_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(connection_id, external_id) DO UPDATE SET
			workspace_id = excluded.workspace_id, provider = excluded.provider,
			name = excluded.name, kind = excluded.kind, direction = excluded.direction,
			account_id = excluded.account_id,
			settings_json = excluded.settings_json, enabled = excluded.enabled,
			updated_at_ms = excluded.updated_at_ms`,
		b.ID, b.WorkspaceID, b.ConnectionID, b.Provider, b.ExternalID, b.Name, b.Kind, b.Direction,
		b.AccountID, string(settings), enabled, ms(b.CreatedAt), ms(b.UpdatedAt))
	return err
}

// ListBindings returns bindings in a workspace, optionally filtered by connection.
func (s *Store) ListBindings(ctx context.Context, workspaceID, connectionID string) ([]*domain.ResourceBinding, error) {
	q := bindingColumns + ` FROM resource_bindings WHERE workspace_id = ?`
	args := []any{workspaceID}
	if connectionID != "" {
		q += ` AND connection_id = ?`
		args = append(args, connectionID)
	}
	q += ` ORDER BY created_at_ms ASC`
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.ResourceBinding
	for rows.Next() {
		b, err := scanBinding(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// GetBinding returns one binding or nil.
func (s *Store) GetBinding(ctx context.Context, id string) (*domain.ResourceBinding, error) {
	row := s.db.QueryRowContext(ctx, bindingColumns+` FROM resource_bindings WHERE id = ?`, id)
	b, err := scanBinding(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return b, err
}

// DeleteBinding removes one binding.
func (s *Store) DeleteBinding(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM resource_bindings WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

const bindingColumns = `SELECT id, workspace_id, connection_id, provider, external_id, name, kind, direction, account_id,
	settings_json, enabled, last_synced_at_ms, last_error, created_at_ms, updated_at_ms`

func scanBinding(sc interface{ Scan(...any) error }) (*domain.ResourceBinding, error) {
	var (
		b          domain.ResourceBinding
		settings   string
		enabled    int
		lastSynced sql.NullInt64
		created    int64
		updated    int64
	)
	if err := sc.Scan(&b.ID, &b.WorkspaceID, &b.ConnectionID, &b.Provider, &b.ExternalID, &b.Name,
		&b.Kind, &b.Direction, &b.AccountID, &settings, &enabled, &lastSynced, &b.LastError, &created, &updated); err != nil {
		return nil, err
	}
	b.LastSyncedAt = fromNull(lastSynced)
	b.Settings = map[string]string{}
	_ = json.Unmarshal([]byte(settings), &b.Settings)
	b.Enabled = enabled == 1
	b.CreatedAt = fromMS(created)
	b.UpdatedAt = fromMS(updated)
	return &b, nil
}

// WritebackAllowed reports whether the binding for a work item permits write-back.
// Connections without a matching binding keep the legacy behavior.
func (s *Store) WritebackAllowed(ctx context.Context, provider, connectionID, instance string) bool {
	if instance != "" {
		query := `SELECT direction FROM resource_bindings
			WHERE provider = ? AND external_id = ? AND enabled = 1`
		args := []any{provider, instance}
		if connectionID != "" {
			query += ` AND (connection_id = ? OR connection_id = '')`
			args = append(args, connectionID)
		}
		query += ` LIMIT 1`
		var direction string
		if err := s.db.QueryRowContext(ctx, query, args...).Scan(&direction); err == nil {
			return direction != "read"
		}
	}
	if connectionID != "" {
		var direction string
		if err := s.db.QueryRowContext(ctx, `SELECT sync_direction FROM connections WHERE id = ?`, connectionID).Scan(&direction); err == nil {
			return direction != "read"
		}
	}
	return true
}

// BindingAccountID returns the account configured for a work instance, if any.
func (s *Store) BindingAccountID(ctx context.Context, provider, connectionID, instance string) string {
	if instance == "" {
		return ""
	}
	query := `SELECT account_id FROM resource_bindings WHERE provider = ? AND external_id = ? AND enabled = 1`
	args := []any{provider, instance}
	if connectionID != "" {
		query += ` AND (connection_id = ? OR connection_id = '')`
		args = append(args, connectionID)
	}
	query += ` LIMIT 1`
	var accountID string
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&accountID); err == nil {
		return accountID
	}
	return ""
}

// SetBindingSyncState records the last sync time and error for one binding.
func (s *Store) SetBindingSyncState(ctx context.Context, id string, at time.Time, errText string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE resource_bindings SET last_synced_at_ms = ?, last_error = ?, updated_at_ms = ? WHERE id = ?`,
		ms(at), errText, ms(time.Now()), id)
	return err
}

// FindBinding returns the enabled binding for a provider external resource.
func (s *Store) FindBinding(ctx context.Context, provider, externalID string) (*domain.ResourceBinding, error) {
	row := s.db.QueryRowContext(ctx, bindingColumns+` FROM resource_bindings
		WHERE provider = ? AND external_id = ? AND enabled = 1 LIMIT 1`, provider, externalID)
	binding, err := scanBinding(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return binding, err
}

// ListAllBindings returns every enabled binding across workspaces.
func (s *Store) ListAllBindings(ctx context.Context) ([]*domain.ResourceBinding, error) {
	rows, err := s.db.QueryContext(ctx, bindingColumns+` FROM resource_bindings WHERE enabled = 1 ORDER BY created_at_ms ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.ResourceBinding
	for rows.Next() {
		b, err := scanBinding(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

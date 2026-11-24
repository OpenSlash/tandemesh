package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

// UpsertConnection stores a provider connection (HiFox-style).
func (s *Store) UpsertConnection(ctx context.Context, c *domain.Connection) error {
	if c.WorkspaceID == "" {
		c.WorkspaceID = "ws_default"
	}
	now := time.Now().UTC()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now
	if c.Status == "" {
		c.Status = domain.ConnectionPending
	}
	if c.AuthMethod == "" {
		c.AuthMethod = "access_token"
	}
	if c.SyncDirection == "" {
		c.SyncDirection = "read_write"
	}
	if c.ExecutionMode == "" {
		c.ExecutionMode = "cloud"
	}
	enabled := 0
	if c.Enabled {
		enabled = 1
	}
	var expires, callback any
	if c.TokenExpiresAt != nil {
		expires = ms(*c.TokenExpiresAt)
	}
	if c.LastCallbackAt != nil {
		callback = ms(*c.LastCallbackAt)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO connections
		(id, workspace_id, provider, name, status, auth_method, sync_direction, instance, base_url, account_login,
		 execution_mode, runner_machine_id, token_expires_at_ms, last_callback_at_ms, automation_id, enabled, created_at_ms, updated_at_ms)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			provider = excluded.provider, name = excluded.name, status = excluded.status,
			auth_method = excluded.auth_method, sync_direction = excluded.sync_direction,
			instance = excluded.instance, base_url = excluded.base_url,
			account_login = excluded.account_login, execution_mode = excluded.execution_mode,
			runner_machine_id = excluded.runner_machine_id, token_expires_at_ms = excluded.token_expires_at_ms,
			last_callback_at_ms = excluded.last_callback_at_ms, automation_id = excluded.automation_id,
			enabled = excluded.enabled, updated_at_ms = excluded.updated_at_ms`,
		c.ID, c.WorkspaceID, c.Provider, c.Name, c.Status, c.AuthMethod, c.SyncDirection, c.Instance, c.BaseURL,
		c.AccountLogin, c.ExecutionMode, c.RunnerMachineID, expires, callback, c.AutomationID, enabled, ms(c.CreatedAt), ms(c.UpdatedAt))
	return err
}

// GetConnection returns one connection or nil.
func (s *Store) GetConnection(ctx context.Context, id string) (*domain.Connection, error) {
	row := s.db.QueryRowContext(ctx, connectionColumns+` FROM connections WHERE id = ?`, id)
	c, err := scanConnection(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return c, err
}

// ListConnections returns workspace connections, optionally filtered by provider.
func (s *Store) ListConnections(ctx context.Context, workspaceID, provider string) ([]*domain.Connection, error) {
	q := connectionColumns + ` FROM connections WHERE workspace_id = ?`
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
	var out []*domain.Connection
	for rows.Next() {
		c, err := scanConnection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListAllConnections returns every connection across workspaces.
func (s *Store) ListAllConnections(ctx context.Context) ([]*domain.Connection, error) {
	rows, err := s.db.QueryContext(ctx, connectionColumns+` FROM connections ORDER BY created_at_ms ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domain.Connection
	for rows.Next() {
		c, err := scanConnection(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// DeleteConnection removes a connection row.
func (s *Store) DeleteConnection(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM connections WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// SetConnectionStatus updates the connection lifecycle status.
func (s *Store) SetConnectionStatus(ctx context.Context, id, status string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE connections SET status = ?, updated_at_ms = ? WHERE id = ?`,
		status, ms(time.Now()), id)
	return err
}

const connectionColumns = `SELECT id, workspace_id, provider, name, status, auth_method, sync_direction, instance, base_url,
	account_login, execution_mode, runner_machine_id, token_expires_at_ms, last_callback_at_ms, automation_id, enabled,
	created_at_ms, updated_at_ms`

func scanConnection(sc interface{ Scan(...any) error }) (*domain.Connection, error) {
	var (
		c        domain.Connection
		expires  sql.NullInt64
		callback sql.NullInt64
		enabled  int
		created  int64
		updated  int64
	)
	if err := sc.Scan(&c.ID, &c.WorkspaceID, &c.Provider, &c.Name, &c.Status, &c.AuthMethod, &c.SyncDirection,
		&c.Instance, &c.BaseURL, &c.AccountLogin, &c.ExecutionMode, &c.RunnerMachineID,
		&expires, &callback, &c.AutomationID, &enabled, &created, &updated); err != nil {
		return nil, err
	}
	c.TokenExpiresAt = fromNull(expires)
	c.LastCallbackAt = fromNull(callback)
	c.Enabled = enabled == 1
	c.CreatedAt = fromMS(created)
	c.UpdatedAt = fromMS(updated)
	return &c, nil
}

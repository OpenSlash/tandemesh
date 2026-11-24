package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// OAuthApp is a provider OAuth application configuration. The client secret
// lives in the credential vault, never in this table.
type OAuthApp struct {
	Provider  string    `json:"provider"`
	ClientID  string    `json:"clientId"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (s *Store) GetOAuthApp(ctx context.Context, provider string) (*OAuthApp, error) {
	var app OAuthApp
	var updated int64
	err := s.db.QueryRowContext(ctx, `SELECT provider, client_id, updated_at_ms FROM oauth_apps WHERE provider = ?`, provider).
		Scan(&app.Provider, &app.ClientID, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	app.UpdatedAt = fromMS(updated)
	return &app, nil
}

func (s *Store) PutOAuthApp(ctx context.Context, provider, clientID string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO oauth_apps(provider, client_id, updated_at_ms) VALUES(?, ?, ?)
		ON CONFLICT(provider) DO UPDATE SET client_id = excluded.client_id, updated_at_ms = excluded.updated_at_ms`,
		provider, clientID, ms(time.Now()))
	return err
}

func (s *Store) DeleteOAuthApp(ctx context.Context, provider string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM oauth_apps WHERE provider = ?`, provider)
	return err
}

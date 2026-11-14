package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// RebuildLocalOnly migrates a legacy multi-tenant database to the local-only
// schema. Workspaces are preserved; organizations, organization members and
// workspace members are removed. It is idempotent: a database that already
// uses the local workspace schema is left untouched apart from dropping any
// leftover organization tables.
func (s *Store) RebuildLocalOnly(ctx context.Context) error {
	exists, err := s.tableExists(ctx, "workspaces")
	if err != nil {
		return err
	}
	if exists {
		legacy, err := s.columnExists(ctx, "workspaces", "org_id")
		if err != nil {
			return err
		}
		if legacy {
			if err := s.rebuildWorkspacesLocal(ctx); err != nil {
				return err
			}
		}
	}
	for _, stmt := range []string{
		`DROP TABLE IF EXISTS organization_members`,
		`DROP TABLE IF EXISTS workspace_members`,
		`DROP TABLE IF EXISTS organizations`,
	} {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("local rebuild: %w", err)
		}
	}
	return nil
}

// rebuildWorkspacesLocal recreates the workspaces table without org_id while
// preserving every existing workspace row.
func (s *Store) rebuildWorkspacesLocal(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	steps := []string{
		`DROP TABLE IF EXISTS workspaces_local`,
		`CREATE TABLE workspaces_local (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL DEFAULT '',
			slug TEXT NOT NULL DEFAULT '',
			created_at_ms INTEGER NOT NULL
		)`,
		`INSERT INTO workspaces_local(id, name, slug, created_at_ms)
			SELECT id, name, slug, created_at_ms FROM workspaces`,
		`DROP TABLE workspaces`,
		`ALTER TABLE workspaces_local RENAME TO workspaces`,
		`CREATE INDEX IF NOT EXISTS workspaces_slug_idx ON workspaces(slug)`,
	}
	for _, stmt := range steps {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("local rebuild: %w", err)
		}
	}
	return tx.Commit()
}

func (s *Store) tableExists(ctx context.Context, table string) (bool, error) {
	var probe int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM `+table+` LIMIT 1`).Scan(&probe)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		if isMissingSchemaObject(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Store) columnExists(ctx context.Context, table, column string) (bool, error) {
	var probe any
	err := s.db.QueryRowContext(ctx, `SELECT `+column+` FROM `+table+` LIMIT 1`).Scan(&probe)
	if err == sql.ErrNoRows {
		return true, nil
	}
	if err != nil {
		if isMissingSchemaObject(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func isMissingSchemaObject(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such table") ||
		strings.Contains(msg, "no such column") ||
		strings.Contains(msg, "does not exist")
}

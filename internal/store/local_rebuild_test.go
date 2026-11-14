package store

import (
	"path/filepath"
	"testing"
)

func TestFreshSchemaIsLocalOnly(t *testing.T) {
	st := testStore(t)
	ctx := t.Context()
	var count int
	if err := st.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('organizations','organization_members','workspace_members')`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("org tables=%d err=%v", count, err)
	}
	exists, err := st.columnExists(ctx, "workspaces", "org_id")
	if err != nil || exists {
		t.Fatalf("org_id column present=%v err=%v", exists, err)
	}
	var name string
	if err := st.db.QueryRowContext(ctx, `SELECT name FROM workspaces WHERE id = 'ws_default'`).Scan(&name); err != nil || name != "Default" {
		t.Fatalf("default workspace name=%q err=%v", name, err)
	}
}

func TestRebuildLocalOnly(t *testing.T) {
	st := testStore(t)
	ctx := t.Context()
	// Simulate a database created before the tenancy removal.
	if _, err := st.db.ExecContext(ctx, `ALTER TABLE workspaces ADD COLUMN org_id TEXT NOT NULL DEFAULT ''`); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`CREATE TABLE organizations (id TEXT PRIMARY KEY, name TEXT NOT NULL, slug TEXT NOT NULL UNIQUE, created_at_ms INTEGER NOT NULL)`,
		`CREATE TABLE organization_members (org_id TEXT NOT NULL, user_id TEXT NOT NULL, role TEXT NOT NULL DEFAULT 'member', created_at_ms INTEGER NOT NULL, PRIMARY KEY (org_id, user_id))`,
		`CREATE TABLE workspace_members (workspace_id TEXT NOT NULL, user_id TEXT NOT NULL, role TEXT NOT NULL DEFAULT 'member', created_at_ms INTEGER NOT NULL, PRIMARY KEY (workspace_id, user_id))`,
	} {
		if _, err := st.db.ExecContext(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := st.db.ExecContext(ctx, `INSERT INTO workspaces(id, org_id, name, slug, created_at_ms) VALUES ('ws_keep','org_1','Keep','keep',1)`); err != nil {
		t.Fatal(err)
	}
	if err := st.RebuildLocalOnly(ctx); err != nil {
		t.Fatal(err)
	}
	var name string
	if err := st.db.QueryRowContext(ctx, `SELECT name FROM workspaces WHERE id = 'ws_keep'`).Scan(&name); err != nil || name != "Keep" {
		t.Fatalf("workspace name=%q err=%v", name, err)
	}
	var count int
	if err := st.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('organizations','organization_members','workspace_members')`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("org tables=%d err=%v", count, err)
	}
	exists, err := st.columnExists(ctx, "workspaces", "org_id")
	if err != nil || exists {
		t.Fatalf("org_id column present=%v err=%v", exists, err)
	}
}

func TestMigrateUpgradesLegacyTenantSchema(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "legacy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := t.Context()
	for _, stmt := range []string{
		`CREATE TABLE workspaces (id TEXT PRIMARY KEY, org_id TEXT NOT NULL, name TEXT NOT NULL, slug TEXT NOT NULL, created_at_ms INTEGER NOT NULL, UNIQUE(org_id, slug))`,
		`CREATE TABLE organizations (id TEXT PRIMARY KEY, name TEXT NOT NULL, slug TEXT NOT NULL UNIQUE, created_at_ms INTEGER NOT NULL)`,
		`CREATE TABLE organization_members (org_id TEXT NOT NULL, user_id TEXT NOT NULL, role TEXT NOT NULL DEFAULT 'member', created_at_ms INTEGER NOT NULL, PRIMARY KEY (org_id, user_id))`,
		`CREATE TABLE workspace_members (workspace_id TEXT NOT NULL, user_id TEXT NOT NULL, role TEXT NOT NULL DEFAULT 'member', created_at_ms INTEGER NOT NULL, PRIMARY KEY (workspace_id, user_id))`,
		`INSERT INTO organizations(id, name, slug, created_at_ms) VALUES ('org_1','Acme','acme',1)`,
		`INSERT INTO workspaces(id, org_id, name, slug, created_at_ms) VALUES ('ws_keep','org_1','Keep','keep',1)`,
	} {
		if _, err := st.db.ExecContext(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	var name string
	if err := st.db.QueryRowContext(ctx, `SELECT name FROM workspaces WHERE id = 'ws_keep'`).Scan(&name); err != nil || name != "Keep" {
		t.Fatalf("workspace name=%q err=%v", name, err)
	}
	var def string
	if err := st.db.QueryRowContext(ctx, `SELECT name FROM workspaces WHERE id = 'ws_default'`).Scan(&def); err != nil || def != "Default" {
		t.Fatalf("default workspace name=%q err=%v", def, err)
	}
	var count int
	if err := st.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('organizations','organization_members','workspace_members')`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("org tables=%d err=%v", count, err)
	}
	exists, err := st.columnExists(ctx, "workspaces", "org_id")
	if err != nil || exists {
		t.Fatalf("org_id column present=%v err=%v", exists, err)
	}
}

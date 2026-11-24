package history

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestScanOpenCodeV2(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".local", "share", "opencode")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "opencode.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	statements := []string{
		`CREATE TABLE session_v2 (id TEXT PRIMARY KEY, title TEXT, directory TEXT, time_updated INTEGER)`,
		`CREATE TABLE session_message (id TEXT, session_id TEXT, type TEXT, seq INTEGER, data TEXT)`,
		`INSERT INTO session_v2 VALUES ('ses_a', 'Fix login', 'C:/work/api', 1760000000000)`,
		`INSERT INTO session_message VALUES ('m1', 'ses_a', 'user', 1, '{"text":"fix the login race"}')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := ScanOpenCode(home, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %+v", entries)
	}
	if entries[0].Title != "Fix login" || entries[0].Cwd != "C:/work/api" || entries[0].MessageCount != 1 {
		t.Fatalf("entry = %+v", entries[0])
	}
	if entries[0].Preview != "fix the login race" {
		t.Fatalf("preview = %q", entries[0].Preview)
	}
	if !entries[0].UpdatedAt.Equal(time.UnixMilli(1760000000000)) {
		t.Fatalf("updated = %v", entries[0].UpdatedAt)
	}
}

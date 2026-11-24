package history

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestScanCursorIDE(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".config", "Cursor", "User", "globalStorage")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", filepath.Join(dir, "state.vscdb"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE cursorDiskKV (key TEXT PRIMARY KEY, value BLOB)`); err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`INSERT INTO cursorDiskKV VALUES ('composerData:comp-1', '{"composerId":"comp-1","name":"Fix cache","createdAt":1760000000000,"lastUpdatedAt":1760000002000}')`,
		`INSERT INTO cursorDiskKV VALUES ('bubbleId:comp-1:b1', '{"type":1,"text":"Why is this stale?","workspaceProjectDir":"C:/work/api"}')`,
		`INSERT INTO cursorDiskKV VALUES ('bubbleId:comp-1:b2', '{"type":2,"text":"The cache key was wrong."}')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := ScanCursorIDE(home, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %+v", entries)
	}
	if entries[0].ID != "comp-1" || entries[0].Title != "Fix cache" || entries[0].Cwd != "C:/work/api" || entries[0].MessageCount != 2 {
		t.Fatalf("entry = %+v", entries[0])
	}
}

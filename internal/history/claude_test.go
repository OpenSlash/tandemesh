package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanClaude(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".claude", "projects", "proj")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	lines := `{"type":"user","sessionId":"abc12345","cwd":"C:\\work\\app","gitBranch":"main","slug":"fix-login","timestamp":"2026-01-01T10:00:00Z","message":{"role":"user","content":[{"type":"text","text":"Fix the login race"}]}}
{"type":"assistant","sessionId":"abc12345","timestamp":"2026-01-01T10:01:00Z","message":{"role":"assistant","model":"claude-sonnet-4-5","content":[{"type":"text","text":"I found the race."}]}}
`
	if err := os.WriteFile(filepath.Join(dir, "abc12345.jsonl"), []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}
	items, err := ScanClaude(home, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %+v", items)
	}
	item := items[0]
	if item.ID != "abc12345" || item.Title != "fix-login" || item.Model != "claude-sonnet-4-5" || item.Cwd != `C:\work\app` {
		t.Fatalf("entry = %+v", item)
	}
	if item.Preview != "Fix the login race" {
		t.Fatalf("preview = %q", item.Preview)
	}
}

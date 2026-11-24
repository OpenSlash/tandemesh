package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanCodex(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".codex", "sessions", "2026", "01", "01")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	lines := `{"timestamp":"2026-01-01T10:00:00Z","type":"session_meta","payload":{"id":"codex-123","cwd":"C:\\work\\api","model":"gpt-5-codex","git":{"branch":"feat/login"}}}
{"timestamp":"2026-01-01T10:01:00Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"Fix the login race"}]}}
`
	if err := os.WriteFile(filepath.Join(dir, "rollout-123.jsonl"), []byte(lines), 0o644); err != nil {
		t.Fatal(err)
	}
	items, err := ScanCodex(home, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %+v", items)
	}
	item := items[0]
	if item.ID != "codex-123" || item.Title != "Fix the login race" || item.Model != "gpt-5-codex" || item.Branch != "feat/login" {
		t.Fatalf("entry = %+v", item)
	}
}

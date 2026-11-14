package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanCursorCLI(t *testing.T) {
	home := t.TempDir()
	transcripts := filepath.Join(home, ".cursor", "projects", "work-api", "agent-transcripts", "chat-1")
	if err := os.MkdirAll(transcripts, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"role":"user","message":{"content":[{"type":"text","text":"<timestamp>Sunday, Jul 26, 2026, 1:06 PM (UTC+3)</timestamp>Why is this stale?"}]}}
{"role":"assistant","message":{"content":"The cache key omitted the locale."}}`
	if err := os.WriteFile(filepath.Join(transcripts, "chat-1.jsonl"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	metaDir := filepath.Join(home, ".cursor", "chats", "hash", "chat-1")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(metaDir, "meta.json"), []byte(`{"cwd":"C:/work/api"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := ScanCursor(home, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %+v", entries)
	}
	if entries[0].ID != "chat-1" || entries[0].Title != "Why is this stale?" || entries[0].Cwd != "C:/work/api" || entries[0].MessageCount != 2 {
		t.Fatalf("entry = %+v", entries[0])
	}
}

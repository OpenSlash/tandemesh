package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanGeminiJSON(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, ".gemini", "tmp", "proj", "chats")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `{"sessionId":"gem-1","startTime":"2026-07-17T09:00:00Z","lastUpdated":"2026-07-17T09:05:00Z","messages":[{"type":"user","content":[{"text":"fix the config"}]},{"type":"gemini","content":[{"text":"done"}]}]}`
	if err := os.WriteFile(filepath.Join(dir, "session.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	entries, err := ScanGemini(home, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %+v", entries)
	}
	if entries[0].ID != "gem-1" || entries[0].Title != "fix the config" || entries[0].MessageCount != 2 {
		t.Fatalf("entry = %+v", entries[0])
	}
}

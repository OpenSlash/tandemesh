package agent

import "testing"

func TestDefaultACPBackendsCoverCatalog(t *testing.T) {
	for _, id := range []string{"claude", "codex", "deepseek", "gemini", "cursor", "qwen", "opencode"} {
		spec, ok := DefaultACPBackends[id]
		if !ok || spec.Command == "" {
			t.Fatalf("missing ACP backend %q", id)
		}
	}
	found := false
	for _, runtime := range RuntimeCatalog {
		if runtime.ID == "deepseek" {
			found = true
			if !runtime.ACP {
				t.Fatal("deepseek must be ACP")
			}
		}
	}
	if !found {
		t.Fatal("deepseek missing from runtime catalog")
	}
}

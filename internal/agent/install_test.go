package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNPMSpec(t *testing.T) {
	tests := []struct{ pkg, version, want string }{
		{"@agentclientprotocol/codex-acp@2.12.0", "2.12.0", "@agentclientprotocol/codex-acp@2.12.0"},
		{"@scope/tool", "1.0.0", "@scope/tool@1.0.0"},
		{"cline@3.0.75", "3.0.75", "cline@3.0.75"},
		{"tool", "", "tool"},
	}
	for _, tc := range tests {
		if got := npmSpec(tc.pkg, tc.version); got != tc.want {
			t.Errorf("npmSpec(%q, %q) = %q, want %q", tc.pkg, tc.version, got, tc.want)
		}
	}
}

func TestPipRequirement(t *testing.T) {
	tests := []struct{ pkg, version, want string }{
		{"fast-agent-acp==0.10.66", "0.10.66", "fast-agent-acp==0.10.66"},
		{"minion-code@0.1.44", "0.1.44", "minion-code==0.1.44"},
		{"minion-code", "0.1.44", "minion-code==0.1.44"},
	}
	for _, tc := range tests {
		if got := pipRequirement(tc.pkg, tc.version); got != tc.want {
			t.Errorf("pipRequirement(%q, %q) = %q, want %q", tc.pkg, tc.version, got, tc.want)
		}
	}
}

func TestNPMBin(t *testing.T) {
	tests := []struct {
		name, manifest, want string
	}{
		{"string bin uses unscoped name", `{"name":"@scope/tool","bin":"dist/cli.js"}`, "tool"},
		{"map bin prefers package name", `{"name":"tool","bin":{"helper":"a.js","tool":"b.js"}}`, "tool"},
		{"map bin with one entry", `{"name":"@agentclientprotocol/codex-acp","bin":{"codex-acp":"bin.js"}}`, "codex-acp"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			prefix := t.TempDir()
			var manifest struct{ Name string }
			_ = json.Unmarshal([]byte(tc.manifest), &manifest)
			dir := filepath.Join(prefix, "node_modules", filepath.FromSlash(manifest.Name))
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(tc.manifest), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := npmBin(prefix, manifest.Name)
			if err != nil || got != tc.want {
				t.Fatalf("npmBin = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

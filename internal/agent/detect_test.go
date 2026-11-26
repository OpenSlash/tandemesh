package agent

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func fakeLookPath(installed ...string) func(string) (string, error) {
	set := map[string]bool{}
	for _, name := range installed {
		set[name] = true
	}
	return func(name string) (string, error) {
		if set[name] {
			return "/bin/" + name, nil
		}
		return "", errors.New("not found")
	}
}

func fakeVersion(context.Context, string) string { return "1.0" }

func statusOf(t *testing.T, det Detection, id string) domain.RuntimeStatus {
	t.Helper()
	for _, st := range det.Statuses {
		if st.ID == id {
			return st
		}
	}
	t.Fatalf("no status for %q", id)
	return domain.RuntimeStatus{}
}

func TestDetectRuntimesStatus(t *testing.T) {
	tests := []struct {
		name      string
		installed []string
		id        string
		status    string
		launcher  string
	}{
		{"cli and adapter", []string{"claude", "claude-agent-acp"}, "claude", domain.RuntimeReady, "/bin/claude-agent-acp"},
		{"cli without adapter", []string{"claude"}, "claude", domain.RuntimeAdapterMissing, "claude-agent-acp"},
		{"legacy adapter", []string{"claude-code-acp"}, "claude", domain.RuntimeReady, "/bin/claude-code-acp"},
		{"nothing installed", nil, "deepseek", domain.RuntimeMissing, "dsh"},
		{"cli is the launcher", []string{"dsh"}, "deepseek", domain.RuntimeReady, "/bin/dsh"},
		{"non-acp cli", []string{"goose"}, "goose", domain.RuntimeUnsupported, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			det := DetectRuntimes(context.Background(), DetectOptions{LookPath: fakeLookPath(tc.installed...), Version: fakeVersion})
			st := statusOf(t, det, tc.id)
			if st.Status != tc.status || st.Launcher != tc.launcher {
				t.Fatalf("got status=%q launcher=%q, want %q %q", st.Status, st.Launcher, tc.status, tc.launcher)
			}
			if _, ok := det.Backends[tc.id]; ok != (tc.status == domain.RuntimeReady) {
				t.Fatalf("backend registered = %v for status %q", ok, st.Status)
			}
		})
	}
}

func TestDetectRuntimesRegistryInstall(t *testing.T) {
	dir := t.TempDir()
	command := filepath.Join(dir, "claude-agent-acp")
	if err := os.WriteFile(command, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	det := DetectRuntimes(context.Background(), DetectOptions{
		Installs: []ACPInstall{
			{ID: "claude-acp", Version: "0.9.0", Command: command},
			{ID: "deepagents", Version: "1.2.0", Command: command, Args: []string{"--acp"}},
		},
		LookPath: fakeLookPath(),
		Version:  fakeVersion,
	})

	claude := statusOf(t, det, "claude")
	if claude.Status != domain.RuntimeReady || claude.Source != SourceRegistry || claude.RegistryID != "claude-acp" || claude.Version != "0.9.0" {
		t.Fatalf("aliased registry install: %+v", claude)
	}
	extra := statusOf(t, det, "deepagents")
	if extra.Status != domain.RuntimeReady || det.Backends["deepagents"].Args[0] != "--acp" {
		t.Fatalf("unaliased registry install: %+v %+v", extra, det.Backends["deepagents"])
	}
}

func TestDetectRuntimesFlagOverride(t *testing.T) {
	det := DetectRuntimes(context.Background(), DetectOptions{
		Overrides: map[string]ACPBackendSpec{"codex": {Command: "my-codex-acp"}},
		LookPath:  fakeLookPath("my-codex-acp", "codex-acp"),
		Version:   fakeVersion,
	})
	if st := statusOf(t, det, "codex"); st.Source != SourceFlag || st.Launcher != "/bin/my-codex-acp" {
		t.Fatalf("flag override not preferred: %+v", st)
	}
}

func TestListACPInstalls(t *testing.T) {
	home := t.TempDir()
	write := func(id, version, at string) {
		dir := filepath.Join(home, id, version)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(acpManifest{ID: id, Version: version, Command: filepath.Join(dir, "bin"), InstalledAt: at})
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("cursor", "2.0.0", "2026-02-01T00:00:00Z")
	write("cursor", "1.0.0", "2026-01-01T00:00:00Z")
	if err := os.MkdirAll(filepath.Join(home, "broken", "1"), 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(home, "broken", "1", "manifest.json"), []byte("{"), 0o644)

	installs, err := ListACPInstalls(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(installs) != 2 || installs[1].Version != "2.0.0" {
		t.Fatalf("installs = %+v, want newest last and broken skipped", installs)
	}
}

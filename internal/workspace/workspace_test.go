package workspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func gitBin(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("ORCA_GIT"); p != "" {
		return p
	}
	p, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not available; set ORCA_GIT to run this test")
	}
	return p
}

func initRepo(t *testing.T, git, dir string) {
	t.Helper()
	cmds := [][]string{
		{"init", "-b", "main"},
		{"config", "user.name", "Test"},
		{"config", "user.email", "test@example.com"},
	}
	for _, args := range cmds {
		cmd := exec.Command(git, args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "."}, {"commit", "-m", "init"}} {
		cmd := exec.Command(git, args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
}

func TestPrepareAndCleanup(t *testing.T) {
	git := gitBin(t)
	source := t.TempDir()
	initRepo(t, git, source)

	root := t.TempDir()
	m := New(filepath.Join(root, "data"), git, time.Hour, nil)

	ws, err := m.Prepare(context.Background(), Request{
		RunID: "run_01ABCDEF",
		Repo:  source,
		Base:  "main",
	})
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if !strings.HasPrefix(ws.Branch, "orca/") {
		t.Fatalf("branch = %q", ws.Branch)
	}
	if _, err := os.Stat(filepath.Join(ws.Dir, "README.md")); err != nil {
		t.Fatalf("worktree missing file: %v", err)
	}

	// A second run gets a different worktree on a different branch.
	ws2, err := m.Prepare(context.Background(), Request{
		RunID: "run_02ABCDEF",
		Repo:  source,
		Base:  "main",
	})
	if err != nil {
		t.Fatalf("second prepare: %v", err)
	}
	if ws2.Dir == ws.Dir {
		t.Fatal("runs share a worktree")
	}
	if ws2.Branch == ws.Branch {
		t.Fatal("runs share a branch")
	}

	if err := m.Cleanup(context.Background(), "run_01ABCDEF"); err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if _, err := os.Stat(ws.Dir); !os.IsNotExist(err) {
		t.Fatalf("worktree still exists: %v", err)
	}
}

func TestPrepareRejectsBadRunID(t *testing.T) {
	m := New(t.TempDir(), "git", time.Hour, nil)
	if _, err := m.Prepare(context.Background(), Request{RunID: "../escape", Repo: "x"}); err == nil {
		t.Fatal("invalid run id accepted")
	}
}

func TestPrepareSessionReusesWorktree(t *testing.T) {
	git := gitBin(t)
	source := t.TempDir()
	initRepo(t, git, source)

	m := New(filepath.Join(t.TempDir(), "data"), git, time.Hour, nil)
	first, err := m.PrepareSession(context.Background(), Request{
		RunID: "ses_01ABCDEF", Repo: source, Base: "main",
	})
	if err != nil {
		t.Fatalf("prepare session: %v", err)
	}
	second, err := m.PrepareSession(context.Background(), Request{
		RunID: "ses_01ABCDEF", Repo: source, Base: "main",
	})
	if err != nil {
		t.Fatalf("prepare session again: %v", err)
	}
	if first.Dir != second.Dir || first.Branch != second.Branch {
		t.Fatalf("session worktree changed: %+v %+v", first, second)
	}
	if _, err := os.Stat(filepath.Join(first.Dir, "README.md")); err != nil {
		t.Fatalf("session worktree missing file: %v", err)
	}
}

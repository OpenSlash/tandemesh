package gitops

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}

func TestShipCommitsAndPushes(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	work := filepath.Join(root, "work")
	git(t, root, "init", "--bare", remote)
	if err := os.Mkdir(work, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, work, "init", "-b", "main")
	git(t, work, "config", "user.name", "Test")
	git(t, work, "config", "user.email", "test@example.com")
	git(t, work, "remote", "add", "origin", remote)
	if err := os.WriteFile(filepath.Join(work, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, work, "add", ".")
	git(t, work, "commit", "-m", "init")
	git(t, work, "push", "-u", "origin", "main")
	if err := os.WriteFile(filepath.Join(work, "README.md"), []byte("hello world\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := New()
	status, err := svc.Inspect(context.Background(), work)
	if err != nil || status.Clean {
		t.Fatalf("status = %+v err=%v", status, err)
	}
	shipped, err := svc.Ship(context.Background(), work, "update readme")
	if err != nil {
		t.Fatalf("ship: %v", err)
	}
	if !shipped.Clean {
		t.Fatalf("ship status = %+v", shipped)
	}
}

func TestGitLabRemoteDetection(t *testing.T) {
	cases := map[string]bool{
		"git@gitlab.com:acme/api.git":             true,
		"https://gitlab.example.com/acme/api.git": true,
		"git@github.com:acme/api.git":             false,
		"https://bitbucket.org/acme/api.git":      false,
	}
	for remote, want := range cases {
		if got := isGitLabRemote(remote); got != want {
			t.Fatalf("%s: got %v", remote, got)
		}
	}
}

func TestBitbucketRepo(t *testing.T) {
	cases := []struct {
		remote    string
		workspace string
		repo      string
		ok        bool
	}{
		{"git@bitbucket.org:acme/api.git", "acme", "api", true},
		{"https://bitbucket.org/acme/api.git", "acme", "api", true},
		{"git@github.com:acme/api.git", "", "", false},
	}
	for _, tc := range cases {
		workspace, repo, ok := bitbucketRepo(tc.remote)
		if workspace != tc.workspace || repo != tc.repo || ok != tc.ok {
			t.Fatalf("%s: got %q %q %v", tc.remote, workspace, repo, ok)
		}
	}
}

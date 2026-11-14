// Package workspace prepares an isolated working tree for every run.
//
// Repositories are cloned once into a cache and reused; each run gets its own
// git worktree on a dedicated branch, so parallel runs never share files.
package workspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Request describes the workspace to prepare.
type Request struct {
	RunID  string
	Repo   string // clone URL or local path
	Base   string // branch or ref; defaults to main
	Branch string // branch to create; defaults to orca/<last 8 of run id>
}

// Workspace is a prepared working tree.
type Workspace struct {
	Dir    string
	Branch string
	Repo   string
}

// Manager owns the repository cache and run workspaces.
type Manager struct {
	Root   string // data root, e.g. ~/.orca
	Git    string // git executable
	Retain time.Duration
	Logger *slog.Logger
	mu     sync.Mutex
}

// New creates a manager. Empty git defaults to "git"; empty root defaults to
// ~/.orca; zero retain defaults to 168h (7 days).
func New(root, git string, retain time.Duration, log *slog.Logger) *Manager {
	if root == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			root = filepath.Join(home, ".orca")
		} else {
			root = ".orca"
		}
	}
	if git == "" {
		git = "git"
	}
	if retain <= 0 {
		retain = 7 * 24 * time.Hour
	}
	if log == nil {
		log = slog.Default()
	}
	return &Manager{Root: root, Git: git, Retain: retain, Logger: log}
}

var runIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// Prepare clones or fetches the repository and creates a worktree for the run.
func (m *Manager) Prepare(ctx context.Context, req Request) (*Workspace, error) {
	if !runIDPattern.MatchString(req.RunID) {
		return nil, fmt.Errorf("invalid run id %q", req.RunID)
	}
	if strings.TrimSpace(req.Repo) == "" {
		return nil, fmt.Errorf("repo is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cache := m.repoCachePath(req.Repo)
	if err := os.MkdirAll(filepath.Dir(cache), 0o755); err != nil {
		return nil, err
	}
	if _, err := os.Stat(cache); os.IsNotExist(err) {
		if out, err := m.git(ctx, "", "clone", "--quiet", "--", req.Repo, cache); err != nil {
			return nil, fmt.Errorf("clone %s: %w: %s", req.Repo, err, strings.TrimSpace(out))
		}
	} else if out, err := m.git(ctx, cache, "fetch", "--quiet", "--all", "--prune"); err != nil {
		return nil, fmt.Errorf("fetch %s: %w: %s", req.Repo, err, strings.TrimSpace(out))
	}

	base := strings.TrimSpace(req.Base)
	if base == "" {
		base = "main"
	}
	if _, err := m.git(ctx, cache, "rev-parse", "--verify", "--quiet", base+"^{commit}"); err != nil {
		if out2, err2 := m.git(ctx, cache, "rev-parse", "--verify", "--quiet", "HEAD^{commit}"); err2 != nil {
			return nil, fmt.Errorf("resolve base %q: %w: %s", base, err, strings.TrimSpace(out2))
		}
		base = "HEAD"
	}

	branch := strings.TrimSpace(req.Branch)
	if branch == "" {
		branch = "orca/" + shortID(req.RunID)
	}

	dir := filepath.Join(m.Root, "runs", req.RunID, "workspace")
	if err := m.removeWorktree(ctx, cache, dir); err != nil {
		m.Logger.Warn("clean stale worktree", "run", req.RunID, "err", err)
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return nil, err
	}
	if out, err := m.git(ctx, cache, "worktree", "add", "--force", "-B", branch, dir, base); err != nil {
		return nil, fmt.Errorf("worktree add: %w: %s", err, strings.TrimSpace(out))
	}
	if err := m.prepareExtras(ctx, cache, dir); err != nil {
		return nil, fmt.Errorf("worktree setup: %w", err)
	}
	return &Workspace{Dir: dir, Branch: branch, Repo: cache}, nil
}

// PrepareSession returns the persistent worktree for an agent session.
// The first run creates it; later runs reuse the same directory and branch.
func (m *Manager) PrepareSession(ctx context.Context, req Request) (*Workspace, error) {
	if !runIDPattern.MatchString(req.RunID) {
		return nil, fmt.Errorf("invalid session id %q", req.RunID)
	}
	if strings.TrimSpace(req.Repo) == "" {
		return nil, fmt.Errorf("repo is required")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cache := m.repoCachePath(req.Repo)
	if err := os.MkdirAll(filepath.Dir(cache), 0o755); err != nil {
		return nil, err
	}
	if _, err := os.Stat(cache); os.IsNotExist(err) {
		if out, err := m.git(ctx, "", "clone", "--quiet", "--", req.Repo, cache); err != nil {
			return nil, fmt.Errorf("clone %s: %w: %s", req.Repo, err, strings.TrimSpace(out))
		}
	} else if out, err := m.git(ctx, cache, "fetch", "--quiet", "--all", "--prune"); err != nil {
		return nil, fmt.Errorf("fetch %s: %w: %s", req.Repo, err, strings.TrimSpace(out))
	}

	base := strings.TrimSpace(req.Base)
	if base == "" {
		base = "main"
	}
	if _, err := m.git(ctx, cache, "rev-parse", "--verify", "--quiet", base+"^{commit}"); err != nil {
		if out2, err2 := m.git(ctx, cache, "rev-parse", "--verify", "--quiet", "HEAD^{commit}"); err2 != nil {
			return nil, fmt.Errorf("resolve base %q: %w: %s", base, err, strings.TrimSpace(out2))
		}
		base = "HEAD"
	}

	branch := strings.TrimSpace(req.Branch)
	if branch == "" {
		branch = "orca/session-" + shortID(req.RunID)
	}

	dir := filepath.Join(m.Root, "sessions", req.RunID, "workspace")
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return &Workspace{Dir: dir, Branch: branch, Repo: cache}, nil
	}
	if err := m.removeWorktree(ctx, cache, dir); err != nil {
		m.Logger.Warn("clean stale session worktree", "session", req.RunID, "err", err)
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o755); err != nil {
		return nil, err
	}
	if out, err := m.git(ctx, cache, "worktree", "add", "--force", "-B", branch, dir, base); err != nil {
		return nil, fmt.Errorf("worktree add: %w: %s", err, strings.TrimSpace(out))
	}
	if err := m.prepareExtras(ctx, cache, dir); err != nil {
		return nil, fmt.Errorf("worktree setup: %w", err)
	}
	return &Workspace{Dir: dir, Branch: branch, Repo: cache}, nil
}

// Cleanup removes a run's worktree. Missing workspaces are not an error.
func (m *Manager) Cleanup(ctx context.Context, runID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	dir := filepath.Join(m.Root, "runs", runID, "workspace")
	// Try to unregister from any cache that knows this worktree.
	for _, cache := range m.repoCaches() {
		if err := m.removeWorktree(ctx, cache, dir); err == nil {
			return nil
		}
	}
	return os.RemoveAll(dir)
}

// GC removes run directories older than the retention window.
func (m *Manager) GC(ctx context.Context, now time.Time) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	runsDir := filepath.Join(m.Root, "runs")
	entries, err := os.ReadDir(runsDir)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if now.Sub(info.ModTime()) < m.Retain {
			continue
		}
		dir := filepath.Join(runsDir, e.Name())
		for _, cache := range m.repoCaches() {
			_ = m.removeWorktree(ctx, cache, filepath.Join(dir, "workspace"))
		}
		if err := os.RemoveAll(dir); err == nil {
			removed++
		}
	}
	for _, cache := range m.repoCaches() {
		_, _ = m.git(ctx, cache, "worktree", "prune")
	}
	return removed, nil
}

func (m *Manager) repoCachePath(repo string) string {
	sum := sha256.Sum256([]byte(repo))
	return filepath.Join(m.Root, "repos", hex.EncodeToString(sum[:])[:16])
}

func (m *Manager) repoCaches() []string {
	dir := filepath.Join(m.Root, "repos")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out
}

func (m *Manager) removeWorktree(ctx context.Context, cache, dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_, _ = m.git(ctx, cache, "worktree", "prune")
		return nil
	}
	if out, err := m.git(ctx, cache, "worktree", "remove", "--force", dir); err != nil {
		_, _ = m.git(ctx, cache, "worktree", "prune")
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("remove %s: %w (git: %s)", dir, err, strings.TrimSpace(out))
		}
	}
	return nil
}

func (m *Manager) git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, m.Git, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func shortID(id string) string {
	if len(id) <= 8 {
		return id
	}
	return id[len(id)-8:]
}

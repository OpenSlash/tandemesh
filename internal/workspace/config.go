package workspace

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// prepareExtras applies repo-local worktree conventions after git worktree add.
//
// Supported files:
//   - orca.yaml / .orca.yaml: scripts.setup block or inline command
//   - .worktreeinclude: newline-separated paths or globs copied from the cache
func (m *Manager) prepareExtras(ctx context.Context, cache, dir string) error {
	if err := m.copyIncludes(cache, dir); err != nil {
		return err
	}
	script := readSetupScript(dir)
	if strings.TrimSpace(script) == "" {
		return nil
	}
	m.Logger.Info("run worktree setup", "dir", dir, "script", firstLine(script))
	runCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(runCtx, "cmd", "/C", script)
	} else {
		cmd = exec.CommandContext(runCtx, "sh", "-c", script)
	}
	cmd.Dir = dir
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("setup script timed out")
		}
		return fmt.Errorf("setup script failed: %w", err)
	}
	return nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return strings.TrimSpace(s)
}

// readSetupScript is a deliberately small YAML reader. It understands the two
// common shapes and avoids adding a YAML dependency to the agent daemon:
//
//	scripts:
//	  setup: npm install
//
//	scripts:
//	  setup: |
//	    npm install
//	    npm test
func readSetupScript(dir string) string {
	for _, name := range []string{"orca.yaml", ".orca.yaml"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		if script := parseSetup(string(data)); script != "" {
			return script
		}
	}
	return ""
}

func parseSetup(src string) string {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "setup:") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(trimmed, "setup:"))
		if value != "" && value != "|" && value != ">" && value != "|-" && value != ">-" {
			return value
		}
		baseIndent := leadingSpaces(line)
		var block []string
		for _, next := range lines[i+1:] {
			if strings.TrimSpace(next) == "" {
				block = append(block, "")
				continue
			}
			if leadingSpaces(next) <= baseIndent {
				break
			}
			block = append(block, next)
		}
		return dedent(block)
	}
	return ""
}

func leadingSpaces(s string) int {
	return len(s) - len(strings.TrimLeft(s, " \t"))
}

func dedent(lines []string) string {
	min := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		n := leadingSpaces(line)
		if min < 0 || n < min {
			min = n
		}
	}
	if min < 0 {
		return ""
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			out = append(out, "")
			continue
		}
		out = append(out, line[min:])
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func (m *Manager) copyIncludes(cache, dir string) error {
	f, err := os.Open(filepath.Join(dir, ".worktreeinclude"))
	if err != nil {
		return nil
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		pattern := strings.TrimSpace(scanner.Text())
		if pattern == "" || strings.HasPrefix(pattern, "#") {
			continue
		}
		matches, err := filepath.Glob(filepath.Join(cache, pattern))
		if err != nil {
			return fmt.Errorf("worktree include %q: %w", pattern, err)
		}
		for _, match := range matches {
			rel, err := filepath.Rel(cache, match)
			if err != nil || strings.HasPrefix(rel, "..") {
				continue
			}
			if err := copyPath(match, filepath.Join(dir, rel)); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, _ := filepath.Rel(src, path)
			target := filepath.Join(dst, rel)
			if d.IsDir() {
				return os.MkdirAll(target, 0o755)
			}
			return copyFile(path, target)
		})
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

package main

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// captureDiff returns a unified diff for the run worktree relative to its base.
func captureDiff(ctx context.Context, dir, base string) (string, bool) {
	if strings.TrimSpace(dir) == "" {
		return "", false
	}
	runCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(runCtx, "git", "-C", dir, "rev-parse", "--is-inside-work-tree").Output(); err != nil || strings.TrimSpace(string(out)) != "true" {
		return "", false
	}
	base = strings.TrimSpace(base)
	if base == "" {
		base = "main"
	}
	baseRef := base
	if out, err := exec.CommandContext(runCtx, "git", "-C", dir, "merge-base", "HEAD", base).Output(); err == nil {
		if mb := strings.TrimSpace(string(out)); mb != "" {
			baseRef = mb
		}
	}
	if _, err := exec.CommandContext(runCtx, "git", "-C", dir, "rev-parse", "--verify", "--quiet", baseRef+"^{commit}").Output(); err != nil {
		baseRef = "HEAD"
	}
	parts := make([]string, 0, 2)
	if out, err := exec.CommandContext(runCtx, "git", "-C", dir, "diff", "--no-color", baseRef+"...HEAD").Output(); err == nil {
		parts = append(parts, strings.TrimRight(string(out), "\n"))
	}
	if out, err := exec.CommandContext(runCtx, "git", "-C", dir, "diff", "--no-color").Output(); err == nil {
		if text := strings.TrimRight(string(out), "\n"); text != "" {
			parts = append(parts, text)
		}
	}
	diff := strings.TrimSpace(strings.Join(parts, "\n"))
	if diff == "" {
		return "", false
	}
	const max = 1 << 20
	truncated := false
	if len(diff) > max {
		diff = diff[:max] + "\n\n[truncated]\n"
		truncated = true
	}
	return diff, truncated
}

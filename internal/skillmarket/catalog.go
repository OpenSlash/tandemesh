// Package skillmarket contains the discovery catalog for installable skills.
package skillmarket

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
)

// Entry is one skill available in the marketplace.
type Entry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Tags        []string `json:"tags"`
	Body        string   `json:"body"`
	Source      string   `json:"source,omitempty"`
	SourceURL   string   `json:"sourceUrl,omitempty"`
}

// Default is the starter catalog shipped with Orca.
func Default() []Entry {
	return []Entry{
		{
			ID: "code-review", Name: "code-review", Version: "1.0.0", Author: "Orca",
			Description: "Review a diff for correctness, edge cases and missing tests.",
			Tags:        []string{"review", "quality", "git"},
			Body: `# Code review

Review the requested diff. Focus on:
- correctness and regressions
- unhandled edge cases
- missing or weak tests
- security-sensitive changes
- unnecessary complexity

Report only actionable findings. For each finding include file, line, why it matters and a concrete fix. Do not rewrite unrelated code.`,
		},
		{
			ID: "test-first", Name: "test-first", Version: "1.0.0", Author: "Orca",
			Description: "Write a failing test first, then the smallest implementation.",
			Tags:        []string{"testing", "tdd", "quality"},
			Body: `# Test-first workflow

1. Restate the behavior as a small, observable contract.
2. Add a failing test that captures the contract.
3. Implement the smallest change that makes it pass.
4. Run the focused test, then the broader suite.
5. Refactor only with tests green.

Never claim completion without showing the test command and its result.`,
		},
		{
			ID: "security-review", Name: "security-review", Version: "1.0.0", Author: "Orca",
			Description: "Check authentication, authorization, secrets and injection risks.",
			Tags:        []string{"security", "review", "backend"},
			Body: `# Security review

Check for:
- missing authentication or authorization checks
- tenant/workspace boundary leaks
- secrets or tokens in logs, URLs or commits
- SQL/shell/path injection
- unsafe deserialization or file uploads
- overly broad CORS or redirects

For every issue state the attack path, impact and a safe fix. Flag uncertainty explicitly.`,
		},
		{
			ID: "git-commit", Name: "git-commit", Version: "1.0.0", Author: "Orca",
			Description: "Create small, reviewable commits with clear messages.",
			Tags:        []string{"git", "workflow"},
			Body: `# Git commit workflow

1. Inspect the working tree before staging.
2. Keep one logical change per commit.
3. Use a concise conventional subject such as feat(api): add machine metrics.
4. Put context and tradeoffs in the body.
5. Do not commit generated files, local databases or secrets.

Run the relevant tests before committing.`,
		},
		{
			ID: "debugging", Name: "debugging", Version: "1.0.0", Author: "Orca",
			Description: "Reproduce, isolate and verify a bug before fixing it.",
			Tags:        []string{"debugging", "quality"},
			Body: `# Debugging workflow

1. Reproduce the failure with the smallest command.
2. Capture the exact error and relevant inputs.
3. Form one hypothesis at a time and test it.
4. Fix the root cause, not the symptom.
5. Add a regression test and rerun the original reproduction.`,
		},
		{
			ID: "documentation", Name: "documentation", Version: "1.0.0", Author: "Orca",
			Description: "Write concise docs for setup, behavior and failure modes.",
			Tags:        []string{"docs", "writing"},
			Body: `# Documentation workflow

Document the reader's task, not the implementation history.

Include:
- prerequisites and one working setup path
- the smallest useful example
- configuration options and defaults
- failure modes and how to diagnose them
- links to deeper reference material

Keep examples runnable and verify commands where possible.`,
		},
	}
}

// Load reads a custom catalog from JSON, or returns the default catalog.
func Load(path string) ([]Entry, error) {
	if strings.TrimSpace(path) == "" {
		entries := Default()
		for i := range entries {
			entries[i].Source = "builtin"
		}
		return entries, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Load("")
		}
		return nil, err
	}
	var entries []Entry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	for i := range entries {
		if entries[i].Source == "" {
			entries[i].Source = "catalog"
		}
	}
	return entries, nil
}

// Search filters entries by name, description, author or tags.
func Search(entries []Entry, query string) []Entry {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return entries
	}
	out := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		text := strings.ToLower(entry.Name + " " + entry.Description + " " + entry.Author + " " + strings.Join(entry.Tags, " "))
		if strings.Contains(text, query) {
			out = append(out, entry)
		}
	}
	return out
}

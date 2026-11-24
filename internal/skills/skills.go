// Package skills loads SKILL.md playbooks from the user and repository scopes
// and renders them into an agent prompt.
package skills

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Skill is one playbook.
type Skill struct {
	Name        string
	Description string
	Version     string
	Tags        []string
	Body        string
	Path        string
}

// Parse reads a SKILL.md document. Frontmatter is optional but recommended.
func Parse(data []byte) (Skill, error) {
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	skill := Skill{}
	body := text

	if strings.HasPrefix(text, "---\n") {
		rest := text[4:]
		if idx := strings.Index(rest, "\n---"); idx >= 0 {
			front := rest[:idx]
			body = rest[idx+4:]
			for _, line := range strings.Split(front, "\n") {
				key, value, ok := strings.Cut(line, ":")
				if !ok {
					continue
				}
				switch strings.ToLower(strings.TrimSpace(key)) {
				case "name":
					skill.Name = strings.Trim(strings.TrimSpace(value), `"'`)
				case "description":
					skill.Description = strings.Trim(strings.TrimSpace(value), `"'`)
				case "version":
					skill.Version = strings.Trim(strings.TrimSpace(value), `"'`)
				case "tags":
					raw := strings.TrimSpace(value)
					raw = strings.Trim(raw, "[]")
					for _, tag := range strings.Split(raw, ",") {
						if tag = strings.Trim(strings.TrimSpace(tag), `"'`); tag != "" {
							skill.Tags = append(skill.Tags, tag)
						}
					}
				}
			}
		}
	}
	skill.Body = strings.TrimSpace(body)
	if skill.Name == "" {
		return Skill{}, errors.New("skill: missing name")
	}
	return skill, nil
}

// Discover loads skills from the given directories, optionally filtered by
// name. Each skill is a directory containing SKILL.md. Duplicate names keep the
// first occurrence, so repository scopes win when listed first.
func Discover(dirs []string, names []string, maxBytes int) ([]Skill, error) {
	if maxBytes <= 0 {
		maxBytes = 32 * 1024
	}
	wanted := map[string]bool{}
	for _, n := range names {
		wanted[strings.ToLower(strings.TrimSpace(n))] = true
	}

	seen := map[string]bool{}
	var out []Skill
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		candidates := []string{dir}
		for _, entry := range entries {
			if entry.IsDir() {
				candidates = append(candidates, filepath.Join(dir, entry.Name()))
			}
		}
		for _, candidate := range candidates {
			path := filepath.Join(candidate, "SKILL.md")
			raw, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			skill, err := Parse(raw)
			if err != nil {
				continue
			}
			key := strings.ToLower(skill.Name)
			if len(wanted) > 0 && !wanted[key] {
				continue
			}
			if seen[key] {
				continue
			}
			if len(skill.Body) > maxBytes {
				skill.Body = skill.Body[:maxBytes] + "\n\n[truncated]"
			}
			skill.Path = path
			seen[key] = true
			out = append(out, skill)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// Prompt renders skills as a prompt prefix, bounded by maxBytes.
func Prompt(items []Skill, maxBytes int) string {
	if len(items) == 0 {
		return ""
	}
	if maxBytes <= 0 {
		maxBytes = 32 * 1024
	}
	var b strings.Builder
	b.WriteString("Apply the following team skills where relevant.\n")
	for _, s := range items {
		fmt.Fprintf(&b, "\n## Skill: %s\n", s.Name)
		if s.Description != "" {
			fmt.Fprintf(&b, "%s\n", s.Description)
		}
		b.WriteString(s.Body)
		b.WriteString("\n")
		if b.Len() >= maxBytes {
			break
		}
	}
	text := b.String()
	if len(text) > maxBytes {
		text = text[:maxBytes] + "\n\n[skills truncated]"
	}
	return text
}

// Render renders a skill as a SKILL.md document with frontmatter.
func Render(skill Skill) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("name: " + skill.Name + "\n")
	if skill.Description != "" {
		b.WriteString("description: " + skill.Description + "\n")
	}
	if skill.Version != "" {
		b.WriteString("version: " + skill.Version + "\n")
	}
	if len(skill.Tags) > 0 {
		b.WriteString("tags: " + strings.Join(skill.Tags, ", ") + "\n")
	}
	b.WriteString("---\n\n")
	b.WriteString(skill.Body)
	b.WriteString("\n")
	return b.String()
}

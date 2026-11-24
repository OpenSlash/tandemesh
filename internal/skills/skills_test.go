package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeSkill(t *testing.T, root, dir, content string) {
	t.Helper()
	path := filepath.Join(root, dir)
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParse(t *testing.T) {
	skill, err := Parse([]byte("---\nname: deploy\ndescription: Ship it\n---\n\nRun the deploy script.\n"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if skill.Name != "deploy" || skill.Description != "Ship it" {
		t.Fatalf("skill = %+v", skill)
	}
	if !strings.Contains(skill.Body, "deploy script") {
		t.Fatalf("body = %q", skill.Body)
	}
}

func TestParseRequiresName(t *testing.T) {
	if _, err := Parse([]byte("no frontmatter")); err == nil {
		t.Fatal("missing name accepted")
	}
}

func TestDiscoverFiltersAndDeduplicates(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "deploy", "---\nname: deploy\ndescription: ship\n---\nbody-a")
	writeSkill(t, root, "review", "---\nname: review\ndescription: review\n---\nbody-b")
	writeSkill(t, root, "deploy-copy", "---\nname: deploy\ndescription: duplicate\n---\nbody-c")

	all, err := Discover([]string{root}, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("discovered %d skills", len(all))
	}

	one, err := Discover([]string{root}, []string{"review"}, 0)
	if err != nil || len(one) != 1 || one[0].Name != "review" {
		t.Fatalf("filtered = %+v err=%v", one, err)
	}

	prompt := Prompt(all, 0)
	if !strings.Contains(prompt, "Skill: deploy") || !strings.Contains(prompt, "Skill: review") {
		t.Fatalf("prompt = %q", prompt)
	}
}

func TestDiscoverMissingDir(t *testing.T) {
	items, err := Discover([]string{filepath.Join(t.TempDir(), "nope")}, nil, 0)
	if err != nil || len(items) != 0 {
		t.Fatalf("items = %+v err=%v", items, err)
	}
}

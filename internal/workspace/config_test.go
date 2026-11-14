package workspace

import "testing"

func TestParseSetupInline(t *testing.T) {
	src := "scripts:\n  setup: npm install --frozen-lockfile\n"
	if got := parseSetup(src); got != "npm install --frozen-lockfile" {
		t.Fatalf("parse = %q", got)
	}
}

func TestParseSetupBlock(t *testing.T) {
	src := "scripts:\n  setup: |\n    npm install\n    npm test\n"
	if got := parseSetup(src); got != "npm install\nnpm test" {
		t.Fatalf("parse = %q", got)
	}
}

func TestParseSetupMissing(t *testing.T) {
	if got := parseSetup("scripts:\n  test: npm test\n"); got != "" {
		t.Fatalf("parse = %q", got)
	}
}

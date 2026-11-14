package jira

import (
	"strings"
	"testing"
)

func TestADFToMarkdown(t *testing.T) {
	raw := []byte(`{"type":"doc","content":[{"type":"paragraph","content":[{"type":"text","text":"Done","marks":[{"type":"strong"}]}]},{"type":"bulletList","content":[{"type":"listItem","content":[{"type":"paragraph","content":[{"type":"text","text":"one"}]}]}]}]}`)
	got := adfToMarkdown(raw)
	if !strings.Contains(got, "**Done**") || !strings.Contains(got, "- one") {
		t.Fatalf("unexpected markdown: %q", got)
	}
}

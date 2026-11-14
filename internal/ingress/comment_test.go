package ingress

import (
	"testing"
)

const commentPayload = `{
  "action": "created",
  "issue": {
    "number": 7,
    "title": "flaky test",
    "body": "sometimes fails",
    "html_url": "https://github.com/acme/api/issues/7",
    "updated_at": "2026-10-31T00:00:00Z",
    "labels": [{"name": "agent:fix"}],
    "user": {"login": "alice"}
  },
  "comment": {"body": "/orca fix login", "user": {"login": "bob"}},
  "repository": {"full_name": "acme/api"},
  "sender": {"login": "bob"}
}`

func TestGitHubCommentCommand(t *testing.T) {
	p, err := ParseGitHub([]byte(commentPayload))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	work, snap, prompt, skip := NormalizeGitHub(p)
	if skip != "" {
		t.Fatalf("skip = %q", skip)
	}
	if work.ID != "7" || snap.Author != "bob" {
		t.Fatalf("work=%+v snap=%+v", work, snap)
	}
	if prompt == "" || !contains(prompt, "/orca fix login") {
		t.Fatalf("prompt = %q", prompt)
	}
}

func contains(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

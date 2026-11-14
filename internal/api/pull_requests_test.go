package api

import (
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestTaskKeyMatches(t *testing.T) {
	cases := []struct {
		key   string
		pr    integration.PullRequest
		match bool
	}{
		{"ENG-42", integration.PullRequest{Title: "fix ENG-42 checkout"}, true},
		{"ENG-42", integration.PullRequest{Branch: "feature/eng-42"}, true},
		{"ENG-42", integration.PullRequest{Title: "fix ENG-420"}, false},
		{"7", integration.PullRequest{Title: "fix #7"}, true},
		{"7", integration.PullRequest{Title: "fix #17"}, false},
		{"7", integration.PullRequest{Body: "closes /7"}, true},
	}
	for _, tc := range cases {
		task := &domain.Task{WorkID: tc.key}
		if got := taskKeyMatches(task, tc.pr); got != tc.match {
			t.Fatalf("key %q pr %+v = %v, want %v", tc.key, tc.pr, got, tc.match)
		}
	}
}

func TestTrustedPullRequest(t *testing.T) {
	if !trustedPullRequest(nil, integration.PullRequest{AuthorAssociation: "NONE"}) {
		t.Fatal("nil binding should allow")
	}
	binding := &domain.ResourceBinding{Settings: map[string]string{"trustedAuthorsOnly": "true"}}
	if trustedPullRequest(binding, integration.PullRequest{AuthorAssociation: "NONE"}) {
		t.Fatal("untrusted author should be rejected")
	}
	for _, association := range []string{"OWNER", "MEMBER", "COLLABORATOR"} {
		if !trustedPullRequest(binding, integration.PullRequest{AuthorAssociation: association}) {
			t.Fatalf("association %s should be allowed", association)
		}
	}
}

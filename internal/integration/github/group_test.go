package github

import "testing"

func TestIssueGroup(t *testing.T) {
	tests := map[[2]string]string{
		{"open", ""}:              "unstarted",
		{"closed", "completed"}:   "completed",
		{"closed", ""}:            "completed",
		{"closed", "not_planned"}: "cancelled",
	}
	for in, want := range tests {
		if got := issueGroup(in[0], in[1]); got != want {
			t.Errorf("issueGroup(%q, %q) = %q, want %q", in[0], in[1], got, want)
		}
	}
}

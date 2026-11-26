package main

import (
	"os"
	"strings"
	"testing"
)

func TestMergePathLists(t *testing.T) {
	sep := string(os.PathListSeparator)
	tests := []struct{ primary, extra, want string }{
		{"/a" + sep + "/b", "/b" + sep + "/c", "/a" + sep + "/b" + sep + "/c"},
		{"", "/usr/bin", "/usr/bin"},
		{"/a" + sep + sep + "/a", "", "/a"},
	}
	for _, tc := range tests {
		if got := mergePathLists(tc.primary, tc.extra); got != tc.want {
			t.Errorf("mergePathLists(%q, %q) = %q, want %q", tc.primary, tc.extra, got, tc.want)
		}
	}
	if got := mergePathLists("/x", "/usr/bin"); !strings.HasPrefix(got, "/x") {
		t.Errorf("shell PATH should come first: %q", got)
	}
}

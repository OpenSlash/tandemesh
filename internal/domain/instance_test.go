package domain

import "testing"

func TestInstanceCovers(t *testing.T) {
	tests := []struct {
		scope, instance string
		want            bool
	}{
		{"acme/p1", "acme/p1", true},
		{"acme", "acme/p1", true},
		{"acme", "acme", true},
		{"acme", "acmecorp/p1", false},
		{"acme/p1", "acme/p2", false},
		{"", "acme/p1", false},
	}
	for _, tt := range tests {
		if got := InstanceCovers(tt.scope, tt.instance); got != tt.want {
			t.Errorf("InstanceCovers(%q, %q) = %v, want %v", tt.scope, tt.instance, got, tt.want)
		}
	}
}

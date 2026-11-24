package api

import "testing"

func TestActorAllowed(t *testing.T) {
	if !actorAllowed(nil, "alice") {
		t.Fatal("empty allowlist should allow")
	}
	if !actorAllowed([]string{"Alice"}, "alice") {
		t.Fatal("allowlist should be case-insensitive")
	}
	if actorAllowed([]string{"bob"}, "alice") {
		t.Fatal("unlisted actor must be rejected")
	}
}

func TestCommentAllowlist(t *testing.T) {
	if got := commentAllowlist(map[string]any{"commentUsers": "alice, bob"}); len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	if got := commentAllowlist(map[string]any{"commentUsers": []any{"alice"}}); len(got) != 1 {
		t.Fatalf("got %v", got)
	}
}

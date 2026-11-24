package api

import (
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/config"
)

func TestOAuthStateSignature(t *testing.T) {
	s := &Server{cfg: config.Config{SessionKey: "test-key"}}
	state := githubOAuthState{
		Workspace: "ws_default", Instance: "acme/api", Name: "gh",
		Nonce: "n1", Expires: time.Now().Add(time.Minute).Unix(),
	}
	signed, err := s.signOAuthState(state)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := s.verifyOAuthState(signed)
	if !ok || got.Instance != "acme/api" || got.Workspace != "ws_default" {
		t.Fatalf("state = %+v ok=%v", got, ok)
	}
	if _, ok := s.verifyOAuthState(signed + "x"); ok {
		t.Fatal("tampered state accepted")
	}
	expired := state
	expired.Expires = time.Now().Add(-time.Minute).Unix()
	signed, _ = s.signOAuthState(expired)
	if _, ok := s.verifyOAuthState(signed); ok {
		t.Fatal("expired state accepted")
	}
}

package ingress

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyGitHub(t *testing.T) {
	body := []byte(`{"action":"opened"}`)
	secret := "s3cret"

	if err := VerifyGitHub(secret, body, sign(secret, body)); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	if err := VerifyGitHub(secret, body, sign("other", body)); err == nil {
		t.Fatal("invalid signature accepted")
	}
	if err := VerifyGitHub(secret, body, "sha256=zz"); err == nil {
		t.Fatal("malformed signature accepted")
	}
	if err := VerifyGitHub("", body, ""); err != nil {
		t.Fatalf("development mode should skip verification: %v", err)
	}
}

func TestNormalizeGitHub(t *testing.T) {
	payload := &GitHubPayload{Action: "opened"}
	payload.Issue.Number = 42
	payload.Issue.Title = "flaky test"
	payload.Issue.HTMLURL = "https://github.com/acme/api/issues/42"
	payload.Issue.UpdatedAt = "2026-10-30T00:00:00Z"
	payload.Repository.FullName = "acme/api"
	payload.Sender.Login = "alice"
	payload.Issue.State = "open"
	payload.Issue.Assignee.Login = "alice"

	work, snap, prompt, skip := NormalizeGitHub(payload)
	if skip != "" {
		t.Fatalf("unexpected skip: %s", skip)
	}
	if work.Provider != "github" || work.ID != "42" || work.Instance != "acme/api" {
		t.Fatalf("bad work ref: %+v", work)
	}
	if snap.Title != "flaky test" || snap.Author != "alice" || snap.Status != "open" || snap.StatusGroup != "unstarted" || snap.Assignee != "alice" {
		t.Fatalf("bad snapshot: %+v", snap)
	}
	if prompt == "" {
		t.Fatal("empty prompt")
	}

	payload.Action = "deleted"
	if _, _, _, skip := NormalizeGitHub(payload); skip != "SKIP_EVENT_ACTION" {
		t.Fatalf("expected skip, got %q", skip)
	}
}

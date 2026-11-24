package jira

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

const body = `{
  "webhookEvent": "jira:issue_updated",
  "issue": {
    "id": "10001",
    "key": "ACME-7",
    "self": "https://acme.atlassian.net/rest/api/2/issue/10001",
    "fields": {
      "summary": "fix flaky build",
      "labels": ["agent:fix"],
      "updated": "2026-10-31T00:00:00.000+0000",
      "assignee": {"displayName": "alice"}
    }
  },
  "user": {"displayName": "bob"}
}`

func sign(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyAndNormalize(t *testing.T) {
	a := &Adapter{Secret: "s3cret"}
	req := httptest.NewRequest(http.MethodPost, "/webhooks/jira/b", strings.NewReader(body))
	req.Header.Set("X-Hub-Signature", sign("s3cret", body))
	if err := a.Verify(req, []byte(body)); err != nil {
		t.Fatalf("verify: %v", err)
	}
	bad := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	bad.Header.Set("X-Hub-Signature", sign("other", body))
	if err := a.Verify(bad, []byte(body)); err == nil {
		t.Fatal("bad signature accepted")
	}

	ev, err := a.Parse([]byte(body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	work, snap, prompt, skip, err := a.Normalize(ev)
	if err != nil || skip != "" {
		t.Fatalf("normalize: %v %q", err, skip)
	}
	if work.Provider != "jira" || work.ID != "ACME-7" || work.Instance != "https://acme.atlassian.net" {
		t.Fatalf("work = %+v", work)
	}
	if snap.Title != "fix flaky build" || prompt == "" {
		t.Fatalf("snap = %+v prompt = %q", snap, prompt)
	}
}

func TestCommentAndTransition(t *testing.T) {
	var commentAuth, transitionBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/comment"):
			commentAuth = r.Header.Get("Authorization")
			_, _ = w.Write([]byte(`{"id":"1","self":"https://acme.atlassian.net/rest/api/2/issue/ACME-7/comment/1"}`))
		case strings.HasSuffix(r.URL.Path, "/transitions") && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"transitions":[{"id":"31","name":"Done"}]}`))
		case strings.HasSuffix(r.URL.Path, "/transitions"):
			raw, _ := io.ReadAll(r.Body)
			transitionBody = string(raw)
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer srv.Close()

	a := &Adapter{HTTPClient: srv.Client()}
	work := domain.WorkRef{Provider: "jira", Instance: srv.URL, Type: "issue", ID: "ACME-7"}
	cred := integration.Credential{Subject: "dev@example.com", AccessToken: "tok"}

	url, err := a.Comment(t.Context(), work, cred, "hello")
	if err != nil || url == "" {
		t.Fatalf("comment: %v %q", err, url)
	}
	if !strings.HasPrefix(commentAuth, "Basic ") {
		t.Fatalf("auth = %q", commentAuth)
	}
	if err := a.Transition(t.Context(), work, cred, "Done"); err != nil {
		t.Fatalf("transition: %v", err)
	}
	if !strings.Contains(transitionBody, `"id":"31"`) {
		t.Fatalf("transition body = %q", transitionBody)
	}
	if err := a.Transition(t.Context(), work, cred, "Unknown"); err == nil {
		t.Fatal("unknown transition accepted")
	}
}

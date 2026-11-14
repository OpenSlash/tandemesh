package linear

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

const payloadBody = `{
  "action": "create",
  "type": "Issue",
  "organizationId": "org_acme",
  "data": {
    "id": "issue_123",
    "identifier": "ACME-7",
    "title": "flaky build",
    "url": "https://linear.app/acme/issue/ACME-7",
    "updatedAt": "2026-10-31T00:00:00Z",
    "labels": [{"name": "agent:fix"}],
    "assignee": {"name": "alice"},
    "team": {"key": "ACME"}
  },
  "actor": {"name": "bob"}
}`

func sign(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	a := &Adapter{Secret: "s3cret"}
	req := httptest.NewRequest(http.MethodPost, "/webhooks/linear/b", strings.NewReader(payloadBody))
	req.Header.Set("Linear-Signature", sign("s3cret", payloadBody))
	if err := a.Verify(req, []byte(payloadBody)); err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
	bad := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(payloadBody))
	bad.Header.Set("Linear-Signature", sign("other", payloadBody))
	if err := a.Verify(bad, []byte(payloadBody)); err == nil {
		t.Fatal("invalid signature accepted")
	}
}

func TestParseAndNormalize(t *testing.T) {
	a := &Adapter{}
	ev, err := a.Parse([]byte(payloadBody))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if ev.Kind != "create" || ev.Actor != "bob" || len(ev.Labels) != 1 {
		t.Fatalf("event = %+v", ev)
	}
	work, snap, prompt, skip, err := a.Normalize(ev)
	if err != nil || skip != "" {
		t.Fatalf("normalize: %v %q", err, skip)
	}
	if work.Provider != "linear" || work.ID != "issue_123" || work.Instance != "org_acme" {
		t.Fatalf("work = %+v", work)
	}
	if snap.Title != "flaky build" || prompt == "" {
		t.Fatalf("snapshot = %+v prompt = %q", snap, prompt)
	}
}

func TestComment(t *testing.T) {
	var gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"commentCreate":{"success":true,"comment":{"url":"https://linear.app/comment/1"}}}}`))
	}))
	defer srv.Close()

	a := &Adapter{BaseURL: srv.URL, HTTPClient: srv.Client()}
	url, err := a.Comment(t.Context(), domain.WorkRef{ID: "issue_123"}, integration.Credential{AccessToken: "tok"}, "hello")
	if err != nil {
		t.Fatalf("comment: %v", err)
	}
	if url != "https://linear.app/comment/1" {
		t.Fatalf("url = %q", url)
	}
	if gotAuth != "tok" || !strings.Contains(gotBody, "commentCreate") {
		t.Fatalf("request auth=%q body=%q", gotAuth, gotBody)
	}
}

package slack

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestSend(t *testing.T) {
	var gotAuth, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		raw, _ := io.ReadAll(r.Body)
		gotBody = string(raw)
		_, _ = w.Write([]byte(`{"ok":true,"channel":"C123","ts":"111.222"}`))
	}))
	defer srv.Close()

	a := &Adapter{BaseURL: srv.URL, HTTPClient: srv.Client()}
	ref, err := a.Send(t.Context(), integration.Credential{AccessToken: "xoxb-test"}, integration.ChatMessage{Channel: "C123", Text: "hello"})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if ref != "C123:111.222" {
		t.Fatalf("ref = %q", ref)
	}
	if gotAuth != "Bearer xoxb-test" || !strings.Contains(gotBody, `"channel":"C123"`) {
		t.Fatalf("request auth=%q body=%q", gotAuth, gotBody)
	}
}

func TestSendError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ok":false,"error":"channel_not_found"}`))
	}))
	defer srv.Close()

	a := &Adapter{BaseURL: srv.URL, HTTPClient: srv.Client()}
	if _, err := a.Send(t.Context(), integration.Credential{AccessToken: "x"}, integration.ChatMessage{Channel: "bad"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestRequestApprovalCard(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		_, _ = w.Write([]byte(`{"ok":true,"channel":"C1","ts":"9.9"}`))
	}))
	defer srv.Close()

	a := &Adapter{BaseURL: srv.URL, HTTPClient: srv.Client()}
	ref, err := a.RequestApproval(t.Context(), integration.Credential{AccessToken: "x"}, integration.ApprovalCard{
		ApprovalID: "apr_1", Channel: "C1", Tool: "bash", Command: "git push", Risk: "high",
		ApproveURL: "https://orca.test/approve", DenyURL: "https://orca.test/deny",
	})
	if err != nil {
		t.Fatalf("request approval: %v", err)
	}
	if ref != "C1:9.9" {
		t.Fatalf("ref = %q", ref)
	}
	for _, want := range []string{"blocks", "Approve", "Deny", "https://orca.test/approve"} {
		if !strings.Contains(body, want) {
			t.Fatalf("payload missing %q: %s", want, body)
		}
	}
}

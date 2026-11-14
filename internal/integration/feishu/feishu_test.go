package feishu

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestSendAndApprovalCard(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		_, _ = w.Write([]byte(`{"code":0,"msg":"success"}`))
	}))
	defer srv.Close()

	a := &Adapter{Webhook: srv.URL, HTTPClient: srv.Client()}
	if _, err := a.Send(t.Context(), integration.Credential{}, integration.ChatMessage{Text: "hello"}); err != nil {
		t.Fatalf("send: %v", err)
	}
	if !strings.Contains(body, `"msg_type":"text"`) {
		t.Fatalf("text payload = %s", body)
	}

	if _, err := a.RequestApproval(t.Context(), integration.Credential{}, integration.ApprovalCard{
		Tool: "bash", Command: "git push", Risk: "high",
		ApproveURL: "https://orca.test/a", DenyURL: "https://orca.test/d",
	}); err != nil {
		t.Fatalf("approval: %v", err)
	}
	for _, want := range []string{"interactive", "Approve", "Deny", "https://orca.test/a"} {
		if !strings.Contains(body, want) {
			t.Fatalf("card missing %q: %s", want, body)
		}
	}
}

func TestWebhookError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":19001,"msg":"bad webhook"}`))
	}))
	defer srv.Close()

	a := &Adapter{Webhook: srv.URL, HTTPClient: srv.Client()}
	if _, err := a.Send(t.Context(), integration.Credential{}, integration.ChatMessage{Text: "x"}); err == nil {
		t.Fatal("expected error")
	}

	empty := &Adapter{}
	if _, err := empty.Send(t.Context(), integration.Credential{}, integration.ChatMessage{Text: "x"}); err == nil {
		t.Fatal("expected unconfigured error")
	}
}

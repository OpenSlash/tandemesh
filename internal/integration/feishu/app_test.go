package feishu

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestFeishuAppSend(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/open-apis/auth/v3/tenant_access_token/internal":
			_, _ = w.Write([]byte(`{"code":0,"tenant_access_token":"t-1"}`))
		case "/open-apis/im/v1/messages":
			if r.Header.Get("Authorization") != "Bearer t-1" {
				t.Errorf("authorization = %s", r.Header.Get("Authorization"))
			}
			_, _ = w.Write([]byte(`{"code":0,"data":{"message_id":"om_1"}}`))
		default:
			t.Errorf("path = %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	a := &Adapter{BaseURL: srv.URL, HTTPClient: srv.Client()}
	cred := integration.Credential{Fields: map[string]string{"appId": "cli_1", "appSecret": "secret", "chatId": "oc_1"}}
	if err := a.Probe(t.Context(), cred); err != nil {
		t.Fatal(err)
	}
	id, err := a.Send(t.Context(), cred, integration.ChatMessage{Text: "hello"})
	if err != nil || id != "om_1" {
		t.Fatalf("id = %s, err = %v", id, err)
	}
}

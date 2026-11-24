package jira

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestListComments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/2/issue/ENG-1/comment" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"comments":[{"id":"1","body":"hello","created":"2026-01-01T00:00:00.000+0000","author":{"displayName":"Alice"}}]}`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client()}
	items, err := a.ListComments(t.Context(), integration.Credential{AccessToken: "tok"}, domain.WorkRef{Instance: srv.URL, ID: "ENG-1"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Body != "hello" || items[0].Author != "Alice" {
		t.Fatalf("items = %+v", items)
	}
}

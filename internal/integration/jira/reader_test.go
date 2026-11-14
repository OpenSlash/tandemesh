package jira

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestListAssignedCustomFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Query().Get("fields"), "customfield_10010") {
			t.Errorf("fields = %s", r.URL.Query().Get("fields"))
		}
		_, _ = w.Write([]byte(`{"issues":[{"key":"ENG-1","fields":{"summary":"Fix","updated":"2026-01-01T00:00:00.000+0000","status":{"name":"Done","statusCategory":{"key":"done"}},"customfield_10010":{"value":"5"}}}]}`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client()}
	items, err := a.ListAssigned(t.Context(), integration.Credential{AccessToken: "tok"}, integration.ListOptions{
		Instance: srv.URL, Limit: 10, Fields: map[string]string{"customfield_10010": "Story Points"},
	})
	if err != nil || len(items) != 1 {
		t.Fatalf("items = %+v, err = %v", items, err)
	}
	if got := items[0].Fields["Story Points"]; len(got) != 1 || got[0] != "5" {
		t.Fatalf("custom = %+v", items[0].Fields)
	}
}

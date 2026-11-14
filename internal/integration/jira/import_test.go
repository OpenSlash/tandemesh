package jira

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestSearchIssues(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/2/search" || r.URL.Query().Get("jql") != "project = ENG" {
			t.Errorf("request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"issues":[{"key":"ENG-1","fields":{"summary":"Fix","updated":"2026-01-01T00:00:00.000+0000","status":{"name":"Done","statusCategory":{"key":"done"}},"attachment":[{"id":"a1","filename":"spec.pdf","content":"https://example.test/spec.pdf","mimeType":"application/pdf","size":12}]}}]}`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client()}
	items, err := a.SearchIssues(t.Context(), integration.Credential{AccessToken: "tok"}, srv.URL, "project = ENG", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Ref.ID != "ENG-1" || items[0].StatusGroup != "completed" || len(items[0].Attachments) != 1 {
		t.Fatalf("items = %+v", items)
	}
}

func TestSearchIssuesRetriesRateLimit(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"issues":[]}`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client()}
	if _, err := a.SearchIssues(t.Context(), integration.Credential{AccessToken: "tok"}, srv.URL, "project = ENG", 10); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d", calls)
	}
}

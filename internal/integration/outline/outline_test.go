package outline

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestSearchAndFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/documents.search":
			_, _ = w.Write([]byte(`{"data":[{"document":{"id":"doc-1","title":"Runbook","url":"/doc/runbook-abc123"},"context":"restart steps"}]}`))
		case "/api/documents.info":
			_, _ = w.Write([]byte(`{"data":{"id":"abc123","title":"Runbook","text":"# Restart\nDo the thing.","url":"/doc/runbook-abc123","updatedAt":"2026-10-30T00:00:00Z"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	a := &Adapter{BaseURL: srv.URL, HTTPClient: srv.Client()}
	cred := integration.Credential{Provider: "outline", AccessToken: "tok"}

	hits, err := a.Search(t.Context(), cred, "runbook", 5)
	if err != nil || len(hits) != 1 {
		t.Fatalf("hits = %+v err=%v", hits, err)
	}
	if hits[0].ID != "doc-1" || !strings.HasPrefix(hits[0].URL, srv.URL+"/doc/") {
		t.Fatalf("hit = %+v", hits[0])
	}

	doc, err := a.Fetch(t.Context(), cred, "abc123")
	if err != nil || doc.Title != "Runbook" || !strings.Contains(doc.Text, "Restart") {
		t.Fatalf("doc = %+v err=%v", doc, err)
	}
	byURL, err := a.Fetch(t.Context(), cred, srv.URL+"/doc/runbook-abc123")
	if err != nil || byURL.Text == "" {
		t.Fatalf("fetch by url: %+v err=%v", byURL, err)
	}
}

func TestUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	a := &Adapter{BaseURL: srv.URL, HTTPClient: srv.Client()}
	if _, err := a.Search(t.Context(), integration.Credential{AccessToken: "bad"}, "q", 1); err == nil {
		t.Fatal("unauthorized accepted")
	}
}

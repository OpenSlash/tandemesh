package notion

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestSearchAndFetch(t *testing.T) {
	const pageID32 = "0123456789abcdef0123456789abcdef"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1/search":
			_, _ = w.Write([]byte(`{"results":[{"object":"page","id":"` + pageID32 + `","url":"https://notion.so/Page-` + pageID32 + `","properties":{"title":{"title":[{"plain_text":"Design doc"}]}}}]}`))
		case strings.HasPrefix(r.URL.Path, "/v1/pages/"):
			_, _ = w.Write([]byte(`{"properties":{"title":{"title":[{"plain_text":"Design doc"}]}}}`))
		case strings.HasSuffix(r.URL.Path, "/children"):
			_, _ = w.Write([]byte(`{"results":[{"id":"b1","type":"paragraph"}]}`))
		case strings.HasPrefix(r.URL.Path, "/v1/blocks/"):
			_, _ = w.Write([]byte(`{"type":"paragraph","paragraph":{"rich_text":[{"plain_text":"Hello world"}]}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	a := &Adapter{BaseURL: srv.URL, HTTPClient: srv.Client()}
	cred := integration.Credential{AccessToken: "tok"}

	hits, err := a.Search(t.Context(), cred, "design", 5)
	if err != nil || len(hits) != 1 || hits[0].Title != "Design doc" {
		t.Fatalf("hits = %+v err=%v", hits, err)
	}
	if !strings.Contains(hits[0].URL, pageID32) {
		t.Fatalf("url = %q", hits[0].URL)
	}

	doc, err := a.Fetch(t.Context(), cred, hits[0].URL)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if doc.Title != "Design doc" || !strings.Contains(doc.Text, "Hello world") {
		t.Fatalf("doc = %+v", doc)
	}
}

func TestFetchRejectsBadID(t *testing.T) {
	a := &Adapter{}
	if _, err := a.Fetch(t.Context(), integration.Credential{}, ""); err == nil {
		t.Fatal("empty page accepted")
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

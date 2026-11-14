package outlook

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestSearchAndFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v1.0/me/messages":
			if !strings.Contains(r.URL.Query().Get("$search"), "deploy") {
				t.Errorf("search query = %q", r.URL.Query().Get("$search"))
			}
			_, _ = w.Write([]byte(`{"value":[{"id":"AAMkAD123","subject":"Deploy notes","webLink":"https://outlook.office.com/mail/id/AAMkAD123","bodyPreview":"rollback plan"}]}`))
		case strings.HasPrefix(r.URL.Path, "/v1.0/me/messages/"):
			_, _ = w.Write([]byte(`{"subject":"Deploy notes","webLink":"https://outlook.office.com/mail/id/AAMkAD123","body":{"contentType":"html","content":"<p>Rollback <b>plan</b>&nbsp;here</p>"},"lastModifiedDateTime":"2026-10-30T10:00:00Z"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	a := &Adapter{BaseURL: srv.URL, HTTPClient: srv.Client()}
	hits, err := a.Search(t.Context(), integrationCred(), "deploy", 5)
	if err != nil || len(hits) != 1 {
		t.Fatalf("hits = %+v err=%v", hits, err)
	}
	if hits[0].ID != "AAMkAD123" || hits[0].Title != "Deploy notes" {
		t.Fatalf("hit = %+v", hits[0])
	}

	doc, err := a.Fetch(t.Context(), integrationCred(), "outlook:AAMkAD123")
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if doc.Title != "Deploy notes" || strings.Contains(doc.Text, "<") {
		t.Fatalf("doc = %+v", doc)
	}
	if !strings.Contains(doc.Text, "Rollback plan here") {
		t.Fatalf("text = %q", doc.Text)
	}
}

func TestFetchRejectsWebLink(t *testing.T) {
	a := &Adapter{}
	if _, err := a.Fetch(t.Context(), integrationCred(), "https://outlook.office.com/mail/id/x"); err == nil {
		t.Fatal("web link accepted as id")
	}
}

func integrationCred() integration.Credential {
	return integration.Credential{Provider: "outlook", AccessToken: "tok"}
}

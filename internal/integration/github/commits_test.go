package github

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestListCommits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/api/commits" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"sha":"abc123","html_url":"https://github.test/acme/api/commit/abc123","commit":{"message":"fix ENG-42","author":{"name":"Alice"}},"author":{"login":"alice"}}]`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client(), BaseURL: srv.URL}
	commits, err := a.ListCommits(t.Context(), integration.Credential{AccessToken: "tok"}, "acme/api", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 1 || commits[0].SHA != "abc123" || commits[0].Author != "alice" {
		t.Fatalf("commits = %+v", commits)
	}
}

func TestListCommitsForRef(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/api/commits" || r.URL.Query().Get("sha") != "feature" {
			t.Errorf("url = %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`[{"sha":"def456","html_url":"https://github.test/acme/api/commit/def456","commit":{"message":"fix ENG-42","author":{"name":"Alice"}},"author":{"login":"alice"}}]`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client(), BaseURL: srv.URL}
	commits, err := a.ListCommitsForRef(t.Context(), integration.Credential{AccessToken: "tok"}, "acme/api", "feature", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 1 || commits[0].SHA != "def456" {
		t.Fatalf("commits = %+v", commits)
	}
}

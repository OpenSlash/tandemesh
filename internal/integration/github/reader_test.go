package github

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestListAssigned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/issues" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("filter") != "assigned" {
			t.Errorf("filter = %q", r.URL.Query().Get("filter"))
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("auth = %q", r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`[{"number":7,"title":"flaky test","body":"details","html_url":"https://github.com/acme/api/issues/7","repository_url":"https://api.github.com/repos/acme/api","updated_at":"2026-10-30T10:00:00Z","labels":[{"name":"agent:fix"}],"assignee":{"login":"alice"}}]`))
	}))
	defer srv.Close()

	a := &Adapter{BaseURL: srv.URL, HTTPClient: srv.Client()}
	items, err := a.ListAssigned(t.Context(), integration.Credential{AccessToken: "tok"}, integration.ListOptions{Limit: 5})
	if err != nil || len(items) != 1 {
		t.Fatalf("items = %+v err=%v", items, err)
	}
	item := items[0]
	if item.Ref.Instance != "acme/api" || item.Ref.ID != "7" || item.Title != "flaky test" {
		t.Fatalf("item = %+v", item)
	}
	if len(item.Labels) != 1 || item.Labels[0] != "agent:fix" || item.Assignee != "alice" {
		t.Fatalf("labels/assignee = %+v", item)
	}
}

func TestListAssignedForInstance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			_, _ = w.Write([]byte(`{"login":"alice"}`))
		case "/repos/acme/api/issues":
			if got := r.URL.Query().Get("assignee"); got != "alice" {
				t.Errorf("assignee = %q", got)
			}
			_, _ = w.Write([]byte(`[{"number":7,"title":"flaky test","body":"details","html_url":"https://github.com/acme/api/issues/7","repository_url":"https://api.github.com/repos/acme/api","updated_at":"2026-10-30T10:00:00Z","labels":[{"name":"agent:fix"}],"assignee":{"login":"alice"}},{"number":8,"title":"a pull request","pull_request":{"url":"x"}}]`))
		default:
			t.Errorf("path = %q", r.URL.Path)
		}
	}))
	defer srv.Close()

	a := &Adapter{BaseURL: srv.URL, HTTPClient: srv.Client()}
	items, err := a.ListAssigned(t.Context(), integration.Credential{AccessToken: "tok"}, integration.ListOptions{Limit: 5, Instance: "acme/api"})
	if err != nil || len(items) != 1 {
		t.Fatalf("items = %+v err=%v", items, err)
	}
	if items[0].Ref.Instance != "acme/api" || items[0].Ref.ID != "7" {
		t.Fatalf("item = %+v", items[0])
	}
}

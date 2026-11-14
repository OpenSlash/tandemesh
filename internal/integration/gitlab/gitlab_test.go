package gitlab

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestListAssigned(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/issues" || r.URL.Query().Get("scope") != "assigned_to_me" {
			t.Errorf("request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`[{"iid":7,"title":"Fix","description":"body","state":"opened","labels":["bug"],"web_url":"https://gitlab.test/acme/api/-/issues/7","updated_at":"2026-01-01T00:00:00Z","author":{"name":"Alice"},"assignees":[{"name":"Bob"}]}]`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client(), BaseURL: srv.URL}
	items, err := a.ListAssigned(t.Context(), integration.Credential{AccessToken: "tok"}, integration.ListOptions{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Ref.Instance != "acme/api" || items[0].Ref.ID != "7" || items[0].Assignee != "Bob" {
		t.Fatalf("items = %+v", items)
	}
}

func TestCommentAndTransition(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch calls {
		case 1:
			if r.URL.Path != "/api/v4/projects/acme/api/issues/7/notes" {
				t.Errorf("comment path = %s", r.URL.Path)
			}
			_, _ = w.Write([]byte(`{"id":99}`))
		case 2:
			if r.URL.Path != "/api/v4/projects/acme/api/issues/7" {
				t.Errorf("transition path = %s", r.URL.Path)
			}
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client(), BaseURL: srv.URL}
	ref := domain.WorkRef{Instance: "acme/api", ID: "7", URL: "https://gitlab.test/acme/api/-/issues/7"}
	url, err := a.Comment(t.Context(), ref, integration.Credential{AccessToken: "tok"}, "hello")
	if err != nil || url != "https://gitlab.test/acme/api/-/issues/7#note_99" {
		t.Fatalf("url = %s, err = %v", url, err)
	}
	if err := a.Transition(t.Context(), ref, integration.Credential{AccessToken: "tok"}, "done"); err != nil {
		t.Fatal(err)
	}
}

func TestListPullRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/projects/acme/api/merge_requests" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"iid":3,"title":"MR","state":"opened","web_url":"https://gitlab.test/mr/3","source_branch":"ENG-7","author":{"name":"Alice"}}]`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client(), BaseURL: srv.URL}
	items, err := a.ListPullRequests(t.Context(), integration.Credential{AccessToken: "tok"}, "acme/api")
	if err != nil || len(items) != 1 || items[0].Number != 3 {
		t.Fatalf("items = %+v, err = %v", items, err)
	}
}

func TestListCommits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/projects/acme/api/repository/commits" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"id":"abc123","short_id":"abc123","title":"ENG-7 fix","message":"ENG-7 fix","web_url":"https://gitlab.test/commit/abc123","author_name":"Alice"}]`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client(), BaseURL: srv.URL}
	items, err := a.ListCommits(t.Context(), integration.Credential{AccessToken: "tok"}, "acme/api", 10)
	if err != nil || len(items) != 1 || items[0].SHA != "abc123" || items[0].Author != "Alice" {
		t.Fatalf("items = %+v, err = %v", items, err)
	}
}

func TestSearchIssues(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/projects/acme/api/issues" || r.URL.Query().Get("search") != "bug" {
			t.Errorf("request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`[{"iid":7,"title":"Fix","state":"opened","web_url":"https://gitlab.test/issues/7","updated_at":"2026-01-01T00:00:00Z","author":{"name":"Alice"}}]`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client(), BaseURL: srv.URL}
	items, err := a.SearchIssues(t.Context(), integration.Credential{AccessToken: "tok"}, "acme/api", "bug", 10)
	if err != nil || len(items) != 1 || items[0].Ref.ID != "7" {
		t.Fatalf("items = %+v, err = %v", items, err)
	}
}

func TestListComments(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v4/projects/acme/api/issues/7/notes" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"id":1,"body":"hello ![s](/uploads/a/s.pdf)","system":false,"author":{"name":"Alice"},"created_at":"2026-01-01T00:00:00Z"},{"id":2,"body":"changed","system":true,"author":{"name":"System"},"created_at":"2026-01-01T00:00:01Z"}]`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client(), BaseURL: srv.URL}
	items, err := a.ListComments(t.Context(), integration.Credential{AccessToken: "tok"}, domain.WorkRef{Instance: "acme/api", ID: "7"})
	if err != nil || len(items) != 1 || items[0].Body != "hello ![s](/uploads/a/s.pdf)" || items[0].Author != "Alice" || len(items[0].Attachments) != 1 {
		t.Fatalf("items = %+v, err = %v", items, err)
	}
}

func TestAttachmentsFromDescription(t *testing.T) {
	items := attachmentsFromDescription("see ![spec](/uploads/abc/spec.pdf) and [log](/uploads/def/build.log)", "https://gitlab.test/acme/api")
	if len(items) != 2 {
		t.Fatalf("items = %+v", items)
	}
	if items[0].URL != "https://gitlab.test/acme/api/uploads/abc/spec.pdf" || items[0].Name != "spec.pdf" {
		t.Fatalf("item = %+v", items[0])
	}
}

func TestOpenAttachment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = w.Write([]byte("pdf"))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client()}
	body, contentType, err := a.OpenAttachment(t.Context(), integration.Credential{AccessToken: "tok"}, integration.Attachment{URL: srv.URL + "/uploads/a/spec.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	defer body.Close()
	raw, _ := io.ReadAll(body)
	if string(raw) != "pdf" || contentType != "application/pdf" {
		t.Fatalf("body = %q, type = %s", raw, contentType)
	}
}

package github

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestListPullRequests(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/api/pulls" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"number":7,"html_url":"https://github.test/acme/api/pull/7","title":"fix ENG-42","body":"body","state":"open","head":{"ref":"eng-42"},"user":{"login":"alice"}}]`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client(), BaseURL: srv.URL}
	pulls, err := a.ListPullRequests(t.Context(), integration.Credential{AccessToken: "tok"}, "acme/api")
	if err != nil {
		t.Fatal(err)
	}
	if len(pulls) != 1 || pulls[0].Number != 7 || pulls[0].Branch != "eng-42" || pulls[0].Author != "alice" {
		t.Fatalf("pulls = %+v", pulls)
	}
}

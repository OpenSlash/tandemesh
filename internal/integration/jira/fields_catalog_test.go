package jira

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestListFieldOptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/rest/api/2/priority":
			_, _ = w.Write([]byte(`[{"name":"High"},{"name":"Low"}]`))
		case "/rest/api/2/project/ENG/components":
			_, _ = w.Write([]byte(`[{"name":"API"}]`))
		case "/rest/api/2/project/ENG/versions":
			_, _ = w.Write([]byte(`[{"name":"1.2"}]`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client()}
	options, err := a.ListFieldOptions(t.Context(), integration.Credential{AccessToken: "tok"}, srv.URL, "ENG")
	if err != nil || len(options["priority"]) != 2 || options["components"][0] != "API" {
		t.Fatalf("options = %+v, err = %v", options, err)
	}
}

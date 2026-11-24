package jira

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestListStates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/2/status" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[{"id":"1","name":"To Do","statusCategory":{"key":"new"}},{"id":"3","name":"In Progress","statusCategory":{"key":"indeterminate"}}]`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client()}
	states, err := a.ListStates(t.Context(), integration.Credential{AccessToken: "tok"}, srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(states) != 2 || states[1].Name != "In Progress" || states[1].Group != "indeterminate" {
		t.Fatalf("states = %+v", states)
	}
}

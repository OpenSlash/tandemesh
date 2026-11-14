package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestAssign(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/repos/acme/api/issues/7" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		var body struct {
			Assignees []string `json:"assignees"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if len(body.Assignees) != 1 || body.Assignees[0] != "alice" {
			t.Errorf("assignees = %+v", body.Assignees)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client(), BaseURL: srv.URL}
	err := a.Assign(t.Context(), domain.WorkRef{Instance: "acme/api", ID: "7"}, integration.Credential{AccessToken: "tok"}, "alice")
	if err != nil {
		t.Fatal(err)
	}
}

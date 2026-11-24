package linear

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
		var body struct {
			Query     string            `json:"query"`
			Variables map[string]string `json:"variables"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Variables["assigneeId"] != "user_1" {
			t.Errorf("variables = %+v", body.Variables)
		}
		_, _ = w.Write([]byte(`{"data":{"issueUpdate":{"success":true}}}`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client(), BaseURL: srv.URL}
	err := a.Assign(t.Context(), domain.WorkRef{ID: "issue_1"}, integration.Credential{AccessToken: "tok"}, "user_1")
	if err != nil {
		t.Fatal(err)
	}
}

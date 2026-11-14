package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestUpdateFields(t *testing.T) {
	var updated bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/rest/api/2/priority":
			_, _ = w.Write([]byte(`[{"id":"2","name":"High"}]`))
		case r.URL.Path == "/rest/api/2/project/ENG/components":
			_, _ = w.Write([]byte(`[{"id":"c1","name":"API"},{"id":"c2","name":"Web"}]`))
		case r.URL.Path == "/rest/api/2/project/ENG/versions":
			_, _ = w.Write([]byte(`[{"id":"v1","name":"1.2"}]`))
		case r.Method == http.MethodPut && r.URL.Path == "/rest/api/2/issue/ENG-1":
			var body struct {
				Fields struct {
					Priority struct {
						ID string `json:"id"`
					} `json:"priority"`
					Components []struct {
						ID string `json:"id"`
					} `json:"components"`
				} `json:"fields"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.Fields.Priority.ID != "2" {
				t.Errorf("priority = %+v", body.Fields.Priority)
			}
			if len(body.Fields.Components) != 2 {
				t.Errorf("components = %+v", body.Fields.Components)
			}
			updated = true
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client()}
	err := a.UpdateFields(t.Context(), domain.WorkRef{Instance: srv.URL, ID: "ENG-1"}, integration.Credential{AccessToken: "tok"},
		integration.FieldUpdate{Priority: "High", Components: []string{"API", "Web"}, FixVersions: []string{"1.2"}})
	if err != nil {
		t.Fatal(err)
	}
	if !updated {
		t.Fatal("issue was not updated")
	}
}

func TestUpdateCustomFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
		}
		var body struct {
			Fields map[string]any `json:"fields"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Fields["customfield_10010"] != float64(5) {
			t.Errorf("fields = %+v", body.Fields)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client()}
	err := a.UpdateFields(t.Context(), domain.WorkRef{Instance: srv.URL, ID: "ENG-1"}, integration.Credential{AccessToken: "tok"},
		integration.FieldUpdate{CustomFields: map[string]string{"customfield_10010": "5"}})
	if err != nil {
		t.Fatal(err)
	}
}

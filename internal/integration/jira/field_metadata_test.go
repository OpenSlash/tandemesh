package jira

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestListFieldMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/2/issue/ENG-1/editmeta" {
			t.Errorf("path = %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"fields":{"customfield_10010":{"name":"Story Points","required":false,"schema":{"type":"number"}},"priority":{"name":"Priority","required":false,"schema":{"type":"priority"},"allowedValues":[{"name":"High","value":"2"}]}}}`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client()}
	items, err := a.ListFieldMetadata(t.Context(), integration.Credential{AccessToken: "tok"}, srv.URL, "ENG-1")
	if err != nil || len(items) != 2 {
		t.Fatalf("items = %+v, err = %v", items, err)
	}
}

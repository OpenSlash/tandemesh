package jira

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/integration"
)

func TestSearchUsers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/2/user/search" || r.URL.Query().Get("query") != "ali" {
			t.Errorf("request = %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`[{"accountId":"acc-1","displayName":"Alice","emailAddress":"alice@example.com"}]`))
	}))
	defer srv.Close()
	a := &Adapter{HTTPClient: srv.Client()}
	users, err := a.SearchUsers(t.Context(), integration.Credential{AccessToken: "tok", BaseURL: srv.URL}, "ali", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].ID != "acc-1" || users[0].Email != "alice@example.com" {
		t.Fatalf("users = %+v", users)
	}
}

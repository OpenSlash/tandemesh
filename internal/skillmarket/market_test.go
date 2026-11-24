package skillmarket

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchRemoteCatalog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]Entry{{
			ID: "remote", Name: "remote-skill", Description: "from remote",
			Version: "1.0.0", Author: "test", Tags: []string{"remote"}, Body: "# Remote",
		}})
	}))
	defer server.Close()

	entries, err := Fetch(context.Background(), server.URL, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "remote-skill" || entries[0].Source != "url" {
		t.Fatalf("entries = %+v", entries)
	}
}

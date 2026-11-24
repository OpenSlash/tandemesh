package api

import (
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestSearch(t *testing.T) {
	ts := newTestServer(t)

	postJSON[domain.Task](t, ts.URL+"/api/v1/work/import", map[string]any{
		"provider": "github", "instance": "acme/api", "type": "issue", "id": "101",
		"title": "searchme deploy fix",
	})
	postJSON[domain.Group](t, ts.URL+"/api/v1/groups", map[string]string{
		"workspaceId": "ws_default", "name": "searchme crew",
	})

	result := getJSON[struct {
		Tasks  []domain.Task  `json:"tasks"`
		Groups []domain.Group `json:"groups"`
	}](t, ts.URL+"/api/v1/search?q=searchme&workspace=ws_default")
	if len(result.Tasks) != 1 || len(result.Groups) != 1 {
		t.Fatalf("results = %+v", result)
	}
	if result.Tasks[0].WorkID != "101" || result.Groups[0].Name != "searchme crew" {
		t.Fatalf("unexpected results = %+v", result)
	}

	empty := getJSON[struct {
		Tasks  []domain.Task  `json:"tasks"`
		Groups []domain.Group `json:"groups"`
	}](t, ts.URL+"/api/v1/search?q=nomatch&workspace=ws_default")
	if len(empty.Tasks) != 0 || len(empty.Groups) != 0 {
		t.Fatalf("expected no results, got %+v", empty)
	}
}

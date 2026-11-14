package api

import (
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestActivityPagination(t *testing.T) {
	ts := newTestServer(t)
	task := postJSON[domain.Task](t, ts.URL+"/api/v1/work/import", map[string]any{
		"provider": "github", "instance": "acme/api", "type": "issue", "id": "201",
		"title": "activity paging",
	})
	for _, body := range []string{"note one", "note two", "note three"} {
		postJSON[domain.TaskNote](t, ts.URL+"/api/v1/tasks/"+task.ID+"/notes", map[string]string{
			"body": body, "author": "alice",
		})
	}

	latest := getJSON[struct {
		Items   []taskActivityItem `json:"items"`
		Total   int                `json:"total"`
		Offset  int                `json:"offset"`
		HasMore bool               `json:"hasMore"`
	}](t, ts.URL+"/api/v1/tasks/"+task.ID+"/activity?limit=2")
	if latest.Total < 4 || len(latest.Items) != 2 || !latest.HasMore {
		t.Fatalf("latest page = %+v", latest)
	}

	oldest := getJSON[struct {
		Items   []taskActivityItem `json:"items"`
		HasMore bool               `json:"hasMore"`
	}](t, ts.URL+"/api/v1/tasks/"+task.ID+"/activity?limit=2&offset=0")
	if len(oldest.Items) != 2 || oldest.HasMore {
		t.Fatalf("oldest page = %+v", oldest)
	}
}

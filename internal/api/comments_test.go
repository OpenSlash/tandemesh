package api

import (
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestTaskComments(t *testing.T) {
	ts := newTestServer(t)
	task := postJSON[domain.Task](t, ts.URL+"/api/v1/work/import", map[string]any{
		"provider": "github", "instance": "acme/api", "type": "issue", "id": "301",
		"title": "comment task",
	})

	comment := postJSON[domain.TaskComment](t, ts.URL+"/api/v1/tasks/"+task.ID+"/comments", map[string]string{
		"body": "looks good", "author": "alice",
	})
	if comment.ID == "" || comment.Body != "looks good" {
		t.Fatalf("comment = %+v", comment)
	}

	list := getJSON[struct {
		Items []domain.TaskComment `json:"items"`
	}](t, ts.URL+"/api/v1/tasks/"+task.ID+"/comments")
	if len(list.Items) != 1 || list.Items[0].Author != "alice" {
		t.Fatalf("comments = %+v", list.Items)
	}

	tasks := getJSON[struct {
		Items []domain.Task `json:"items"`
	}](t, ts.URL+"/api/v1/tasks?workspace=ws_default")
	if len(tasks.Items) != 1 || tasks.Items[0].CommentCount != 1 {
		t.Fatalf("task list = %+v", tasks.Items)
	}
	if tasks.Items[0].LastComment != "looks good" || tasks.Items[0].LastCommentBy != "alice" {
		t.Fatalf("last comment = %+v", tasks.Items[0])
	}
}

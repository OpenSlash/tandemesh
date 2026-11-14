package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestCommentDeliveriesAPI(t *testing.T) {
	ts, st := newTestServerWithStore(t)
	task := mustTask(t, st, "1", "Comment ledger")
	if err := st.UpsertCommentDelivery(t.Context(), &domain.CommentDelivery{
		ID: "cd_1", WorkspaceID: task.WorkspaceID, TaskID: task.ID,
		CommentID: "pc_1", Provider: "github", Status: "sent", ExternalID: "https://example.com/1", Attempts: 1,
	}); err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(ts.URL + "/api/v1/tasks/" + task.ID + "/comment-deliveries")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(data), `"status":"sent"`) {
		t.Fatalf("status=%d body=%s", resp.StatusCode, data)
	}
}

func TestCommentFailureStatus(t *testing.T) {
	if got := commentFailureStatus(context.DeadlineExceeded); got != "uncertain" {
		t.Fatalf("deadline = %s", got)
	}
	if got := commentFailureStatus(errors.New("connection reset by peer")); got != "uncertain" {
		t.Fatalf("reset = %s", got)
	}
	if got := commentFailureStatus(errors.New("401 unauthorized")); got != "failed" {
		t.Fatalf("auth = %s", got)
	}
}

package api

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func newTestServerWithStore(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(t.Context()); err != nil {
		t.Fatal(err)
	}
	srv := New(config.Config{User: "alice"}, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)
	return ts, st
}

func mustTask(t *testing.T, st *store.Store, id, title string) *domain.Task {
	t.Helper()
	task, err := st.EnsureTaskIn(t.Context(), "ws_default", domain.WorkRef{
		Provider: "test", Instance: "default", Type: "issue", ID: id,
	}, domain.Snapshot{Title: title})
	if err != nil || task == nil {
		t.Fatalf("ensure task %s: %v", id, err)
	}
	return task
}

func getAt[T any](t *testing.T, url string) T {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out T
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestOrchestrationDispatchDAG(t *testing.T) {
	ts, st := newTestServerWithStore(t)
	taskA := mustTask(t, st, "a", "Task A")
	taskB := mustTask(t, st, "b", "Task B")

	if err := st.AddTaskDependency(t.Context(), taskB.ID, taskA.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.EvaluateTaskReadiness(t.Context(), taskB.ID); err != nil {
		t.Fatal(err)
	}
	b, _ := st.GetTask(t.Context(), taskB.ID)
	if b.ExecutionState != domain.TaskExecutionPending {
		t.Fatalf("task B execution state = %q, want pending", b.ExecutionState)
	}

	run := postJSON[domain.Run](t, ts.URL+"/api/v1/tasks/"+taskA.ID+"/runs", map[string]any{"agentId": "claude", "repo": "repo"})
	if run.ID == "" {
		t.Fatal("run not created")
	}
	dispatches := getAt[struct {
		Dispatches []domain.Dispatch `json:"dispatches"`
	}](t, ts.URL+"/api/v1/tasks/"+taskA.ID+"/dispatches")
	if len(dispatches.Dispatches) != 1 {
		t.Fatalf("dispatches = %+v", dispatches)
	}
	d := dispatches.Dispatches[0]
	if d.TaskID != taskA.ID || d.RunID != run.ID || d.Status != domain.DispatchPending {
		t.Fatalf("dispatch = %+v", d)
	}

	completed := postJSON[struct {
		Task     domain.Task `json:"task"`
		Promoted []string    `json:"promoted"`
	}](t, ts.URL+"/api/v1/dispatches/"+d.ID+"/complete", map[string]any{
		"taskId": taskA.ID, "outcome": "succeeded", "result": "done", "filesModified": []string{"a.go"},
	})
	if completed.Task.Status != domain.TaskDone || completed.Task.ExecutionState != domain.TaskExecutionCompleted {
		t.Fatalf("completed task = %+v", completed.Task)
	}
	if len(completed.Promoted) != 1 || completed.Promoted[0] != taskB.ID {
		t.Fatalf("promoted = %+v", completed.Promoted)
	}
	b, _ = st.GetTask(t.Context(), taskB.ID)
	if b.ExecutionState != domain.TaskExecutionReady {
		t.Fatalf("task B after promotion = %q", b.ExecutionState)
	}
}

func TestOrchestrationMessagesAndAnnotations(t *testing.T) {
	ts, st := newTestServerWithStore(t)
	task := mustTask(t, st, "m", "Task M")
	run := postJSON[domain.Run](t, ts.URL+"/api/v1/tasks/"+task.ID+"/runs", map[string]any{"agentId": "claude"})

	sent := postJSON[struct {
		Messages []domain.AgentMessage `json:"messages"`
	}](t, ts.URL+"/api/v1/tasks/"+task.ID+"/messages", map[string]any{
		"to": run.ID, "type": "status", "subject": "ping", "body": "hello",
	})
	if len(sent.Messages) != 1 {
		t.Fatalf("sent = %+v", sent)
	}
	checked := postJSON[struct {
		Messages []domain.AgentMessage `json:"messages"`
	}](t, ts.URL+"/api/v1/tasks/"+task.ID+"/messages/check?runId="+run.ID, map[string]any{})
	if len(checked.Messages) != 1 || checked.Messages[0].Subject != "ping" {
		t.Fatalf("checked = %+v", checked)
	}

	ann := postJSON[domain.DiffAnnotation](t, ts.URL+"/api/v1/tasks/"+task.ID+"/annotations", map[string]any{
		"runId": run.ID, "filePath": "src/a.ts", "line": 12, "body": "rename this",
	})
	if ann.ID == "" {
		t.Fatalf("annotation = %+v", ann)
	}
	batch := postJSON[struct {
		BatchID string `json:"batchId"`
		Count   int    `json:"count"`
		Prompt  string `json:"prompt"`
	}](t, ts.URL+"/api/v1/tasks/"+task.ID+"/annotations/send", map[string]any{"runId": run.ID})
	if batch.Count != 1 || batch.BatchID == "" || batch.Prompt == "" {
		t.Fatalf("batch = %+v", batch)
	}
}

func TestWorktreeLifecycle(t *testing.T) {
	ts, st := newTestServerWithStore(t)
	task := mustTask(t, st, "wt", "Task W")
	run := postJSON[domain.Run](t, ts.URL+"/api/v1/tasks/"+task.ID+"/runs", map[string]any{"agentId": "claude", "repo": "repo"})
	if run.ID == "" {
		t.Fatal("run not created")
	}
	wt, err := st.GetWorktreeByRun(t.Context(), run.ID)
	if err != nil || wt == nil {
		t.Fatalf("worktree intent: %+v err=%v", wt, err)
	}
	if wt.Status != "creating" || wt.Branch == "" {
		t.Fatalf("worktree intent = %+v", wt)
	}
	if err := st.MarkWorktreeReady(t.Context(), run.ID, "/tmp/ws", "orca/abc"); err != nil {
		t.Fatal(err)
	}
	got := getAt[domain.Worktree](t, ts.URL+"/api/v1/runs/"+run.ID+"/worktree")
	if got.Status != "ready" || got.Path != "/tmp/ws" || got.Branch != "orca/abc" {
		t.Fatalf("ready worktree = %+v", got)
	}
	updated := postJSON[domain.Worktree](t, ts.URL+"/api/v1/worktrees/"+got.ID+"/checkpoint", map[string]any{
		"comment": "halfway", "workspaceStatus": "in-progress",
	})
	if updated.CheckpointComment != "halfway" || updated.WorkspaceStatus != "in-progress" {
		t.Fatalf("checkpoint = %+v", updated)
	}
	list := getAt[struct {
		Worktrees []domain.Worktree `json:"worktrees"`
	}](t, ts.URL+"/api/v1/worktrees?taskId="+task.ID)
	if len(list.Worktrees) != 1 {
		t.Fatalf("list = %+v", list)
	}
	archived := postJSON[domain.Worktree](t, ts.URL+"/api/v1/worktrees/"+got.ID+"/archive", map[string]any{})
	if archived.Status != "archived" || archived.ArchivedAt == nil {
		t.Fatalf("archive = %+v", archived)
	}
}
func TestAgentSession(t *testing.T) {
	ts, st := newTestServerWithStore(t)
	task := mustTask(t, st, "ses", "Task S")
	sess := postJSON[domain.AgentSession](t, ts.URL+"/api/v1/sessions", map[string]any{
		"taskId": task.ID, "agentId": "claude", "mode": "chat", "title": "Refactor chat",
	})
	if sess.ID == "" || sess.Status != "active" || sess.TaskID != task.ID {
		t.Fatalf("session = %+v", sess)
	}
	run := postJSON[domain.Run](t, ts.URL+"/api/v1/sessions/"+sess.ID+"/runs", map[string]any{"prompt": "hello agent"})
	if run.ID == "" || run.SessionID != sess.ID || run.TaskID != task.ID || run.Prompt != "hello agent" {
		t.Fatalf("session run = %+v", run)
	}
	runs := getAt[struct {
		Items []domain.Run `json:"items"`
	}](t, ts.URL+"/api/v1/sessions/"+sess.ID+"/runs")
	if len(runs.Items) != 1 || runs.Items[0].ID != run.ID {
		t.Fatalf("runs = %+v", runs)
	}
	msgs := getAt[struct {
		Messages []sessionMessage `json:"messages"`
	}](t, ts.URL+"/api/v1/sessions/"+sess.ID+"/messages")
	if len(msgs.Messages) == 0 || msgs.Messages[0].Text != "hello agent" {
		t.Fatalf("messages = %+v", msgs)
	}
	if err := st.RecordSessionRun(t.Context(), sess.ID, "backend-sess-1", run.ID); err != nil {
		t.Fatal(err)
	}
	got := getAt[struct {
		Session  domain.AgentSession `json:"session"`
		RunCount int                 `json:"runCount"`
	}](t, ts.URL+"/api/v1/sessions/"+sess.ID)
	if got.Session.BackendSessionID != "backend-sess-1" || got.RunCount != 1 {
		t.Fatalf("session detail = %+v", got)
	}
	closed := postJSON[domain.AgentSession](t, ts.URL+"/api/v1/sessions/"+sess.ID+"/close", map[string]any{})
	if closed.Status != "closed" || closed.ClosedAt == nil {
		t.Fatalf("closed = %+v", closed)
	}
}
func TestRunDiffAndAnnotation(t *testing.T) {
	ts, st := newTestServerWithStore(t)
	task := mustTask(t, st, "diff", "Task Diff")
	run := postJSON[domain.Run](t, ts.URL+"/api/v1/tasks/"+task.ID+"/runs", map[string]any{"agentId": "claude"})
	diff := "diff --git a/a.go b/a.go\n--- a/a.go\n+++ b/a.go\n@@ -1,2 +1,2 @@\n-old\n+new\n context\n"
	if _, err := st.AppendEvent(t.Context(), run.ID, "workspace.diff", map[string]any{"diff": diff, "truncated": false}); err != nil {
		t.Fatal(err)
	}
	got := getAt[struct {
		Diff      string `json:"diff"`
		Truncated bool   `json:"truncated"`
	}](t, ts.URL+"/api/v1/runs/"+run.ID+"/diff")
	if got.Diff != diff || got.Truncated {
		t.Fatalf("diff = %+v", got)
	}
	ann := postJSON[domain.DiffAnnotation](t, ts.URL+"/api/v1/tasks/"+task.ID+"/annotations", map[string]any{
		"runId": run.ID, "filePath": "a.go", "line": 1, "side": "new", "body": "rename",
	})
	if ann.ID == "" || ann.Status != "open" {
		t.Fatalf("annotation = %+v", ann)
	}
}

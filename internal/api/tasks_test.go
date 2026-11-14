package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenSlash/tandemesh/internal/config"
	"github.com/OpenSlash/tandemesh/internal/domain"
	"github.com/OpenSlash/tandemesh/internal/hub"
	"github.com/OpenSlash/tandemesh/internal/store"
)

func TestTaskProjectionAndRun(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "tasks.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := t.Context()
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	task, err := st.EnsureTask(ctx, domain.WorkRef{
		Provider: "github", Instance: "acme/api", Type: "issue", ID: "7",
		URL: "https://github.com/acme/api/issues/7", Revision: "r1",
	}, domain.Snapshot{Title: "flaky test", Body: "details", Labels: []string{"agent:fix"}, Author: "alice"})
	if err != nil || task == nil {
		t.Fatalf("ensure: %+v %v", task, err)
	}
	if task.ID == "" || task.Title != "flaky test" {
		t.Fatalf("task = %+v", task)
	}

	srv := New(config.Config{}, st, nil, hub.New(), testRegistry(), nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Router())
	t.Cleanup(ts.Close)

	noteBody, _ := json.Marshal(map[string]string{"author": "alice", "body": "Use the staging database."})
	noteResp, err := http.Post(ts.URL+"/api/v1/tasks/"+task.ID+"/notes", "application/json", bytes.NewReader(noteBody))
	if err != nil {
		t.Fatal(err)
	}
	noteResp.Body.Close()
	if noteResp.StatusCode != http.StatusCreated {
		t.Fatalf("note status = %d", noteResp.StatusCode)
	}

	body, _ := json.Marshal(map[string]string{"agentId": "codex", "repo": "acme/api", "role": "reviewer"})
	resp, err := http.Post(ts.URL+"/api/v1/tasks/"+task.ID+"/runs", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d body=%s", resp.StatusCode, raw)
	}
	var run domain.Run
	if err := json.Unmarshal(raw, &run); err != nil {
		t.Fatal(err)
	}
	if run.TaskID != task.ID || run.AgentID != "codex" || run.Role != "reviewer" {
		t.Fatalf("run = %+v", run)
	}
	if !strings.Contains(run.Prompt, "Use the staging database.") {
		t.Fatalf("notes missing from prompt: %q", run.Prompt)
	}

	getResp, err := http.Get(ts.URL + "/api/v1/tasks/" + task.ID)
	if err != nil {
		t.Fatal(err)
	}
	getRaw, _ := io.ReadAll(getResp.Body)
	getResp.Body.Close()
	var detail struct {
		Task domain.Task  `json:"task"`
		Runs []domain.Run `json:"runs"`
	}
	if err := json.Unmarshal(getRaw, &detail); err != nil {
		t.Fatal(err)
	}
	if len(detail.Runs) != 1 || detail.Task.Title != "flaky test" {
		t.Fatalf("detail = %+v", detail)
	}
	queuedTask, _ := st.GetTask(ctx, task.ID)
	if queuedTask.ActiveRuns != 1 || queuedTask.ReadyToClose {
		t.Fatalf("queued task = %+v", queuedTask)
	}
	if ok, err := st.UpdateStatus(ctx, run.ID, domain.RunQueued, domain.RunSucceeded, "", ""); err != nil || !ok {
		t.Fatalf("finish run: %v %v", ok, err)
	}
	readyTask, _ := st.GetTask(ctx, task.ID)
	if !readyTask.ReadyToClose || readyTask.Succeeded != 1 || readyTask.ActiveRuns != 0 {
		t.Fatalf("ready task = %+v", readyTask)
	}

	team, _ := json.Marshal(map[string]any{
		"repo": "acme/api",
		"roles": []map[string]string{
			{"role": "lead", "agentId": "claude"},
			{"role": "reviewer", "agentId": "codex"},
		},
	})
	teamResp, err := http.Post(ts.URL+"/api/v1/tasks/"+task.ID+"/team", "application/json", bytes.NewReader(team))
	if err != nil {
		t.Fatal(err)
	}
	teamRaw, _ := io.ReadAll(teamResp.Body)
	teamResp.Body.Close()
	if teamResp.StatusCode != http.StatusCreated {
		t.Fatalf("team status = %d body=%s", teamResp.StatusCode, teamRaw)
	}
	var crew struct {
		Items []domain.Run `json:"items"`
	}
	if err := json.Unmarshal(teamRaw, &crew); err != nil {
		t.Fatal(err)
	}
	if len(crew.Items) != 2 || crew.Items[0].Role != "lead" || crew.Items[1].Role != "reviewer" {
		t.Fatalf("crew = %+v", crew.Items)
	}

	filterResp, err := http.Get(ts.URL + "/api/v1/tasks?agent=codex&label=agent:fix")
	if err != nil {
		t.Fatal(err)
	}
	filterRaw, _ := io.ReadAll(filterResp.Body)
	filterResp.Body.Close()
	var filtered struct {
		Items []domain.Task `json:"items"`
	}
	if err := json.Unmarshal(filterRaw, &filtered); err != nil {
		t.Fatal(err)
	}
	if len(filtered.Items) != 1 || filtered.Items[0].ID != task.ID {
		t.Fatalf("filtered = %+v", filtered.Items)
	}

	noteCmd, _ := json.Marshal(map[string]string{"text": "/orca note verify on staging"})
	noteCmdResp, err := http.Post(ts.URL+"/api/v1/tasks/"+task.ID+"/command", "application/json", bytes.NewReader(noteCmd))
	if err != nil {
		t.Fatal(err)
	}
	noteCmdRaw, _ := io.ReadAll(noteCmdResp.Body)
	noteCmdResp.Body.Close()
	if noteCmdResp.StatusCode != http.StatusCreated || !strings.Contains(string(noteCmdRaw), `"action":"note"`) {
		t.Fatalf("note command = %d %s", noteCmdResp.StatusCode, noteCmdRaw)
	}
	allNotes, _ := st.ListTaskNotes(ctx, task.ID)
	if len(allNotes) != 2 {
		t.Fatalf("notes = %d", len(allNotes))
	}

	runCmd, _ := json.Marshal(map[string]string{"text": "/orca run codex", "repo": "acme/api"})
	runCmdResp, err := http.Post(ts.URL+"/api/v1/tasks/"+task.ID+"/command", "application/json", bytes.NewReader(runCmd))
	if err != nil {
		t.Fatal(err)
	}
	runCmdRaw, _ := io.ReadAll(runCmdResp.Body)
	runCmdResp.Body.Close()
	var commandRun struct {
		Action string     `json:"action"`
		Run    domain.Run `json:"run"`
	}
	if err := json.Unmarshal(runCmdRaw, &commandRun); err != nil || commandRun.Action != "run" || commandRun.Run.Role != "implementer" {
		t.Fatalf("run command = %s err=%v", runCmdRaw, err)
	}

	actResp, err := http.Get(ts.URL + "/api/v1/tasks/" + task.ID + "/activity")
	if err != nil {
		t.Fatal(err)
	}
	actRaw, _ := io.ReadAll(actResp.Body)
	actResp.Body.Close()
	var activity struct {
		Items []taskActivityItem `json:"items"`
	}
	if err := json.Unmarshal(actRaw, &activity); err != nil {
		t.Fatal(err)
	}
	if len(activity.Items) == 0 {
		t.Fatalf("activity empty: %s", actRaw)
	}

	patch, _ := json.Marshal(map[string]string{"status": "done"})
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/api/v1/tasks/"+task.ID, bytes.NewReader(patch))
	req.Header.Set("Content-Type", "application/json")
	patchResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	patchResp.Body.Close()
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d", patchResp.StatusCode)
	}
	updated, _ := st.GetTask(ctx, task.ID)
	if updated.Status != domain.TaskDone {
		t.Fatalf("status = %q", updated.Status)
	}

	activity2 := getJSON[struct {
		Items []taskActivityItem `json:"items"`
	}](t, ts.URL+"/api/v1/tasks/"+task.ID+"/activity")
	statusEvent := false
	for _, item := range activity2.Items {
		if item.Type == "task.status" && item.Actor == "human" {
			statusEvent = true
		}
	}
	if !statusEvent {
		t.Fatalf("task.status event missing: %+v", activity2.Items)
	}
}

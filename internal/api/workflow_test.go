package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/OpenSlash/tandemesh/internal/domain"
)

func TestWorkflowScriptGate(t *testing.T) {
	ts := newTestServer(t)
	wf := postJSON[domain.Workflow](t, ts.URL+"/api/v1/workflows", map[string]any{
		"workspaceId": "ws_default",
		"name":        "test sop",
		"nodes": []map[string]any{
			{"id": "one", "kind": "script", "name": "One", "command": "echo ONE", "next": "gate"},
			{"id": "gate", "kind": "gate", "name": "Gate", "next": "two"},
			{"id": "two", "kind": "script", "name": "Two", "command": "echo TWO"},
		},
	})
	run := postJSON[domain.WorkflowRun](t, ts.URL+"/api/v1/workflows/"+wf.ID+"/runs", map[string]any{})
	waitWorkflow(t, ts.URL+"/api/v1/workflow-runs/"+run.ID, domain.WorkflowWaiting, "gate")

	postJSON[domain.WorkflowRun](t, ts.URL+"/api/v1/workflow-runs/"+run.ID+"/gate", map[string]any{
		"nodeId": "gate", "action": "approve", "note": "ok",
	})
	final := waitWorkflow(t, ts.URL+"/api/v1/workflow-runs/"+run.ID, domain.WorkflowSucceeded, "")
	if len(final.Nodes) != 3 || strings.TrimSpace(final.Nodes[0].Output) != "ONE" || strings.TrimSpace(final.Nodes[2].Output) != "TWO" {
		t.Fatalf("nodes = %+v", final.Nodes)
	}
}

func waitWorkflow(t *testing.T, url, status, current string) domain.WorkflowRun {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		run := getJSON[domain.WorkflowRun](t, url)
		if run.Status == status && (current == "" || run.CurrentNode == current) {
			return run
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("workflow did not reach %s/%s", status, current)
	return domain.WorkflowRun{}
}

func TestWorkflowSubflow(t *testing.T) {
	ts := newTestServer(t)
	child := postJSON[domain.Workflow](t, ts.URL+"/api/v1/workflows", map[string]any{
		"workspaceId": "ws_default", "name": "child",
		"nodes": []map[string]any{{"id": "c", "kind": "script", "name": "Child", "command": "echo CHILD"}},
	})
	parent := postJSON[domain.Workflow](t, ts.URL+"/api/v1/workflows", map[string]any{
		"workspaceId": "ws_default", "name": "parent",
		"nodes": []map[string]any{
			{"id": "sub", "kind": "workflow", "name": "Sub", "workflowId": child.ID, "next": "after"},
			{"id": "after", "kind": "script", "name": "After", "command": "echo AFTER"},
		},
	})
	run := postJSON[domain.WorkflowRun](t, ts.URL+"/api/v1/workflows/"+parent.ID+"/runs", map[string]any{})
	final := waitWorkflow(t, ts.URL+"/api/v1/workflow-runs/"+run.ID, domain.WorkflowSucceeded, "")
	if len(final.Nodes) != 2 || final.Nodes[0].Status != domain.WorkflowNodeSucceeded || strings.TrimSpace(final.Nodes[1].Output) != "AFTER" {
		t.Fatalf("subflow nodes = %+v", final.Nodes)
	}
}

func TestWorkflowOnFail(t *testing.T) {
	ts := newTestServer(t)
	wf := postJSON[domain.Workflow](t, ts.URL+"/api/v1/workflows", map[string]any{
		"workspaceId": "ws_default", "name": "fallback",
		"nodes": []map[string]any{
			{"id": "bad", "kind": "script", "name": "Bad", "command": "tandemesh-nonexistent-command", "retries": 1, "onFail": "recover"},
			{"id": "recover", "kind": "script", "name": "Recover", "command": "echo RECOVERED"},
		},
	})
	run := postJSON[domain.WorkflowRun](t, ts.URL+"/api/v1/workflows/"+wf.ID+"/runs", map[string]any{})
	final := waitWorkflow(t, ts.URL+"/api/v1/workflow-runs/"+run.ID, domain.WorkflowSucceeded, "")
	if len(final.Nodes) != 2 || final.Nodes[0].Status != domain.WorkflowNodeFailed || strings.TrimSpace(final.Nodes[1].Output) != "RECOVERED" {
		t.Fatalf("fallback nodes = %+v", final.Nodes)
	}
}

func TestWorkflowArtifacts(t *testing.T) {
	ts := newTestServer(t)
	wf := postJSON[domain.Workflow](t, ts.URL+"/api/v1/workflows", map[string]any{
		"workspaceId": "ws_default", "name": "artifacts",
		"nodes": []map[string]any{{"id": "report", "kind": "script", "name": "Report", "command": artifactCommand()}},
	})
	run := postJSON[domain.WorkflowRun](t, ts.URL+"/api/v1/workflows/"+wf.ID+"/runs", map[string]any{})
	final := waitWorkflow(t, ts.URL+"/api/v1/workflow-runs/"+run.ID, domain.WorkflowSucceeded, "")
	if len(final.Nodes) != 1 || len(final.Nodes[0].Artifacts) != 1 {
		t.Fatalf("artifacts = %+v", final.Nodes)
	}
	if final.Nodes[0].Artifacts[0].Name != "report.txt" || strings.TrimSpace(final.Nodes[0].Artifacts[0].Content) != "hello" {
		t.Fatalf("artifact = %+v", final.Nodes[0].Artifacts[0])
	}
	resp, err := http.Get(ts.URL + "/api/v1/workflow-runs/" + run.ID + "/artifacts/report/report.txt")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("download: %v %v", err, resp)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if strings.TrimSpace(string(raw)) != "hello" {
		t.Fatalf("download body = %q", string(raw))
	}
}
func artifactCommand() string {
	if runtime.GOOS == "windows" {
		return `echo hello>%TANDEMESH_ARTIFACT_DIR%\report.txt`
	}
	return `echo hello>$TANDEMESH_ARTIFACT_DIR/report.txt`
}

func TestWorkflowCancel(t *testing.T) {
	ts := newTestServer(t)
	wf := postJSON[domain.Workflow](t, ts.URL+"/api/v1/workflows", map[string]any{
		"workspaceId": "ws_default", "name": "control",
		"nodes": []map[string]any{{"id": "gate", "kind": "gate", "name": "Gate"}},
	})
	run := postJSON[domain.WorkflowRun](t, ts.URL+"/api/v1/workflows/"+wf.ID+"/runs", map[string]any{})
	waitWorkflow(t, ts.URL+"/api/v1/workflow-runs/"+run.ID, domain.WorkflowWaiting, "gate")
	cancelled := postJSON[domain.WorkflowRun](t, ts.URL+"/api/v1/workflow-runs/"+run.ID+"/cancel", map[string]any{})
	if cancelled.Status != domain.WorkflowCancelled {
		t.Fatalf("cancel status = %s", cancelled.Status)
	}
}

func TestWorkflowCancelCascades(t *testing.T) {
	ts := newTestServer(t)
	child := postJSON[domain.Workflow](t, ts.URL+"/api/v1/workflows", map[string]any{
		"workspaceId": "ws_default", "name": "gate-child",
		"nodes": []map[string]any{{"id": "hold", "kind": "gate", "name": "Hold"}},
	})
	parent := postJSON[domain.Workflow](t, ts.URL+"/api/v1/workflows", map[string]any{
		"workspaceId": "ws_default", "name": "parent",
		"nodes": []map[string]any{{"id": "sub", "kind": "workflow", "name": "Sub", "workflowId": child.ID}},
	})
	run := postJSON[domain.WorkflowRun](t, ts.URL+"/api/v1/workflows/"+parent.ID+"/runs", map[string]any{})
	waitWorkflow(t, ts.URL+"/api/v1/workflow-runs/"+run.ID, domain.WorkflowWaiting, "sub")
	current := getJSON[domain.WorkflowRun](t, ts.URL+"/api/v1/workflow-runs/"+run.ID)
	if len(current.Nodes) == 0 || current.Nodes[0].RunID == "" {
		t.Fatalf("child run missing: %+v", current.Nodes)
	}
	postJSON[domain.WorkflowRun](t, ts.URL+"/api/v1/workflow-runs/"+run.ID+"/cancel", map[string]any{})
	childRun := getJSON[domain.WorkflowRun](t, ts.URL+"/api/v1/workflow-runs/"+current.Nodes[0].RunID)
	if childRun.Status != domain.WorkflowCancelled {
		t.Fatalf("child status = %s", childRun.Status)
	}
}

func TestWorkflowCondition(t *testing.T) {
	ts := newTestServer(t)
	wf := postJSON[domain.Workflow](t, ts.URL+"/api/v1/workflows", map[string]any{
		"workspaceId": "ws_default", "name": "condition",
		"nodes": []map[string]any{
			{"id": "first", "kind": "script", "name": "First", "command": "echo HELLO", "next": "check"},
			{"id": "check", "kind": "condition", "name": "Check", "when": "contains:HELLO", "next": "yes", "onFail": "no"},
			{"id": "yes", "kind": "script", "name": "Yes", "command": "echo YES"},
			{"id": "no", "kind": "script", "name": "No", "command": "echo NO"},
		},
	})
	run := postJSON[domain.WorkflowRun](t, ts.URL+"/api/v1/workflows/"+wf.ID+"/runs", map[string]any{})
	final := waitWorkflow(t, ts.URL+"/api/v1/workflow-runs/"+run.ID, domain.WorkflowSucceeded, "")
	yes, no := findNodeRun(&final, "yes"), findNodeRun(&final, "no")
	if yes == nil || no == nil || yes.Status != domain.WorkflowNodeSucceeded || no.Status != domain.WorkflowNodePending {
		t.Fatalf("condition nodes = %+v", final.Nodes)
	}
	if strings.TrimSpace(yes.Output) != "YES" {
		t.Fatalf("yes output = %q", yes.Output)
	}
}

func TestConditionMatches(t *testing.T) {
	cases := []struct {
		expr  string
		input string
		want  bool
	}{
		{"contains:HELLO", "hello world", true},
		{"not_contains:ERROR", "all good", true},
		{"equals:APPROVED", " approved ", true},
		{"prefix:ERR", "error: bad", true},
		{"and:contains:a|contains:b", "a and b", true},
		{"and:contains:a|contains:z", "a and b", false},
		{"or:contains:a|contains:z", "a and b", true},
		{"not:contains:z", "a and b", true},
	}
	for _, tc := range cases {
		if got := conditionMatches(tc.expr, tc.input); got != tc.want {
			t.Fatalf("conditionMatches(%q, %q) = %v, want %v", tc.expr, tc.input, got, tc.want)
		}
	}
}

func TestWorkflowUpdateDelete(t *testing.T) {
	ts := newTestServer(t)
	wf := postJSON[domain.Workflow](t, ts.URL+"/api/v1/workflows", map[string]any{
		"workspaceId": "ws_default", "name": "editable",
		"nodes": []map[string]any{{"id": "one", "kind": "script", "name": "One", "command": "echo one"}},
	})
	body, _ := json.Marshal(map[string]any{
		"workspaceId": "ws_default", "name": "renamed",
		"nodes": []map[string]any{{"id": "two", "kind": "script", "name": "Two", "command": "echo two"}},
	})
	req, _ := http.NewRequest(http.MethodPut, ts.URL+"/api/v1/workflows/"+wf.ID, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("update: %v %v", err, resp.StatusCode)
	}
	resp.Body.Close()
	updated := getJSON[domain.Workflow](t, ts.URL+"/api/v1/workflows/"+wf.ID)
	if updated.Name != "renamed" || len(updated.Nodes) != 1 || updated.Nodes[0].ID != "two" {
		t.Fatalf("updated = %+v", updated)
	}
	req, _ = http.NewRequest(http.MethodDelete, ts.URL+"/api/v1/workflows/"+wf.ID, nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusNoContent {
		t.Fatalf("delete: %v %v", err, resp.StatusCode)
	}
	resp.Body.Close()
}

func TestWorkflowTemplates(t *testing.T) {
	ts := newTestServer(t)
	list := getJSON[struct {
		Items []workflowTemplate `json:"items"`
	}](t, ts.URL+"/api/v1/workflow-templates")
	if len(list.Items) < 3 {
		t.Fatalf("templates = %+v", list.Items)
	}
}

func TestWorkflowParallel(t *testing.T) {
	ts := newTestServer(t)
	wf := postJSON[domain.Workflow](t, ts.URL+"/api/v1/workflows", map[string]any{
		"workspaceId": "ws_default", "name": "parallel",
		"nodes": []map[string]any{
			{"id": "fan", "kind": "parallel", "name": "Fan", "commands": []string{"echo A", "echo B"}, "next": "after"},
			{"id": "after", "kind": "script", "name": "After", "command": "echo DONE"},
		},
	})
	run := postJSON[domain.WorkflowRun](t, ts.URL+"/api/v1/workflows/"+wf.ID+"/runs", map[string]any{})
	final := waitWorkflow(t, ts.URL+"/api/v1/workflow-runs/"+run.ID, domain.WorkflowSucceeded, "")
	fan := findNodeRun(&final, "fan")
	if fan == nil || !strings.Contains(fan.Output, "A") || !strings.Contains(fan.Output, "B") {
		t.Fatalf("parallel output = %+v", fan)
	}
}

func TestTaskRunCreatesSession(t *testing.T) {
	ts := newTestServer(t)
	agent := postJSON[domain.Agent](t, ts.URL+"/api/v1/agents", map[string]any{
		"workspaceId": "ws_default", "name": "session agent", "backend": "claude", "sessionMode": "off",
	})
	task := postJSON[domain.Task](t, ts.URL+"/api/v1/tasks", map[string]any{
		"workspaceId": "ws_default", "title": "session task", "body": "hello",
	})
	run := postJSON[domain.Run](t, ts.URL+"/api/v1/tasks/"+task.ID+"/runs", map[string]any{"agentId": agent.ID, "repo": "", "role": "lead"})
	if run.SessionID == "" {
		t.Fatalf("run has no session: %+v", run)
	}
	list := getJSON[struct {
		Sessions []domain.AgentSession `json:"sessions"`
	}](t, ts.URL+"/api/v1/sessions?workspaceId=ws_default&taskId="+task.ID)
	if len(list.Sessions) != 1 || list.Sessions[0].ID != run.SessionID || list.Sessions[0].TaskID != task.ID {
		t.Fatalf("sessions = %+v", list.Sessions)
	}
}

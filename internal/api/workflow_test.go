package api

import (
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

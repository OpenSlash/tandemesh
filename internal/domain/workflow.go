package domain

import "time"

const (
	WorkflowRunning   = "running"
	WorkflowWaiting   = "waiting"
	WorkflowSucceeded = "succeeded"
	WorkflowFailed    = "failed"
	WorkflowCancelled = "cancelled"

	WorkflowNodePending   = "pending"
	WorkflowNodeRunning   = "running"
	WorkflowNodeWaiting   = "waiting"
	WorkflowNodeSucceeded = "succeeded"
	WorkflowNodeFailed    = "failed"
	WorkflowNodeSkipped   = "skipped"
)

type WorkflowNode struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	AgentID string `json:"agentId,omitempty"`
	Prompt  string `json:"prompt,omitempty"`
	Command string `json:"command,omitempty"`
	Next    string `json:"next,omitempty"`
	OnFail  string `json:"onFail,omitempty"`
}

type Workflow struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspaceId"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Nodes       []WorkflowNode `json:"nodes"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

type WorkflowNodeRun struct {
	NodeID     string     `json:"nodeId"`
	Kind       string     `json:"kind"`
	Name       string     `json:"name"`
	Status     string     `json:"status"`
	Output     string     `json:"output,omitempty"`
	Error      string     `json:"error,omitempty"`
	RunID      string     `json:"runId,omitempty"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

type WorkflowRun struct {
	ID          string            `json:"id"`
	WorkflowID  string            `json:"workflowId"`
	WorkspaceID string            `json:"workspaceId"`
	TaskID      string            `json:"taskId,omitempty"`
	Status      string            `json:"status"`
	CurrentNode string            `json:"currentNode,omitempty"`
	Nodes       []WorkflowNodeRun `json:"nodes"`
	Error       string            `json:"error,omitempty"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

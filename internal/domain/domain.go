// Package domain holds the core types and invariants of Orca.
//
// Orca does not own tasks, documents or chat. It only stores execution facts:
// runs, events, machines, agents and artifacts. A WorkRef points at the
// external system that remains the source of truth.
package domain

import (
	"encoding/json"
	"strings"
	"time"
)

// RunStatus is the lifecycle state of a run.
type RunStatus string

const (
	RunQueued          RunStatus = "queued"
	RunLeased          RunStatus = "leased"
	RunPreparing       RunStatus = "preparing"
	RunRunning         RunStatus = "running"
	RunWaitingApproval RunStatus = "waiting_approval"
	RunFinalizing      RunStatus = "finalizing"
	RunSucceeded       RunStatus = "succeeded"
	RunFailed          RunStatus = "failed"
	RunCancelled       RunStatus = "cancelled"
)

// Terminal reports whether no further transition is expected.
func (s RunStatus) Terminal() bool {
	switch s {
	case RunSucceeded, RunFailed, RunCancelled:
		return true
	}
	return false
}

// Valid reports whether s is a known status.
func (s RunStatus) Valid() bool {
	switch s {
	case RunQueued, RunLeased, RunPreparing, RunRunning,
		RunWaitingApproval, RunFinalizing, RunSucceeded, RunFailed, RunCancelled:
		return true
	}
	return false
}

// WorkRef is a pointer to an external work item. It is never a copy of the
// item body: provider, instance, type and id identify it; revision guards
// write-back against concurrent edits.
type WorkRef struct {
	Provider string `json:"provider"`
	Instance string `json:"instance"`
	Type     string `json:"type"`
	ID       string `json:"id"`
	URL      string `json:"url"`
	Revision string `json:"revision"`
}

// Snapshot is an immutable, audit-only capture taken when the run is created.
type Snapshot struct {
	Title       string   `json:"title"`
	Body        string   `json:"body,omitempty"`
	BodyHash    string   `json:"bodyHash"`
	Labels      []string `json:"labels"`
	Author      string   `json:"author"`
	Assignee    string   `json:"assignee,omitempty"`
	Status      string   `json:"status,omitempty"`
	StatusGroup string   `json:"statusGroup,omitempty"`
}

// Run is a single execution instance bound to one WorkRef and one agent.
type Run struct {
	ID           string    `json:"id"`
	WorkspaceID  string    `json:"workspaceId"`
	Backend      string    `json:"backend,omitempty"`
	Work         WorkRef   `json:"work"`
	Snapshot     Snapshot  `json:"snapshot"`
	AutomationID string    `json:"automationId,omitempty"`
	AgentID      string    `json:"agentId,omitempty"`
	MachineID    string    `json:"machineId,omitempty"`
	Status       RunStatus `json:"status"`
	Priority     int       `json:"priority"`
	Attempt      int       `json:"attempt"`
	Fence        int64     `json:"fence"`

	Repo   string `json:"repo,omitempty"`
	Base   string `json:"base,omitempty"`
	Branch string `json:"branch,omitempty"`
	Prompt string `json:"prompt,omitempty"`

	VariantOf       string `json:"variantOf,omitempty"`
	VariantLabel    string `json:"variantLabel,omitempty"`
	TaskID          string `json:"taskId,omitempty"`
	Role            string `json:"role,omitempty"`
	SessionID       string `json:"sessionId,omitempty"`
	SessionRecordID string `json:"sessionRecordId,omitempty"`
	WorktreePath    string `json:"worktreePath,omitempty"`

	InputTokens      int               `json:"inputTokens"`
	OutputTokens     int               `json:"outputTokens"`
	CacheReadTokens  int               `json:"cacheReadTokens"`
	CacheWriteTokens int               `json:"cacheWriteTokens"`
	CostUSD          float64           `json:"costUsd"`
	UsageEstimated   bool              `json:"usageEstimated"`
	Model            string            `json:"model,omitempty"`
	AgentEnv         map[string]string `json:"agentEnv,omitempty"`

	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`

	CreatedAt      time.Time  `json:"createdAt"`
	StartedAt      *time.Time `json:"startedAt,omitempty"`
	EndedAt        *time.Time `json:"endedAt,omitempty"`
	LeaseExpiresAt *time.Time `json:"leaseExpiresAt,omitempty"`
}

// AuditLog records a mutating API action.
type AuditLog struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	Actor       string    `json:"actor"`
	Action      string    `json:"action"`
	Resource    string    `json:"resource"`
	Detail      string    `json:"detail,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// Document is a workspace knowledge note.
type Document struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	Tags        []string  `json:"tags,omitempty"`
	CreatedBy   string    `json:"createdBy,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Notification is an in-app message for the notification center.
type Notification struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspaceId"`
	Type        string     `json:"type"`
	Title       string     `json:"title"`
	Body        string     `json:"body,omitempty"`
	Link        string     `json:"link,omitempty"`
	ReadAt      *time.Time `json:"readAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// Machine is a registered execution host.
type Machine struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Labels []string `json:"labels"`
	Agents []string `json:"agents"`
	// Runtimes maps each backend the host can run right now to its version.
	Runtimes map[string]string `json:"runtimes,omitempty"`
	// RuntimeDetails explains every known backend, including the ones that
	// cannot run yet and why.
	RuntimeDetails []RuntimeStatus `json:"runtimeDetails,omitempty"`
	Concurrency    int             `json:"concurrency"`
	Status         string          `json:"status"`
	Version        string          `json:"version,omitempty"`
	LastSeen       time.Time       `json:"lastSeen"`
	Metrics        *MachineMetrics `json:"metrics,omitempty"`
}

// Runtime statuses reported by an execution host.
const (
	RuntimeReady          = "ready"           // the ACP launcher resolves
	RuntimeAdapterMissing = "adapter-missing" // the CLI is installed but its ACP adapter is not
	RuntimeMissing        = "missing"         // neither the CLI nor a launcher is installed
	RuntimeUnsupported    = "unsupported"     // the CLI is installed but does not speak ACP
)

// RuntimeStatus is one backend as seen by an execution host.
type RuntimeStatus struct {
	ID      string `json:"id"`
	Name    string `json:"name,omitempty"`
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
	// CLI is the resolved path of the agent CLI, when installed.
	CLI string `json:"cli,omitempty"`
	// Launcher is the ACP command the host starts, or would start.
	Launcher string `json:"launcher,omitempty"`
	// Source tells where the launcher came from: flag, custom, registry or path.
	Source string `json:"source,omitempty"`
	// RegistryID and RegistryVersion identify an ACP Registry install.
	RegistryID      string `json:"registryId,omitempty"`
	RegistryVersion string `json:"registryVersion,omitempty"`
}

// Machine job kinds. Jobs are queued by the plane and executed by the host.
const (
	MachineJobRuntimeCheck = "runtime-check"
	MachineJobACPInstall   = "acp-install"
	MachineJobACPUninstall = "acp-uninstall"
)

// MachineJob is work the plane asks one execution host to perform locally.
type MachineJob struct {
	ID          string          `json:"id"`
	MachineID   string          `json:"machineId"`
	Kind        string          `json:"kind"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	Status      string          `json:"status"`
	Result      json.RawMessage `json:"result,omitempty"`
	Error       string          `json:"error,omitempty"`
	RequestedAt time.Time       `json:"requestedAt"`
	StartedAt   *time.Time      `json:"startedAt,omitempty"`
	CompletedAt *time.Time      `json:"completedAt,omitempty"`
}

// MachineMetrics is the latest host-level resource sample.
type MachineMetrics struct {
	CPUPercent  float64   `json:"cpuPercent"`
	MemoryUsed  uint64    `json:"memoryUsedBytes"`
	MemoryTotal uint64    `json:"memoryTotalBytes"`
	DiskUsed    uint64    `json:"diskUsedBytes"`
	DiskTotal   uint64    `json:"diskTotalBytes"`
	CollectedAt time.Time `json:"collectedAt"`
}

// Event is one append-only execution fact for a run.
type Event struct {
	RunID   string         `json:"runId"`
	Seq     int            `json:"seq"`
	TS      time.Time      `json:"ts"`
	Type    string         `json:"type"`
	Payload map[string]any `json:"payload,omitempty"`
}

// Event type constants.
const (
	EventRunCreated      = "run.created"
	EventRunLeased       = "run.leased"
	EventRunStarted      = "run.started"
	EventAgentMessage    = "agent.message"
	EventToolCall        = "tool.call"
	EventToolResult      = "tool.result"
	EventRunFinished     = "run.finished"
	EventRunLeaseExpired = "run.lease_expired"
)

// Agent is a persistent agent configuration. It is not a process: runs bind to
// an agent, and the agent selects a runner backend.
type Agent struct {
	ID           string         `json:"id"`
	WorkspaceID  string         `json:"workspaceId"`
	WorkspaceIDs []string       `json:"workspaceIds,omitempty"`
	MachineID    string         `json:"machineId,omitempty"`
	Name         string         `json:"name"`
	Instructions string         `json:"instructions"`
	Backend      string         `json:"backend"`
	Role         string         `json:"role"`
	Capabilities []string       `json:"capabilities"`
	Model        string         `json:"model"`
	Skills       []string       `json:"skills"`
	Repos        []string       `json:"repos"`
	SessionMode  string         `json:"sessionMode,omitempty"`
	Tools        map[string]any `json:"tools"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
}

// Trigger kinds for event automations.
const (
	TriggerAny     = "any"
	TriggerCreated = "created"
	TriggerUpdated = "updated"
	TriggerLabel   = "label"
	TriggerAssign  = "assign"
	TriggerComment = "comment"
)

// Approval is a pending human decision for a tool call.
type Approval struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspaceId,omitempty"`
	RunID       string     `json:"runId"`
	Tool        string     `json:"tool"`
	Command     string     `json:"command"`
	Risk        string     `json:"risk"`
	Status      string     `json:"status"`
	RequestedAt time.Time  `json:"requestedAt"`
	ExpiresAt   time.Time  `json:"expiresAt"`
	DecidedAt   *time.Time `json:"decidedAt,omitempty"`
	DecidedBy   string     `json:"decidedBy,omitempty"`
	MessageRef  string     `json:"messageRef,omitempty"`
}

// Approval statuses.
const (
	ApprovalPending  = "pending"
	ApprovalApproved = "approved"
	ApprovalDenied   = "denied"
	ApprovalExpired  = "expired"
)

// Writeback is a queued result publication to an external system.
type Connection struct {
	ID              string             `json:"id"`
	WorkspaceID     string             `json:"workspaceId"`
	Provider        string             `json:"provider"`
	Name            string             `json:"name"`
	Status          string             `json:"status"` // pending|connected|needs_reauth|revoked
	AuthMethod      string             `json:"authMethod"`
	SyncDirection   string             `json:"syncDirection"`
	Instance        string             `json:"instance"`
	BaseURL         string             `json:"baseUrl"`
	AccountLogin    string             `json:"accountLogin"`
	ExecutionMode   string             `json:"executionMode"`
	RunnerMachineID string             `json:"runnerMachineId"`
	TokenExpiresAt  *time.Time         `json:"tokenExpiresAt,omitempty"`
	LastCallbackAt  *time.Time         `json:"lastCallbackAt,omitempty"`
	Enabled         bool               `json:"enabled"`
	AutomationID    string             `json:"automationId,omitempty"`
	Transition      string             `json:"transition,omitempty"`
	AgentID         string             `json:"agentId,omitempty"`
	CreatedAt       time.Time          `json:"createdAt"`
	UpdatedAt       time.Time          `json:"updatedAt"`
	Bindings        []*ResourceBinding `json:"bindings,omitempty"`
	AllBindings     []*ResourceBinding `json:"allBindings,omitempty"`
}

type ConnectionAccount struct {
	ID                string    `json:"id"`
	WorkspaceID       string    `json:"workspaceId"`
	ConnectionID      string    `json:"connectionId"`
	Provider          string    `json:"provider"`
	Kind              string    `json:"kind"` // org|user|bot
	MemberID          string    `json:"memberId,omitempty"`
	Login             string    `json:"login"`
	Email             string    `json:"email,omitempty"`
	ExternalID        string    `json:"externalId,omitempty"`
	CredentialSubject string    `json:"credentialSubject"`
	IsDefault         bool      `json:"isDefault"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type MemberIdentity struct {
	ID                string    `json:"id"`
	WorkspaceID       string    `json:"workspaceId"`
	MemberID          string    `json:"memberId"`
	Provider          string    `json:"provider"`
	ExternalID        string    `json:"externalId"`
	Login             string    `json:"login"`
	Email             string    `json:"email,omitempty"`
	CredentialSubject string    `json:"credentialSubject"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type ExternalIdentity struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	Provider    string    `json:"provider"`
	ExternalID  string    `json:"externalId"`
	Login       string    `json:"login"`
	Email       string    `json:"email,omitempty"`
	MemberID    string    `json:"memberId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ResourceBinding struct {
	ID           string            `json:"id"`
	WorkspaceID  string            `json:"workspaceId"`
	ConnectionID string            `json:"connectionId"`
	Provider     string            `json:"provider"`
	ExternalID   string            `json:"externalId"`
	Name         string            `json:"name"`
	Kind         string            `json:"kind"` // repo|project|channel|resource
	Direction    string            `json:"direction"`
	AccountID    string            `json:"accountId,omitempty"`
	Settings     map[string]string `json:"settings,omitempty"`
	Enabled      bool              `json:"enabled"`
	LastSyncedAt *time.Time        `json:"lastSyncedAt,omitempty"`
	LastError    string            `json:"lastError,omitempty"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

type AssignmentRule struct {
	ID           string    `json:"id"`
	WorkspaceID  string    `json:"workspaceId"`
	Name         string    `json:"name"`
	Provider     string    `json:"provider"`
	Instance     string    `json:"instance"`
	Label        string    `json:"label"`
	Author       string    `json:"author"`
	Assignee     string    `json:"assignee"`
	AgentID      string    `json:"agentId"`
	GroupID      string    `json:"groupId"`
	Repo         string    `json:"repo"`
	AutoDispatch bool      `json:"autoDispatch"`
	Policy       string    `json:"policy"`
	Enabled      bool      `json:"enabled"`
	Priority     int       `json:"priority"`
	CreatedAt    time.Time `json:"createdAt"`
}

// Connection statuses.
const (
	ConnectionPending     = "pending"
	ConnectionConnected   = "connected"
	ConnectionNeedsReauth = "needs_reauth"
	ConnectionRevoked     = "revoked"
)

type Writeback struct {
	ID           string     `json:"id"`
	RunID        string     `json:"runId"`
	Provider     string     `json:"provider"`
	ConnectionID string     `json:"connectionId"`
	AccountID    string     `json:"accountId,omitempty"`
	Action       string     `json:"action"` // comment|attach
	Work         WorkRef    `json:"work"`
	Markdown     string     `json:"markdown,omitempty"`
	URL          string     `json:"url,omitempty"`
	Kind         string     `json:"kind,omitempty"`
	Attempts     int        `json:"attempts"`
	NextAt       time.Time  `json:"nextAt"`
	DoneAt       *time.Time `json:"doneAt,omitempty"`
	LastError    string     `json:"lastError,omitempty"`
}

// Automation is a unified scheduled or event-triggered agent action.
type Automation struct {
	ID                 string            `json:"id"`
	WorkspaceID        string            `json:"workspaceId"`
	Name               string            `json:"name"`
	Schedule           string            `json:"schedule,omitempty"`
	Provider           string            `json:"provider,omitempty"`
	Instance           string            `json:"instance,omitempty"`
	TriggerKind        string            `json:"triggerKind,omitempty"`
	TriggerValue       string            `json:"triggerValue,omitempty"`
	Writeback          map[string]any    `json:"writeback,omitempty"`
	StateMap           map[string]string `json:"stateMap,omitempty"`
	ConnectionID       string            `json:"connectionId,omitempty"`
	Priority           int               `json:"priority,omitempty"`
	AgentID            string            `json:"agentId"`
	Repo               string            `json:"repo"`
	Base               string            `json:"base"`
	Prompt             string            `json:"prompt"`
	Enabled            bool              `json:"enabled"`
	Precheck           string            `json:"precheck,omitempty"`
	MissedGraceSeconds int               `json:"missedGraceSeconds"`
	SessionID          string            `json:"sessionId,omitempty"`
	ReuseSession       bool              `json:"reuseSession"`
	LastStatus         string            `json:"lastStatus,omitempty"`
	LastError          string            `json:"lastError,omitempty"`
	NextRun            *time.Time        `json:"nextRun,omitempty"`
	LastRun            *time.Time        `json:"lastRun,omitempty"`
	CreatedAt          time.Time         `json:"createdAt"`
}

// Task is the projection of an external work item. The source system remains
// the record; this row exists so the workspace can be task-first.
type Task struct {
	ID                 string              `json:"id"`
	WorkspaceID        string              `json:"workspaceId"`
	Provider           string              `json:"provider"`
	Instance           string              `json:"instance"`
	WorkType           string              `json:"workType"`
	WorkID             string              `json:"workId"`
	URL                string              `json:"url"`
	Revision           string              `json:"revision"`
	Title              string              `json:"title"`
	Body               string              `json:"body"`
	Labels             []string            `json:"labels"`
	Author             string              `json:"author"`
	Assignee           string              `json:"assignee"`
	AssigneeMemberID   string              `json:"assigneeMemberId,omitempty"`
	Origin             string              `json:"origin,omitempty"`
	Fields             map[string][]string `json:"fields,omitempty"`
	AssigneeAgentID    string              `json:"assigneeAgentId"`
	AssigneeGroupID    string              `json:"assigneeGroupId"`
	AutoDispatchedAt   *time.Time          `json:"autoDispatchedAt,omitempty"`
	Status             string              `json:"status"`
	PlatformState      string              `json:"platformState"`
	PlatformStateGroup string              `json:"platformStateGroup"`
	ExecutionState     string              `json:"executionState"`
	InboxState         string              `json:"inboxState"`
	InboxReadAt        *time.Time          `json:"inboxReadAt,omitempty"`
	IterationID        string              `json:"iterationId,omitempty"`
	ParentID           string              `json:"parentId,omitempty"`
	DoneAt             *time.Time          `json:"doneAt,omitempty"`
	RunCount           int                 `json:"runCount"`
	CommentCount       int                 `json:"commentCount"`
	LastComment        string              `json:"lastComment,omitempty"`
	LastCommentBy      string              `json:"lastCommentBy,omitempty"`
	ActiveRuns         int                 `json:"activeRuns"`
	ActiveAgentID      string              `json:"activeAgentId"`
	FailedRuns         int                 `json:"failedRuns"`
	Succeeded          int                 `json:"succeededRuns"`
	ReadyToClose       bool                `json:"readyToClose"`
	CreatedAt          time.Time           `json:"createdAt"`
	UpdatedAt          time.Time           `json:"updatedAt"`
	SyncedAt           time.Time           `json:"syncedAt"`
}

type ImportBatch struct {
	ID           string     `json:"id"`
	WorkspaceID  string     `json:"workspaceId"`
	Provider     string     `json:"provider"`
	ConnectionID string     `json:"connectionId"`
	Query        string     `json:"query"`
	Status       string     `json:"status"`
	Imported     int        `json:"imported"`
	NextStartAt  int        `json:"nextStartAt"`
	Limit        int        `json:"limit"`
	Failed       int        `json:"failed"`
	Error        string     `json:"error,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	FinishedAt   *time.Time `json:"finishedAt,omitempty"`
}

type TaskField struct {
	TaskID   string `json:"taskId"`
	Provider string `json:"provider"`
	Key      string `json:"key"`
	Value    string `json:"value"`
}

type TaskAttachment struct {
	ID         string    `json:"id"`
	TaskID     string    `json:"taskId"`
	Provider   string    `json:"provider"`
	ExternalID string    `json:"externalId"`
	Name       string    `json:"name"`
	URL        string    `json:"url"`
	Mime       string    `json:"mime,omitempty"`
	Size       int64     `json:"size,omitempty"`
	Author     string    `json:"author,omitempty"`
	Source     string    `json:"source,omitempty"`
	CreatedAt  time.Time `json:"createdAt"`
}

type TaskCommitLink struct {
	ID          string    `json:"id"`
	TaskID      string    `json:"taskId"`
	WorkspaceID string    `json:"workspaceId"`
	Provider    string    `json:"provider"`
	Repo        string    `json:"repo"`
	SHA         string    `json:"sha"`
	URL         string    `json:"url"`
	Message     string    `json:"message"`
	Author      string    `json:"author,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type TaskPullRequest struct {
	ID           string     `json:"id"`
	TaskID       string     `json:"taskId"`
	WorkspaceID  string     `json:"workspaceId"`
	Provider     string     `json:"provider"`
	Repo         string     `json:"repo"`
	Number       int        `json:"number"`
	URL          string     `json:"url"`
	Title        string     `json:"title"`
	State        string     `json:"state"`
	Branch       string     `json:"branch,omitempty"`
	Author       string     `json:"author,omitempty"`
	BacklinkedAt *time.Time `json:"backlinkedAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

// Task statuses.
const (
	TaskOpen       = "open"
	TaskInProgress = "in_progress"
	TaskDone       = "done"
)

// Agent roles on a task.
const (
	RoleLead        = "lead"
	RoleImplementer = "implementer"
	RoleReviewer    = "reviewer"
)

// TaskNote is a local instruction for the agents working a task. It is not
// written back to the source system.
type TaskNote struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"taskId"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

// TaskInbox is one user's triage state for a task.
type TaskInbox struct {
	TaskID    string     `json:"taskId"`
	UserID    string     `json:"userId"`
	State     string     `json:"state"`
	ReadAt    *time.Time `json:"readAt,omitempty"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// Workspace partitions the local workspace list into projects.
type Workspace struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"createdAt"`
}

// Group is a named team of agents inside a workspace.
type Group struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

// GroupMember links an agent to a group with a role.
type GroupMember struct {
	GroupID string `json:"groupId"`
	AgentID string `json:"agentId"`
	Role    string `json:"role"`
}

type CommentThread struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	TaskID      string    `json:"taskId"`
	Provider    string    `json:"provider"`
	ExternalID  string    `json:"externalId"`
	Source      string    `json:"source"`
	Resolved    bool      `json:"resolved"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CommentDelivery records one outbound comment target outcome.
type CommentDelivery struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	TaskID      string    `json:"taskId"`
	CommentID   string    `json:"commentId"`
	Provider    string    `json:"provider"`
	Status      string    `json:"status"` // pending|sent|failed|uncertain
	ExternalID  string    `json:"externalId,omitempty"`
	Error       string    `json:"error,omitempty"`
	Attempts    int       `json:"attempts"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
type ExternalComment struct {
	ID               string    `json:"id"`
	TaskID           string    `json:"taskId"`
	Provider         string    `json:"provider"`
	ExternalID       string    `json:"externalId"`
	Author           string    `json:"author"`
	Body             string    `json:"body"`
	ThreadID         string    `json:"threadId,omitempty"`
	ParentExternalID string    `json:"parentExternalId,omitempty"`
	AuthorExternalID string    `json:"authorExternalId,omitempty"`
	AuthorMemberID   string    `json:"authorMemberId,omitempty"`
	Direction        string    `json:"direction,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
}

type SyncCursor struct {
	ConnectionID string    `json:"connectionId"`
	BindingID    string    `json:"bindingId,omitempty"`
	Cursor       time.Time `json:"cursor"`
	LastSyncedAt time.Time `json:"lastSyncedAt"`
	LastFullAt   time.Time `json:"lastFullAt"`
	LastError    string    `json:"lastError"`
}

type WebhookDelivery struct {
	ID           string    `json:"id"`
	Provider     string    `json:"provider"`
	AutomationID string    `json:"automationId"`
	Delivery     string    `json:"delivery"`
	Kind         string    `json:"kind"`
	Status       string    `json:"status"`
	Reason       string    `json:"reason"`
	RunID        string    `json:"runId"`
	Body         string    `json:"body,omitempty"`
	Attempts     int       `json:"attempts"`
	CreatedAt    time.Time `json:"createdAt"`
}

// TaskEvent records a human or system action on a task projection.
type TaskEvent struct {
	ID        string         `json:"id"`
	TaskID    string         `json:"taskId"`
	Actor     string         `json:"actor"`
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload,omitempty"`
	CreatedAt time.Time      `json:"createdAt"`
}

// SavedView is a named task filter inside a workspace.
type SavedView struct {
	ID          string            `json:"id"`
	WorkspaceID string            `json:"workspaceId"`
	Name        string            `json:"name"`
	Filters     map[string]string `json:"filters"`
	CreatedAt   time.Time         `json:"createdAt"`
}

// Iteration is a sprint-like period inside a workspace.
type Iteration struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	Name        string    `json:"name"`
	StartDate   string    `json:"startDate"`
	EndDate     string    `json:"endDate"`
	Status      string    `json:"status"`
	TaskCount   int       `json:"taskCount"`
	DoneCount   int       `json:"doneCount"`
	CreatedAt   time.Time `json:"createdAt"`
}

// BurndownPoint is one day of an iteration burndown.
type BurndownPoint struct {
	Date      string `json:"date"`
	Remaining int    `json:"remaining"`
	Total     int    `json:"total"`
}

// IterationStats summarises an iteration.
type IterationStats struct {
	Total      int `json:"total"`
	Done       int `json:"done"`
	Open       int `json:"open"`
	InProgress int `json:"inProgress"`
}

// Skill is a workspace-scoped playbook injected into agent prompts.
type Skill struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Version     string    `json:"version,omitempty"`
	Source      string    `json:"source"`
	SourceURL   string    `json:"sourceUrl,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
	Enabled     bool      `json:"enabled"`
	Checksum    string    `json:"checksum,omitempty"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// TaskComment is a local discussion entry on a task.
type TaskComment struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"taskId"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

// Task execution states used by the dispatch/DAG layer.
const (
	TaskExecutionPending    = "pending"
	TaskExecutionReady      = "ready"
	TaskExecutionDispatched = "dispatched"
	TaskExecutionCompleted  = "completed"
	TaskExecutionFailed     = "failed"
	TaskExecutionBlocked    = "blocked"
)

// Dispatch statuses.
const (
	DispatchPending       = "pending"
	DispatchRunning       = "running"
	DispatchSucceeded     = "succeeded"
	DispatchFailed        = "failed"
	DispatchCancelled     = "cancelled"
	DispatchCircuitBroken = "circuit_broken"
)

// Dispatch is one execution attempt of a task. The task outlives dispatches.
type Dispatch struct {
	ID            string     `json:"id"`
	TaskID        string     `json:"taskId"`
	WorkspaceID   string     `json:"workspaceId"`
	RunID         string     `json:"runId,omitempty"`
	AgentID       string     `json:"agentId,omitempty"`
	Attempt       int        `json:"attempt"`
	Status        string     `json:"status"`
	Outcome       string     `json:"outcome,omitempty"`
	Result        string     `json:"result,omitempty"`
	FilesModified []string   `json:"filesModified,omitempty"`
	FailureCount  int        `json:"failureCount"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
	EndedAt       *time.Time `json:"endedAt,omitempty"`
}

// AgentMessage is mailbox mail between agent runs working the same task.
type AgentMessage struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspaceId"`
	TaskID      string     `json:"taskId"`
	RunID       string     `json:"runId,omitempty"`
	DispatchID  string     `json:"dispatchId,omitempty"`
	Sender      string     `json:"sender"`
	Recipient   string     `json:"recipient"`
	Type        string     `json:"type"`
	Subject     string     `json:"subject"`
	Body        string     `json:"body"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	HandledAt   *time.Time `json:"handledAt,omitempty"`
}

// DecisionGate blocks a task until a human records a resolution.
type DecisionGate struct {
	ID         string     `json:"id"`
	TaskID     string     `json:"taskId"`
	DispatchID string     `json:"dispatchId,omitempty"`
	Question   string     `json:"question"`
	Options    []string   `json:"options"`
	Resolution string     `json:"resolution,omitempty"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"createdAt"`
	ResolvedAt *time.Time `json:"resolvedAt,omitempty"`
}

// DiffAnnotation is a line-anchored review note on an agent-produced diff.
type DiffAnnotation struct {
	ID          string     `json:"id"`
	WorkspaceID string     `json:"workspaceId"`
	TaskID      string     `json:"taskId"`
	RunID       string     `json:"runId,omitempty"`
	FilePath    string     `json:"filePath"`
	Line        int        `json:"line"`
	Side        string     `json:"side"`
	Body        string     `json:"body"`
	Author      string     `json:"author"`
	Status      string     `json:"status"`
	BatchID     string     `json:"batchId,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	SentAt      *time.Time `json:"sentAt,omitempty"`
	ResolvedAt  *time.Time `json:"resolvedAt,omitempty"`
}

// Worktree is the control-plane projection of an agent's on-disk git worktree.
type Worktree struct {
	ID                string     `json:"id"`
	WorkspaceID       string     `json:"workspaceId"`
	TaskID            string     `json:"taskId"`
	RunID             string     `json:"runId"`
	Repo              string     `json:"repo"`
	Path              string     `json:"path"`
	Branch            string     `json:"branch"`
	BaseRef           string     `json:"baseRef"`
	StartRef          string     `json:"startRef"`
	Status            string     `json:"status"`
	CheckpointComment string     `json:"checkpointComment,omitempty"`
	WorkspaceStatus   string     `json:"workspaceStatus,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
	ArchivedAt        *time.Time `json:"archivedAt,omitempty"`
}

// WorktreeCleanup asks an execution machine to delete a worktree from disk.
type WorktreeCleanup struct {
	ID          string     `json:"id"`
	WorktreeID  string     `json:"worktreeId"`
	RunID       string     `json:"runId"`
	Path        string     `json:"path"`
	Repo        string     `json:"repo"`
	MachineID   string     `json:"machineId"`
	Status      string     `json:"status"`
	Attempts    int        `json:"attempts"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// AgentSession is a persistent conversation with an agent across runs.
type AgentSession struct {
	ID               string     `json:"id"`
	WorkspaceID      string     `json:"workspaceId"`
	TaskID           string     `json:"taskId,omitempty"`
	AgentID          string     `json:"agentId"`
	Backend          string     `json:"backend,omitempty"`
	Model            string     `json:"model,omitempty"`
	Reasoning        string     `json:"reasoning,omitempty"`
	Mode             string     `json:"mode"`
	Title            string     `json:"title"`
	Repo             string     `json:"repo,omitempty"`
	Base             string     `json:"base,omitempty"`
	WorktreeID       string     `json:"worktreeId,omitempty"`
	Status           string     `json:"status"`
	LastRunStatus    string     `json:"lastRunStatus,omitempty"`
	BackendSessionID string     `json:"backendSessionId,omitempty"`
	LastRunID        string     `json:"lastRunId,omitempty"`
	CreatedBy        string     `json:"createdBy,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
	ClosedAt         *time.Time `json:"closedAt,omitempty"`
	Context          string     `json:"-"`
}

// SessionMessage is one durable message in an agent conversation.
type SessionMessage struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionId"`
	RunID     string    `json:"runId,omitempty"`
	Role      string    `json:"role"`
	Kind      string    `json:"kind"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

// TerminalSession is a live pseudo-terminal attached to a workspace session.
type TerminalEvent struct {
	TerminalID string    `json:"terminalId"`
	Seq        int       `json:"seq"`
	Data       string    `json:"data"`
	CreatedAt  time.Time `json:"createdAt"`
}

type TerminalInput struct {
	ID         string    `json:"id"`
	TerminalID string    `json:"terminalId"`
	Seq        int       `json:"seq"`
	Data       string    `json:"data"`
	CreatedAt  time.Time `json:"createdAt"`
}

type TerminalSession struct {
	ID             string     `json:"id"`
	WorkspaceID    string     `json:"workspaceId"`
	AgentSessionID string     `json:"agentSessionId,omitempty"`
	MachineID      string     `json:"machineId,omitempty"`
	Shell          string     `json:"shell"`
	Cwd            string     `json:"cwd,omitempty"`
	Cols           int        `json:"cols"`
	Rows           int        `json:"rows"`
	Status         string     `json:"status"`
	ExitCode       *int       `json:"exitCode,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
	LastSeenAt     *time.Time `json:"lastSeenAt,omitempty"`
}
type SessionComment struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionId"`
	File      string    `json:"file"`
	Line      int       `json:"line"`
	Side      string    `json:"side"`
	Body      string    `json:"body"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}
type GitJob struct {
	ID          string     `json:"id"`
	SessionID   string     `json:"sessionId"`
	WorktreeID  string     `json:"worktreeId"`
	MachineID   string     `json:"machineId"`
	Path        string     `json:"path"`
	Action      string     `json:"action"`
	Message     string     `json:"message"`
	Status      string     `json:"status"`
	ResultJSON  string     `json:"resultJson,omitempty"`
	Error       string     `json:"error,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}
type AgentQuestion struct {
	ID         string     `json:"id"`
	SessionID  string     `json:"sessionId"`
	RunID      string     `json:"runId"`
	Question   string     `json:"question"`
	Options    []string   `json:"options,omitempty"`
	Status     string     `json:"status"`
	Answer     string     `json:"answer,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
	AnsweredAt *time.Time `json:"answeredAt,omitempty"`
	ExpiresAt  *time.Time `json:"expiresAt,omitempty"`
}
type HistoryScan struct {
	ID          string     `json:"id"`
	MachineID   string     `json:"machineId"`
	Status      string     `json:"status"`
	Error       string     `json:"error,omitempty"`
	RequestedAt time.Time  `json:"requestedAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}
type SessionShare struct {
	Token     string     `json:"token"`
	SessionID string     `json:"sessionId"`
	CreatedBy string     `json:"createdBy,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
}

// InstanceCovers reports whether a connection or binding scope covers a work
// item's instance: the same instance, or a parent scope such as a Plane
// workspace "acme" covering "acme/<project>".
func InstanceCovers(scope, instance string) bool {
	return scope == instance || (scope != "" && strings.HasPrefix(instance, scope+"/"))
}

// PlatformStateUnbound marks a synced task whose project is no longer bound
// to its workspace but which has execution history worth keeping.
const PlatformStateUnbound = "unbound"

// Package agent defines the runner seam between the daemon and agent backends.
//
// M0 ships a mock runner. M1 adds an ACP runner that drives Claude Code, Codex
// and other compatible agents over stdio. Keeping the seam small lets both
// coexist without leaking backend details into the daemon.
package agent

import (
	"context"
	"time"
)

// MCPServer is an MCP server exposed to the agent session.
type MCPServer struct {
	Name    string            `json:"name"`
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
}

// Spec describes one execution request.
type Spec struct {
	RunID     string
	SessionID string // backend session to resume, when known
	Repo      string
	Base      string
	Branch    string
	Workdir   string
	Prompt    string
	Backend   string
	Model     string
	Reasoning string // low, medium, high
	Timeout   time.Duration

	// Env is added to the agent process environment.
	Env map[string]string

	// AskTools lists tool names that require human approval before use.
	AskTools []string

	// MCPServers are passed to the agent session.
	MCPServers []MCPServer

	// Approve asks a human to allow a tool call. It is only called for tools
	// matched by AskTools. Returning false denies the call.
	Approve func(ctx context.Context, tool, command, risk string) (bool, error)

	// Ask asks a human a question and waits for the answer.
	Ask func(ctx context.Context, question string, options []string) (string, error)
}

// Event is one streamed execution fact.
type Event struct {
	Type    string
	Payload map[string]any
}

// Usage reports token consumption when the backend exposes it.
type Usage struct {
	InputTokens      int     `json:"inputTokens"`
	OutputTokens     int     `json:"outputTokens"`
	CacheReadTokens  int     `json:"cacheReadTokens"`
	CacheWriteTokens int     `json:"cacheWriteTokens"`
	CostUSD          float64 `json:"costUsd"`
	Estimated        bool    `json:"estimated"`
	Model            string  `json:"model,omitempty"`
}

// Result is the terminal outcome of a run.
type Result struct {
	Status  string // "succeeded" or "failed"
	Code    string
	Message string
	Usage   *Usage
}

// Emitter receives events produced by a runner.
type Emitter func(Event)

// Runner executes one run to completion.
type Runner interface {
	Name() string
	Run(ctx context.Context, spec Spec, emit Emitter) (Result, error)
}

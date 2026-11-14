package agent

import (
	"context"
	"strings"
	"time"
)

// Mock is a deterministic runner used for local development, demos and CI.
// It never calls a real agent CLI and never touches the network.
type Mock struct {
	StepDelay time.Duration
}

// NewMock creates a mock runner. A zero delay defaults to 500 ms per step.
func NewMock(delay time.Duration) *Mock {
	if delay <= 0 {
		delay = 500 * time.Millisecond
	}
	return &Mock{StepDelay: delay}
}

// Name implements Runner.
func (m *Mock) Name() string { return "mock" }

// Run implements Runner.
func (m *Mock) Run(ctx context.Context, spec Spec, emit Emitter) (Result, error) {
	if strings.Contains(spec.Prompt, "[[ask]]") && spec.Ask != nil {
		emit(Event{Type: "agent.message", Payload: map[string]any{"text": "mock agent needs input"}})
		answer, err := spec.Ask(ctx, "Which branch should I use?", []string{"main", "release"})
		if err != nil {
			return Result{Status: "failed", Code: "E_QUESTION", Message: err.Error()}, nil
		}
		emit(Event{Type: "agent.message", Payload: map[string]any{"text": "mock answer: " + answer}})
	}
	steps := []Event{
		{Type: "agent.message", Payload: map[string]any{"text": "mock agent started", "runId": spec.RunID}},
		{Type: "tool.call", Payload: map[string]any{"name": "bash", "args": "go test ./..."}},
		{Type: "tool.result", Payload: map[string]any{"status": "ok", "durationMs": 120}},
		{Type: "agent.message", Payload: map[string]any{"text": "mock agent finished"}},
	}
	for _, ev := range steps {
		select {
		case <-ctx.Done():
			return Result{Status: "failed", Code: "E_CANCELLED", Message: ctx.Err().Error()}, nil
		case <-time.After(m.StepDelay):
			emit(ev)
		}
	}
	return Result{Status: "succeeded"}, nil
}

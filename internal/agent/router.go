package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// Router selects a runner by the backend named in the run spec.
type Router struct {
	Default   Runner
	ByBackend map[string]Runner

	// mu guards ByBackend after SetBackends; hosts re-detect runtimes while
	// runs are in flight.
	mu sync.RWMutex
}

// SetBackends replaces the backend table.
func (r *Router) SetBackends(byBackend map[string]Runner) {
	r.mu.Lock()
	r.ByBackend = byBackend
	r.mu.Unlock()
}

// Name implements Runner.
func (r *Router) Name() string { return "router" }

// Run implements Runner.
func (r *Router) Run(ctx context.Context, spec Spec, emit Emitter) (Result, error) {
	key := strings.ToLower(strings.TrimSpace(spec.Backend))
	r.mu.RLock()
	runner := r.ByBackend[key]
	r.mu.RUnlock()
	if runner == nil {
		runner = r.Default
	}
	if runner == nil {
		return Result{Status: "failed", Code: "E_BACKEND", Message: fmt.Sprintf("no runner for backend %q", spec.Backend)}, nil
	}
	return runner.Run(ctx, spec, emit)
}

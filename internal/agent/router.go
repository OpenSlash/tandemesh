package agent

import (
	"context"
	"fmt"
	"strings"
)

// Router selects a runner by the backend named in the run spec.
type Router struct {
	Default   Runner
	ByBackend map[string]Runner
}

// Name implements Runner.
func (r *Router) Name() string { return "router" }

// Run implements Runner.
func (r *Router) Run(ctx context.Context, spec Spec, emit Emitter) (Result, error) {
	key := strings.ToLower(strings.TrimSpace(spec.Backend))
	runner := r.ByBackend[key]
	if runner == nil {
		runner = r.Default
	}
	if runner == nil {
		return Result{Status: "failed", Code: "E_BACKEND", Message: fmt.Sprintf("no runner for backend %q", spec.Backend)}, nil
	}
	return runner.Run(ctx, spec, emit)
}

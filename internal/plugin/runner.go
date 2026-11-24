package plugin

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/OpenSlash/tandemesh/internal/agent"
)

// ServiceName is the gRPC service a plugin must implement.
const ServiceName = "orca.plugin.v1.Runner"

// Runner executes agent runs in an external process.
type Runner struct {
	conn    *grpc.ClientConn
	timeout time.Duration
}

// Dial connects to a plugin. The connection is lazy; use Ping to probe.
func Dial(addr string) (*Runner, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("plugin dial: %w", err)
	}
	return &Runner{conn: conn, timeout: 30 * time.Minute}, nil
}

// Close closes the connection.
func (r *Runner) Close() error { return r.conn.Close() }

// Name implements agent.Runner.
func (r *Runner) Name() string { return "grpc-plugin" }

// Run implements agent.Runner. Events are replayed to the local emitter, so a
// plugin behaves like an in-process runner.
func (r *Runner) Run(ctx context.Context, spec agent.Spec, emit agent.Emitter) (agent.Result, error) {
	req := &RunRequest{
		RunID:   spec.RunID,
		Repo:    spec.Repo,
		Branch:  spec.Branch,
		Workdir: spec.Workdir,
		Prompt:  spec.Prompt,
		AgentID: spec.Backend,
	}
	var resp RunResponse
	if err := r.conn.Invoke(ctx, "/"+ServiceName+"/Run", req, &resp, grpc.ForceCodec(Codec{})); err != nil {
		return agent.Result{Status: "failed", Code: "E_PLUGIN", Message: err.Error()}, nil
	}
	if err := resp.Validate(); err != nil {
		return agent.Result{Status: "failed", Code: "E_PLUGIN", Message: err.Error()}, nil
	}
	for _, ev := range resp.Events {
		emit(agent.Event{Type: ev.Type, Payload: ev.Payload})
	}
	return agent.Result{Status: resp.Status, Code: resp.Code, Message: resp.Message}, nil
}

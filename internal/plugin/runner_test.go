package plugin

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"

	"github.com/OpenSlash/tandemesh/internal/agent"
)

type testPlugin interface {
	Run(context.Context, *RunRequest) (*RunResponse, error)
}

type testImpl struct{}

func (testImpl) Run(_ context.Context, req *RunRequest) (*RunResponse, error) {
	return &RunResponse{
		Status: "succeeded",
		Events: []Event{
			{Type: "agent.message", Payload: map[string]any{"text": "hello " + req.RunID}},
		},
	}, nil
}

var testDesc = grpc.ServiceDesc{
	ServiceName: ServiceName,
	HandlerType: (*testPlugin)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Run",
			Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
				in := new(RunRequest)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(testPlugin).Run(ctx, in)
				}
				info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/" + ServiceName + "/Run"}
				handler := func(ctx context.Context, req any) (any, error) {
					return srv.(testPlugin).Run(ctx, req.(*RunRequest))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
	},
}

func startPlugin(t *testing.T) string {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer(grpc.ForceServerCodec(Codec{}))
	srv.RegisterService(&testDesc, testImpl{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)
	return lis.Addr().String()
}

func TestRunnerOverGRPC(t *testing.T) {
	addr := startPlugin(t)
	runner, err := Dial(addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = runner.Close() })

	var events []agent.Event
	res, err := runner.Run(context.Background(), agent.Spec{RunID: "run_1", Prompt: "hi"}, func(e agent.Event) {
		events = append(events, e)
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Status != "succeeded" {
		t.Fatalf("result = %+v", res)
	}
	if len(events) != 1 || events[0].Type != "agent.message" {
		t.Fatalf("events = %+v", events)
	}
	if events[0].Payload["text"] != "hello run_1" {
		t.Fatalf("payload = %+v", events[0].Payload)
	}
}

func TestValidateStatus(t *testing.T) {
	if err := (RunResponse{Status: "done"}).Validate(); err == nil {
		t.Fatal("invalid status accepted")
	}
	if err := (RunResponse{Status: "failed"}).Validate(); err != nil {
		t.Fatal(err)
	}
}

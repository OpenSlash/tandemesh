package plugin

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	internal "github.com/OpenSlash/tandemesh/internal/plugin"
)

type echo struct{}

func (echo) Run(_ context.Context, req *Request) (*Response, error) {
	return &Response{
		Status: "succeeded",
		Events: []Event{{Type: "agent.message", Payload: map[string]any{"text": req.Prompt}}},
	}, nil
}

func TestServeAndRun(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := Serve(lis, echo{})
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	var resp Response
	err = conn.Invoke(context.Background(), "/"+internal.ServiceName+"/Run",
		&Request{RunID: "run_1", Prompt: "ship it"}, &resp, grpc.ForceCodec(internal.Codec{}))
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}
	if resp.Status != "succeeded" || len(resp.Events) != 1 || resp.Events[0].Payload["text"] != "ship it" {
		t.Fatalf("resp = %+v", resp)
	}
}

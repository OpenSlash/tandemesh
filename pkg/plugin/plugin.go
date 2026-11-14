// Package plugin is the public SDK for writing Orca runner plugins.
//
// A plugin is a gRPC server implementing /orca.plugin.v1.Runner/Run with JSON
// encoding. Use Serve to publish a handler; the Orca daemon connects with
// -runner grpc -grpc-addr host:port.
package plugin

import (
	"context"
	"net"

	"google.golang.org/grpc"

	internal "github.com/OpenSlash/tandemesh/internal/plugin"
)

// Request is the run request sent by the daemon.
type Request = internal.RunRequest

// Event is one update emitted by the plugin.
type Event = internal.Event

// Response is the terminal result.
type Response = internal.RunResponse

// Handler implements a plugin.
type Handler interface {
	Run(ctx context.Context, req *Request) (*Response, error)
}

// Serve registers the handler and serves gRPC on the listener.
func Serve(lis net.Listener, handler Handler, opts ...grpc.ServerOption) *grpc.Server {
	options := append([]grpc.ServerOption{grpc.ForceServerCodec(internal.Codec{})}, opts...)
	srv := grpc.NewServer(options...)
	srv.RegisterService(&serviceDesc, handler)
	return srv
}

var serviceDesc = grpc.ServiceDesc{
	ServiceName: internal.ServiceName,
	HandlerType: (*Handler)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Run",
			Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
				in := new(Request)
				if err := dec(in); err != nil {
					return nil, err
				}
				if interceptor == nil {
					return srv.(Handler).Run(ctx, in)
				}
				info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/" + internal.ServiceName + "/Run"}
				handler := func(ctx context.Context, req any) (any, error) {
					return srv.(Handler).Run(ctx, req.(*Request))
				}
				return interceptor(ctx, in, info, handler)
			},
		},
	},
}

package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"time"

	sdkprotocol "github.com/bobacgo/cron-job/internal/executor/sdk/protocol"
	grpcpkg "google.golang.org/grpc"
)

const defaultMethod = "/cronjob.v1.Executor/Run"

type workerServer struct{}

func (workerServer) Run(_ context.Context, req *sdkprotocol.RunRequest) (*sdkprotocol.RunResponse, error) {
	message := "worker executed run " + req.RunID
	return &sdkprotocol.RunResponse{
		Status:  "succeeded",
		Message: "ok",
		Output:  message,
	}, nil
}

func main() {
	addr := flag.String("addr", envOr("SDK_WORKER_ADDR", ":50051"), "gRPC listen address")
	flag.Parse()

	sdkprotocol.RegisterJSONCodec()

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	server := grpcpkg.NewServer(grpcpkg.ForceServerCodec(sdkprotocol.JSONCodec{}))
	registerExecutorService(server, workerServer{})

	log.Printf("sdk worker listening on %s at %s", *addr, time.Now().UTC().Format(time.RFC3339))
	if err := server.Serve(listener); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func registerExecutorService(server *grpcpkg.Server, impl workerServer) {
	service := &grpcpkg.ServiceDesc{
		ServiceName: "cronjob.v1.Executor",
		HandlerType: (*workerServer)(nil),
		Methods: []grpcpkg.MethodDesc{
			{
				MethodName: "Run",
				Handler: func(srv any, ctx context.Context, dec func(any) error, interceptor grpcpkg.UnaryServerInterceptor) (any, error) {
					in := new(sdkprotocol.RunRequest)
					if err := dec(in); err != nil {
						return nil, err
					}
					if interceptor == nil {
						return impl.Run(ctx, in)
					}
					info := &grpcpkg.UnaryServerInfo{Server: srv, FullMethod: defaultMethod}
					handler := func(ctx context.Context, req any) (any, error) {
						return impl.Run(ctx, req.(*sdkprotocol.RunRequest))
					}
					return interceptor(ctx, in, info, handler)
				},
			},
		},
	}
	server.RegisterService(service, impl)
}

func envOr(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

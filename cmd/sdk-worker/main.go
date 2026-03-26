package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"time"

	sdkprotocol "github.com/bobacgo/cron-job/internal/executor/sdk/protocol"
	"github.com/bobacgo/cron-job/kit/core"
	grpcpkg "google.golang.org/grpc"
)

const defaultMethod = "/cronjob.v1.Executor/Run"

type workerServer struct{}

func (workerServer) Run(_ context.Context, req *sdkprotocol.RunRequest) (*sdkprotocol.RunResponse, error) {
	version, ok := sdkprotocol.NegotiateVersion(req.ProtocolVersion, req.SupportedVersions)
	if !ok {
		return &sdkprotocol.RunResponse{
			ProtocolVersion: sdkprotocol.CurrentVersion,
			Status:          "failed",
			ErrorCode:       sdkprotocol.ErrorCodeInvalidVersion,
			Message:         "no compatible protocol version",
		}, nil
	}
	if req.JobID == "" || req.RunID == "" {
		return &sdkprotocol.RunResponse{
			ProtocolVersion: version,
			Status:          "failed",
			ErrorCode:       sdkprotocol.ErrorCodeInvalidRequest,
			Message:         "job_id and run_id are required",
		}, nil
	}
	message := "worker executed run " + req.RunID
	return &sdkprotocol.RunResponse{
		ProtocolVersion: version,
		Status:          "succeeded",
		ErrorCode:       sdkprotocol.ErrorCodeNone,
		Message:         "ok",
		Output:          message,
	}, nil
}

func main() {
	addr := flag.String("addr", envOr("SDK_WORKER_ADDR", ":50051"), "gRPC listen address")
	flag.Parse()

	sdkprotocol.RegisterJSONCodec()

	grpcServer := core.NewGRPCServer(*addr, grpcpkg.ForceServerCodec(sdkprotocol.JSONCodec{}))
	server := core.NewServer[struct{}]()
	server.SetShutdownTimeout(10 * time.Second)
	server.Add(func(*struct{}) (core.Service, error) {
		return grpcServer, nil
	})
	registerExecutorService(grpcServer.Server(), workerServer{})

	log.Printf("sdk worker listening on %s at %s", *addr, time.Now().UTC().Format(time.RFC3339))
	if err := server.Run(); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, grpcpkg.ErrServerStopped) {
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

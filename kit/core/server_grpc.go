package core

import (
	"context"
	"net"

	"google.golang.org/grpc"
)

type GRPCServer struct {
	addr     string
	server   *grpc.Server
	listener net.Listener
}

func NewGRPCServer(addr string, opts ...grpc.ServerOption) *GRPCServer {
	return &GRPCServer{
		addr:   addr,
		server: grpc.NewServer(opts...),
	}
}

func (s *GRPCServer) Server() *grpc.Server {
	return s.server
}

func (s *GRPCServer) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return err
	}
	s.listener = listener
	return s.server.Serve(listener)
}

func (s *GRPCServer) Stop(ctx context.Context) error {
	stopped := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(stopped)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-stopped:
		return nil
	}
}

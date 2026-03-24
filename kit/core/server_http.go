package core

import (
	"context"
	"net/http"
	"time"
)

type HTTPServer struct {
	server *http.Server
}

func NewHTTPServer(addr string, handler http.Handler, readHeaderTimeout time.Duration) *HTTPServer {
	return &HTTPServer{
		server: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: readHeaderTimeout,
		},
	}
}

func (s *HTTPServer) Start(ctx context.Context) error {
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Stop(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.Shutdown(ctx)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

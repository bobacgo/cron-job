package hs

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

type Engine struct {
	addr    string
	srv     *http.Server
	handler http.Handler
}

func New(addr string) *Engine {
	return &Engine{
		addr:    addr,
		handler: http.DefaultServeMux,
	}
}

func (e *Engine) SetHandler(handler http.Handler) {
	e.handler = handler
}

func (e *Engine) Start(ctx context.Context) error {
	e.srv = &http.Server{
		Addr:    e.addr,
		Handler: e.handler,
	}
	slog.Info("Starting server", "address", e.addr)
	return e.srv.ListenAndServe()
}

func (e *Engine) Stop(ctx context.Context) error {
	slog.Info("Shutting down server gracefully")

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := e.srv.Shutdown(ctx); err != nil {
		slog.Error("Server shutdown failed", "error", err)
		return err
	}
	slog.Info("Server stopped gracefully")
	return nil
}

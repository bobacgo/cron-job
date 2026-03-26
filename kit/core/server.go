package core

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const defaultShutdownTimeout = 10 * time.Second // 默认的服务器优雅停机超时时间

type Service interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type Server[T any] struct {
	App *T

	// 多个 service 组成一个 server，server 负责统一的生命周期管理。
	svcs []Service
	// beforeStartHooks 在所有 service 启动前执行，用于初始化配置、日志等依赖。
	beforeStartHooks []func(context.Context) error

	sigCh           chan os.Signal
	shutdownTimeout time.Duration
}

func NewServer[T any]() *Server[T] {
	server := &Server[T]{
		App:             new(T),
		shutdownTimeout: defaultShutdownTimeout,
		sigCh:           make(chan os.Signal, 1),
	}
	return server
}

func (s *Server[T]) Init(initFunc func(*T) error) {
	if initFunc != nil {
		if err := initFunc(s.App); err != nil {
			log.Fatalf("failed to initialize app: %v", err)
		}
	}
}

// Add 添加一个 Service 到 Server 中，Server 将负责管理该 Service 的生命周期。
func (s *Server[T]) Add(svcFunc func(*T) (Service, error)) {
	if svcFunc == nil {
		return
	}
	svc, err := svcFunc(s.App)
	if err != nil {
		log.Fatalf("failed to create service: %v", err)
	}
	s.svcs = append(s.svcs, svc)
}

func (s *Server[T]) SetShutdownTimeout(timeout time.Duration) {
	s.shutdownTimeout = timeout
}

// Run 启动 Server 中的所有 Service，并监听系统信号以实现优雅停机。
func (s *Server[T]) Run() error {
	errCh := make(chan error, len(s.svcs))
	for _, svc := range s.svcs {
		currentSvc := svc
		go func() {
			if err := currentSvc.Start(context.Background()); err != nil {
				errCh <- fmt.Errorf("failed to start service: %w", err)
			}
		}()
	}

	signal.Notify(s.sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(s.sigCh)

	select {
	case err := <-errCh:
		return err
	case <-s.sigCh:
	}

	if err := s.Stop(); err != nil {
		return err
	}

	// 服务 Stop 后，Serve/ListenAndServe 通常会返回可预期错误，读取并忽略这些非致命错误。
	for {
		select {
		case err := <-errCh:
			log.Printf("service exit after shutdown: %v", err)
		default:
			return nil
		}
	}
}

// Stop 停止 Server 中的所有 Service，确保在指定的超时时间内完成优雅停机。
func (s *Server[T]) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, len(s.svcs))

	for _, svc := range s.svcs {
		currentSvc := svc
		wg.Go(func() {
			if err := currentSvc.Stop(ctx); err != nil {
				errCh <- errors.New("failed to stop service: " + err.Error())
			}
		})
	}

	waitDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitDone)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-waitDone:
	}

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

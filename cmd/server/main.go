package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"time"

	jobapp "github.com/bobacgo/cron-job/internal/app/job"
	"github.com/bobacgo/cron-job/internal/config"
	dispatchercancel "github.com/bobacgo/cron-job/internal/dispatcher/cancel"
	dispatcherlease "github.com/bobacgo/cron-job/internal/dispatcher/lease"
	dispatcherloop "github.com/bobacgo/cron-job/internal/dispatcher/loop"
	"github.com/bobacgo/cron-job/internal/dispatcher/queue"
	"github.com/bobacgo/cron-job/internal/executor"
	binaryexec "github.com/bobacgo/cron-job/internal/executor/binary"
	sdkgrpc "github.com/bobacgo/cron-job/internal/executor/sdk/grpc"
	sdkhttp "github.com/bobacgo/cron-job/internal/executor/sdk/http"
	shellexec "github.com/bobacgo/cron-job/internal/executor/shell"
	dependencyrepo "github.com/bobacgo/cron-job/internal/repository/dependency"
	jobrepo "github.com/bobacgo/cron-job/internal/repository/job"
	jobrunrepo "github.com/bobacgo/cron-job/internal/repository/jobrun"
	logrepo "github.com/bobacgo/cron-job/internal/repository/log"
	mysqlrepo "github.com/bobacgo/cron-job/internal/repository/sqlite"
	schedulerloop "github.com/bobacgo/cron-job/internal/scheduler/loop"
	"github.com/bobacgo/cron-job/internal/scheduler/planner"
	adminhandler "github.com/bobacgo/cron-job/internal/transport/admin/handler"
	httpapihandler "github.com/bobacgo/cron-job/internal/transport/httpapi/handler"
	"github.com/bobacgo/cron-job/internal/transport/httpapi/router"
	"github.com/bobacgo/cron-job/kit/core"
	"github.com/bobacgo/cron-job/kit/slogx"
)

func main() {
	cfg := config.Load()

	db, err := mysqlrepo.Open(cfg.DBDSN)
	if err != nil {
		log.Fatalf("init mysql: %v", err)
	}
	defer db.Close()

	jobStore := jobrepo.NewMySQLRepository(db)
	jobRunStore := jobrunrepo.NewMySQLRepository(db)
	dependencyStore := dependencyrepo.NewMySQLRepository(db)
	runLogStore, err := logrepo.NewFileRepository(cfg.LogDir)
	if err != nil {
		log.Fatalf("init log repository: %v", err)
	}
	readyQueue := queue.NewInMemoryQueue()
	leaseManager := dispatcherlease.NewMemoryManager(30 * time.Second)
	runCancelManager := dispatchercancel.NewManager()
	plannerSvc := planner.New()

	executorRegistry := executor.NewRegistry()
	executorRegistry.Register("sdk-http", sdkhttp.NewExecutor(http.DefaultClient))
	executorRegistry.Register("sdk-grpc", sdkgrpc.NewExecutor())
	executorRegistry.Register("binary", binaryexec.NewExecutor())
	executorRegistry.Register("shell", shellexec.NewExecutor())

	jobService := jobapp.NewService(jobStore, jobRunStore, dependencyStore, runLogStore, readyQueue, plannerSvc)
	jobService.SetRunCanceler(runCancelManager)
	apiHandler := httpapihandler.NewJobHandler(jobService)
	adminPages := adminhandler.NewPageHandler(jobService, jobRunStore, runLogStore)

	scheduleLoop := schedulerloop.New(jobStore, jobRunStore, dependencyStore, readyQueue, plannerSvc)
	dependencyLoop := schedulerloop.NewDependency(dependencyStore, jobRunStore, readyQueue)
	runLoop := dispatcherloop.New(jobStore, jobRunStore, runLogStore, readyQueue, leaseManager, runCancelManager, executorRegistry)

	_, cancel := context.WithCancel(context.Background())
	loopSvc := core.NewBackgroundService(func(loopCtx context.Context) {
		go scheduleLoop.Start(loopCtx, 5*time.Second)
		go dependencyLoop.Start(loopCtx, 2*time.Second)
		go runLoop.Start(loopCtx, 2*time.Second)
	}, cancel)

	httpServer := core.NewHTTPServer(cfg.HTTPAddr, router.New(apiHandler, adminPages), 5*time.Second)
	server := core.NewServer(httpServer, loopSvc)
	server.SetShutdownTimeout(10 * time.Second)
	server.BeforeFunc(nil)        // 加载配置
	server.BeforeFunc(initLogger) // 初始化日志
	server.BeforeFunc(nil)        // 初始化数据库

	slog.Info("cron service starting", "addr", cfg.HTTPAddr)
	if err := server.Run(); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, http.ErrServerClosed) {
		slogx.Fatal(context.Background(), "server failed", "error", err)
	}
}

func initLogger(ctx context.Context) error {
	slogx.Init()
	return nil
}

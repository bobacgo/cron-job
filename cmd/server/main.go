package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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
	dependencyrepo "github.com/bobacgo/cron-job/internal/repository/dependency"
	jobrepo "github.com/bobacgo/cron-job/internal/repository/job"
	jobrunrepo "github.com/bobacgo/cron-job/internal/repository/jobrun"
	logrepo "github.com/bobacgo/cron-job/internal/repository/log"
	sqliterepo "github.com/bobacgo/cron-job/internal/repository/sqlite"
	schedulerloop "github.com/bobacgo/cron-job/internal/scheduler/loop"
	"github.com/bobacgo/cron-job/internal/scheduler/planner"
	adminhandler "github.com/bobacgo/cron-job/internal/transport/admin/handler"
	httpapihandler "github.com/bobacgo/cron-job/internal/transport/httpapi/handler"
	"github.com/bobacgo/cron-job/internal/transport/httpapi/router"
)

func main() {
	cfg := config.Load()

	db, err := sqliterepo.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("init sqlite: %v", err)
	}
	defer db.Close()

	jobStore := jobrepo.NewSQLiteRepository(db)
	jobRunStore := jobrunrepo.NewSQLiteRepository(db)
	dependencyStore := dependencyrepo.NewSQLiteRepository(db)
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

	jobService := jobapp.NewService(jobStore, jobRunStore, dependencyStore, runLogStore, readyQueue, plannerSvc)
	jobService.SetRunCanceler(runCancelManager)
	apiHandler := httpapihandler.NewJobHandler(jobService)
	adminPages := adminhandler.NewPageHandler(jobService, jobRunStore, runLogStore)

	scheduleLoop := schedulerloop.New(jobStore, jobRunStore, dependencyStore, readyQueue, plannerSvc)
	dependencyLoop := schedulerloop.NewDependency(dependencyStore, jobRunStore, readyQueue)
	runLoop := dispatcherloop.New(jobStore, jobRunStore, runLogStore, readyQueue, leaseManager, runCancelManager, executorRegistry)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go scheduleLoop.Start(ctx, 5*time.Second)
	go dependencyLoop.Start(ctx, 2*time.Second)
	go runLoop.Start(ctx, 2*time.Second)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router.New(apiHandler, adminPages),
		ReadHeaderTimeout: 5 * time.Second,
	}

	shutdownDone := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("server shutdown error: %v", err)
		}
		close(shutdownDone)
	}()

	log.Printf("cron service listening on %s", cfg.HTTPAddr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server failed: %v", err)
	}

	<-shutdownDone
}

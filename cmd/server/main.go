package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/bobacgo/cron-job/internal/admin"
	jobapp "github.com/bobacgo/cron-job/internal/app/job"
	"github.com/bobacgo/cron-job/internal/config"
	"github.com/bobacgo/cron-job/internal/dispatcher"
	dispatcherloop "github.com/bobacgo/cron-job/internal/dispatcher/loop"
	"github.com/bobacgo/cron-job/internal/executor"
	binaryexec "github.com/bobacgo/cron-job/internal/executor/binary"
	sdkgrpc "github.com/bobacgo/cron-job/internal/executor/sdk/grpc"
	sdkhttp "github.com/bobacgo/cron-job/internal/executor/sdk/http"
	shellexec "github.com/bobacgo/cron-job/internal/executor/shell"
	"github.com/bobacgo/cron-job/internal/repository"
	"github.com/bobacgo/cron-job/internal/scheduler"
	schedulerloop "github.com/bobacgo/cron-job/internal/scheduler/loop"
	httpapihandler "github.com/bobacgo/cron-job/internal/transport/httpapi/handler"
	"github.com/bobacgo/cron-job/internal/transport/httpapi/router"
	"github.com/bobacgo/cron-job/kit/core"
	"github.com/bobacgo/cron-job/kit/slogx"
	"github.com/bobacgo/cron-job/kit/sqlx"
	"github.com/bobacgo/cron-job/kit/types"

	_ "github.com/go-sql-driver/mysql"
)

type Beans struct {
	APIHandler     *httpapihandler.JobHandler
	AuthHandler    *httpapihandler.AuthHandler
	MgrHandler     *httpapihandler.MgrHandler
	ScheduleLoop   *schedulerloop.Loop
	DependencyLoop *schedulerloop.DependencyLoop
	RunLoop        *dispatcherloop.Loop
}

type App struct {
	Cfg      *config.Config
	DB       types.KV[sqlx.DB]
	HttpAddr string
	Beans    *Beans
}

func main() {
	server := core.NewServer[App]()
	server.SetShutdownTimeout(10 * time.Second)

	// 加载配置
	server.Init(func(app *App) error {
		var err error
		app.Cfg, err = config.Load()
		if err != nil {
			return err
		}
		app.HttpAddr = app.Cfg.HTTPAddr
		return err
	})
	// 初始化日志
	server.Init(func(app *App) error {
		slogx.Init()
		return nil
	})
	// 初始化数据库
	server.Init(func(app *App) error {
		var err error
		app.DB, err = sqlx.NewDBManager(app.Cfg.Database)
		return err
	})
	server.Init(func(app *App) error {
		return initBeans(app)
	})

	// 添加 http 服务
	server.Add(func(a *App) (core.Service, error) {
		httpServer := core.NewHTTPServer(a.HttpAddr, router.New(a.Beans.APIHandler, a.Beans.MgrHandler), 5*time.Second)
		return httpServer, nil
	})

	// 添加调度和执行后台服务
	server.Add(func(a *App) (core.Service, error) {
		_, cancel := context.WithCancel(context.Background())
		loopSvc := core.NewBackgroundService(func(loopCtx context.Context) {
			go a.Beans.ScheduleLoop.Start(loopCtx, 5*time.Second)
			go a.Beans.DependencyLoop.Start(loopCtx, 2*time.Second)
			go a.Beans.RunLoop.Start(loopCtx, 2*time.Second)
		}, cancel)
		return loopSvc, nil
	})

	slog.Info("cron service starting", "addr", server.App.Cfg.HTTPAddr)
	if err := server.Run(); err != nil &&
		!errors.Is(err, context.Canceled) &&
		!errors.Is(err, context.DeadlineExceeded) &&
		!errors.Is(err, http.ErrServerClosed) {
		slogx.Fatal(context.Background(), "server failed", "error", err)
	}
}

// 对象容器
func initBeans(app *App) error {
	db := app.DB.Default()
	adminSvc := admin.New(db)
	if err := adminSvc.Bootstrap(context.Background()); err != nil {
		return err
	}
	repo := repository.NewRepo(app.Cfg, db)

	executorRegistry := executor.NewRegistry()
	executorRegistry.Register("sdk-http", sdkhttp.NewExecutor(http.DefaultClient))
	executorRegistry.Register("sdk-grpc", sdkgrpc.NewExecutor())
	executorRegistry.Register("binary", binaryexec.NewExecutor())
	executorRegistry.Register("shell", shellexec.NewExecutor())

	dispcher := dispatcher.NewDispatcher(repo, executorRegistry)
	schedulerSvc := scheduler.NewScheduler(repo, dispcher.ReadyQueue)

	jobService := jobapp.NewService(repo, dispcher.ReadyQueue, schedulerSvc.Planner)
	jobService.SetRunCanceler(dispcher.RunCancelManager)
	authHandler := httpapihandler.NewAuthHandler()
	app.Beans = &Beans{
		APIHandler:     httpapihandler.NewJobHandler(jobService),
		AuthHandler:    authHandler,
		MgrHandler:     httpapihandler.NewMgrHandler(adminSvc, authHandler),
		ScheduleLoop:   schedulerloop.New(repo, dispcher.ReadyQueue, schedulerSvc.Planner),
		DependencyLoop: schedulerloop.NewDependency(repo.Dependencies, repo.JobRun, dispcher.ReadyQueue),
		RunLoop:        dispatcherloop.New(repo, dispcher.ReadyQueue, dispcher.LeaseManager, dispcher.RunCancelManager, executorRegistry),
	}
	return nil
}

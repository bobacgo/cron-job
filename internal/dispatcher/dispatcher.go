package dispatcher

import (
	"time"

	"github.com/bobacgo/cron-job/internal/dispatcher/cancel"
	"github.com/bobacgo/cron-job/internal/dispatcher/lease"
	"github.com/bobacgo/cron-job/internal/dispatcher/loop"
	"github.com/bobacgo/cron-job/internal/dispatcher/queue"
	"github.com/bobacgo/cron-job/internal/executor"
	"github.com/bobacgo/cron-job/internal/repository"
)

// 调度器
type Dispatcher struct {
	RunCancelManager *cancel.Manager
	LeaseManager     lease.Manager
	Loop             *loop.Loop
	ReadyQueue       queue.Queue
}

func NewDispatcher(repo *repository.Repo, executors *executor.Registry) *Dispatcher {
	runCancelManager := cancel.NewManager()
	leaseMgr := lease.NewMemoryManager(30 * time.Second)
	readyQueue := queue.NewInMemoryQueue()
	return &Dispatcher{
		RunCancelManager: runCancelManager,
		LeaseManager:     leaseMgr,
		Loop:             loop.New(repo, readyQueue, leaseMgr, runCancelManager, executors),
		ReadyQueue:       readyQueue,
	}
}

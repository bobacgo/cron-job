package scheduler

import (
	"github.com/bobacgo/cron-job/internal/dispatcher/queue"
	"github.com/bobacgo/cron-job/internal/repository"
	"github.com/bobacgo/cron-job/internal/scheduler/loop"
	"github.com/bobacgo/cron-job/internal/scheduler/planner"
)

type Scheduler struct {
	Loop    *loop.Loop
	Planner *planner.Planner
}

func NewScheduler(repo *repository.Repo, queue queue.Queue) *Scheduler {
	planner := planner.New()
	return &Scheduler{
		Loop:    loop.New(repo, queue, planner),
		Planner: planner,
	}
}

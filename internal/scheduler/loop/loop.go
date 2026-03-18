package loop

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"time"

	"github.com/bobacgo/cron-job/internal/dispatcher/queue"
	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
	dependencyrepo "github.com/bobacgo/cron-job/internal/repository/dependency"
	jobrepo "github.com/bobacgo/cron-job/internal/repository/job"
	jobrunrepo "github.com/bobacgo/cron-job/internal/repository/jobrun"
	"github.com/bobacgo/cron-job/internal/scheduler/planner"
)

type Loop struct {
	jobs    jobrepo.Repository
	runs    jobrunrepo.Repository
	deps    dependencyrepo.Repository
	queue   queue.Queue
	planner *planner.Planner
}

func New(jobs jobrepo.Repository, runs jobrunrepo.Repository, deps dependencyrepo.Repository, queue queue.Queue, planner *planner.Planner) *Loop {
	return &Loop{jobs: jobs, runs: runs, deps: deps, queue: queue, planner: planner}
}

func (l *Loop) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	l.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.tick(ctx)
		}
	}
}

func (l *Loop) tick(ctx context.Context) {
	jobs, err := l.jobs.ListEnabled(ctx)
	if err != nil {
		log.Printf("schedule loop list jobs: %v", err)
		return
	}

	now := time.Now().UTC()
	for _, item := range jobs {
		if item.NextRunAt.IsZero() {
			nextRunAt, err := l.planner.Next(item, now)
			if err != nil {
				log.Printf("schedule loop bootstrap job %s: %v", item.ID, err)
				continue
			}
			item.NextRunAt = nextRunAt
			item.UpdatedAt = now
			if err := l.jobs.Save(ctx, item); err != nil {
				log.Printf("schedule loop save bootstrap job %s: %v", item.ID, err)
			}
			continue
		}

		dueAt, due, err := l.planner.Due(item, now)
		if err != nil {
			log.Printf("schedule loop plan job %s: %v", item.ID, err)
			continue
		}
		if !due {
			continue
		}

		status := jobrundomain.StatusReady
		edges, err := l.deps.ListByJob(ctx, item.ID)
		if err != nil {
			log.Printf("schedule loop list dependencies %s: %v", item.ID, err)
			continue
		}
		if len(edges) > 0 {
			status = jobrundomain.StatusBlocked
		}

		run := jobrundomain.JobRun{
			ID:          newID(),
			JobID:       item.ID,
			ScheduledAt: dueAt,
			Status:      status,
			Attempt:     1,
			TriggerType: "schedule",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if _, exists, err := l.runs.FindByDedupKey(ctx, run.DedupKey()); err != nil {
			log.Printf("schedule loop dedup lookup: %v", err)
			continue
		} else if exists {
			continue
		}
		if err := l.runs.Save(ctx, run); err != nil {
			log.Printf("schedule loop save run: %v", err)
			continue
		}
		if status == jobrundomain.StatusReady {
			if err := l.queue.Enqueue(ctx, run.ID); err != nil {
				log.Printf("schedule loop enqueue run: %v", err)
				continue
			}
		}

		item.LastRunAt = dueAt
		item.NextRunAt, err = l.planner.Next(item, dueAt)
		if err != nil {
			log.Printf("schedule loop next run job %s: %v", item.ID, err)
			continue
		}
		item.UpdatedAt = now
		if err := l.jobs.Save(ctx, item); err != nil {
			log.Printf("schedule loop update job %s: %v", item.ID, err)
		}
	}
}

func newID() string {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

package loop

import (
	"context"
	"log"
	"time"

	"github.com/bobacgo/cron-job/internal/dispatcher/queue"
	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
	"github.com/bobacgo/cron-job/internal/repository"
)

type DependencyLoop struct {
	dependencies repository.DependencyRepository
	runs         repository.JobRunRepository
	queue        queue.Queue
}

func NewDependency(dependencies repository.DependencyRepository, runs repository.JobRunRepository, queue queue.Queue) *DependencyLoop {
	return &DependencyLoop{dependencies: dependencies, runs: runs, queue: queue}
}

func (l *DependencyLoop) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.tick(ctx)
		}
	}
}

func (l *DependencyLoop) tick(ctx context.Context) {
	blockedRuns, err := l.runs.ListByStatus(ctx, jobrundomain.StatusBlocked)
	if err != nil {
		log.Printf("dependency loop list blocked runs: %v", err)
		return
	}
	for _, run := range blockedRuns {
		ready, err := l.ready(ctx, run.JobID)
		if err != nil {
			log.Printf("dependency loop check run %s: %v", run.ID, err)
			continue
		}
		if !ready {
			continue
		}
		run.Status = jobrundomain.StatusReady
		run.UpdatedAt = time.Now().UTC()
		if err := l.runs.Save(ctx, run); err != nil {
			log.Printf("dependency loop update run %s: %v", run.ID, err)
			continue
		}
		if err := l.queue.Enqueue(ctx, run.ID); err != nil {
			log.Printf("dependency loop enqueue run %s: %v", run.ID, err)
		}
	}
}

func (l *DependencyLoop) ready(ctx context.Context, jobID string) (bool, error) {
	edges, err := l.dependencies.ListByJob(ctx, jobID)
	if err != nil {
		return false, err
	}
	if len(edges) == 0 {
		return true, nil
	}
	for _, edge := range edges {
		runs, err := l.runs.ListByJob(ctx, edge.DependsOnJobID)
		if err != nil {
			return false, err
		}
		if len(runs) == 0 || runs[0].Status != jobrundomain.StatusSucceeded {
			return false, nil
		}
	}
	return true, nil
}

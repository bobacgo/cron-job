package loop

import (
	"context"
	"testing"
	"time"

	"github.com/bobacgo/cron-job/internal/dispatcher/queue"
	dependencydomain "github.com/bobacgo/cron-job/internal/domain/dependency"
	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
	dependencyrepo "github.com/bobacgo/cron-job/internal/repository/dependency"
	jobrunrepo "github.com/bobacgo/cron-job/internal/repository/jobrun"
)

func TestDependencyLoopReleasesBlockedRun(t *testing.T) {
	ctx := context.Background()
	deps := dependencyrepo.NewInMemoryRepository()
	runs := jobrunrepo.NewInMemoryRepository()
	q := queue.NewInMemoryQueue()

	if err := deps.Replace(ctx, "downstream", []dependencydomain.Edge{{JobID: "downstream", DependsOnJobID: "upstream"}}); err != nil {
		t.Fatalf("deps.Replace() error = %v", err)
	}
	now := time.Now().UTC()
	if err := runs.Save(ctx, jobrundomain.JobRun{
		ID:          "run-upstream",
		JobID:       "upstream",
		ScheduledAt: now,
		Status:      jobrundomain.StatusSucceeded,
		Attempt:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("runs.Save(upstream) error = %v", err)
	}
	if err := runs.Save(ctx, jobrundomain.JobRun{
		ID:          "run-downstream",
		JobID:       "downstream",
		ScheduledAt: now,
		Status:      jobrundomain.StatusBlocked,
		Attempt:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("runs.Save(downstream) error = %v", err)
	}

	l := NewDependency(deps, runs, q)
	l.tick(ctx)

	run, err := runs.Get(ctx, "run-downstream")
	if err != nil {
		t.Fatalf("runs.Get() error = %v", err)
	}
	if run.Status != jobrundomain.StatusReady {
		t.Fatalf("downstream status = %s, want %s", run.Status, jobrundomain.StatusReady)
	}
	queued, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("queue.Dequeue() error = %v", err)
	}
	if queued != "run-downstream" {
		t.Fatalf("dequeue run id = %s, want run-downstream", queued)
	}
}

func TestDependencyLoopKeepsBlockedWhenUpstreamNotSucceeded(t *testing.T) {
	ctx := context.Background()
	deps := dependencyrepo.NewInMemoryRepository()
	runs := jobrunrepo.NewInMemoryRepository()
	q := queue.NewInMemoryQueue()

	if err := deps.Replace(ctx, "downstream", []dependencydomain.Edge{{JobID: "downstream", DependsOnJobID: "upstream"}}); err != nil {
		t.Fatalf("deps.Replace() error = %v", err)
	}
	now := time.Now().UTC()
	_ = runs.Save(ctx, jobrundomain.JobRun{ID: "run-upstream", JobID: "upstream", ScheduledAt: now, Status: jobrundomain.StatusFailed, Attempt: 1, CreatedAt: now, UpdatedAt: now})
	_ = runs.Save(ctx, jobrundomain.JobRun{ID: "run-downstream", JobID: "downstream", ScheduledAt: now, Status: jobrundomain.StatusBlocked, Attempt: 1, CreatedAt: now, UpdatedAt: now})

	l := NewDependency(deps, runs, q)
	l.tick(ctx)

	run, err := runs.Get(ctx, "run-downstream")
	if err != nil {
		t.Fatalf("runs.Get() error = %v", err)
	}
	if run.Status != jobrundomain.StatusBlocked {
		t.Fatalf("downstream status = %s, want %s", run.Status, jobrundomain.StatusBlocked)
	}
	if _, err := q.Dequeue(ctx); err == nil {
		t.Fatalf("queue should be empty")
	}
}

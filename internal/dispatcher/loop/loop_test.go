package loop

import (
	"context"
	"errors"
	"testing"
	"time"

	dispatchercancel "github.com/bobacgo/cron-job/internal/dispatcher/cancel"
	dispatcherlease "github.com/bobacgo/cron-job/internal/dispatcher/lease"
	"github.com/bobacgo/cron-job/internal/dispatcher/queue"
	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
	"github.com/bobacgo/cron-job/internal/executor"
	dependencyrepo "github.com/bobacgo/cron-job/internal/repository/dependency"
	jobrepo "github.com/bobacgo/cron-job/internal/repository/job"
	jobrunrepo "github.com/bobacgo/cron-job/internal/repository/jobrun"
	logrepo "github.com/bobacgo/cron-job/internal/repository/log"
)

type failOnceExecutor struct {
	called int
}

func (e *failOnceExecutor) Execute(_ context.Context, req executor.Request) (executor.Result, error) {
	e.called++
	if e.called == 1 {
		return executor.Result{Status: jobrundomain.StatusFailed, Message: "boom", StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC()}, errors.New("boom")
	}
	return executor.Result{Status: jobrundomain.StatusSucceeded, Message: "ok", StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC()}, nil
}

func TestDispatcherSchedulesRetryAfterFailure(t *testing.T) {
	ctx := context.Background()
	jobs := jobrepo.NewInMemoryRepository()
	runs := jobrunrepo.NewInMemoryRepository()
	deps := dependencyrepo.NewInMemoryRepository()
	_ = deps
	logs, err := logrepo.NewFileRepository(t.TempDir())
	if err != nil {
		t.Fatalf("new log repo: %v", err)
	}
	q := queue.NewInMemoryQueue()
	leases := dispatcherlease.NewMemoryManager(30 * time.Second)
	cancels := dispatchercancel.NewManager()
	registry := executor.NewRegistry()
	registry.Register("binary", &failOnceExecutor{})

	job := jobdomain.Job{
		ID:      "job-1",
		Name:    "job-1",
		Enabled: true,
		Executor: jobdomain.ExecutorSpec{
			Kind:   jobdomain.ExecutorKindBinary,
			Binary: &jobdomain.BinaryTarget{Command: "/bin/echo", Args: []string{"ok"}},
		},
		RetryPolicy: jobdomain.RetryPolicy{MaxRetries: 1, InitialBackoff: 5 * time.Millisecond, BackoffMultiple: 1},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := jobs.Save(ctx, job); err != nil {
		t.Fatalf("save job: %v", err)
	}
	run := jobrundomain.JobRun{ID: "run-1", JobID: job.ID, ScheduledAt: time.Now().UTC(), Status: jobrundomain.StatusReady, Attempt: 1, TriggerType: "manual", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := runs.Save(ctx, run); err != nil {
		t.Fatalf("save run: %v", err)
	}
	if err := q.Enqueue(ctx, run.ID); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	l := New(jobs, runs, logs, q, leases, cancels, registry)
	l.tick(ctx)

	all, err := runs.List(ctx)
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(all) < 2 {
		t.Fatalf("expected retry run to be created, got %d runs", len(all))
	}
	var retry jobrundomain.JobRun
	found := false
	for _, item := range all {
		if item.ID != run.ID {
			retry = item
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("retry run not found")
	}
	if retry.Attempt != 2 {
		t.Fatalf("retry attempt = %d, want 2", retry.Attempt)
	}
	if retry.TriggerType != "retry" {
		t.Fatalf("retry trigger = %s, want retry", retry.TriggerType)
	}

	time.Sleep(15 * time.Millisecond)
	if _, err := q.Dequeue(ctx); err != nil {
		t.Fatalf("expected retry run in queue: %v", err)
	}
}

func TestDispatcherCooperativeCancel(t *testing.T) {
	ctx := context.Background()
	jobs := jobrepo.NewInMemoryRepository()
	runs := jobrunrepo.NewInMemoryRepository()
	logs, err := logrepo.NewFileRepository(t.TempDir())
	if err != nil {
		t.Fatalf("new log repo: %v", err)
	}
	q := queue.NewInMemoryQueue()
	leases := dispatcherlease.NewMemoryManager(30 * time.Second)
	cancels := dispatchercancel.NewManager()
	registry := executor.NewRegistry()
	registry.Register("binary", &sleepExecutor{wait: 80 * time.Millisecond})

	job := jobdomain.Job{ID: "job-2", Name: "job-2", Enabled: true, Executor: jobdomain.ExecutorSpec{Kind: jobdomain.ExecutorKindBinary, Binary: &jobdomain.BinaryTarget{Command: "/bin/echo", Args: []string{"ok"}}}, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	_ = jobs.Save(ctx, job)
	run := jobrundomain.JobRun{ID: "run-2", JobID: job.ID, Status: jobrundomain.StatusReady, Attempt: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	_ = runs.Save(ctx, run)
	_ = q.Enqueue(ctx, run.ID)

	l := New(jobs, runs, logs, q, leases, cancels, registry)
	done := make(chan struct{})
	go func() {
		l.tick(ctx)
		close(done)
	}()
	time.Sleep(15 * time.Millisecond)
	if !cancels.Cancel(run.ID) {
		t.Fatalf("expected running cancel to be registered")
	}
	<-done

	updated, err := runs.Get(ctx, run.ID)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	if updated.Status != jobrundomain.StatusCanceled {
		t.Fatalf("run status = %s, want %s", updated.Status, jobrundomain.StatusCanceled)
	}
}

type sleepExecutor struct {
	wait time.Duration
}

func (e *sleepExecutor) Execute(ctx context.Context, req executor.Request) (executor.Result, error) {
	select {
	case <-ctx.Done():
		return executor.Result{Status: jobrundomain.StatusFailed, Message: ctx.Err().Error(), StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC()}, ctx.Err()
	case <-time.After(e.wait):
		return executor.Result{Status: jobrundomain.StatusSucceeded, Message: "ok", StartedAt: time.Now().UTC(), FinishedAt: time.Now().UTC()}, nil
	}
}

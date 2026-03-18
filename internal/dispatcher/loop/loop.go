package loop

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"math"
	"time"

	dispatchercancel "github.com/bobacgo/cron-job/internal/dispatcher/cancel"
	dispatcherlease "github.com/bobacgo/cron-job/internal/dispatcher/lease"
	"github.com/bobacgo/cron-job/internal/dispatcher/queue"
	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
	runlog "github.com/bobacgo/cron-job/internal/domain/log"
	"github.com/bobacgo/cron-job/internal/executor"
	jobrepo "github.com/bobacgo/cron-job/internal/repository/job"
	jobrunrepo "github.com/bobacgo/cron-job/internal/repository/jobrun"
	logrepo "github.com/bobacgo/cron-job/internal/repository/log"
)

type Loop struct {
	jobs      jobrepo.Repository
	runs      jobrunrepo.Repository
	logs      logrepo.Repository
	queue     queue.Queue
	leases    dispatcherlease.Manager
	cancels   *dispatchercancel.Manager
	executors *executor.Registry
}

func New(jobs jobrepo.Repository, runs jobrunrepo.Repository, logs logrepo.Repository, queue queue.Queue, leases dispatcherlease.Manager, cancels *dispatchercancel.Manager, executors *executor.Registry) *Loop {
	if cancels == nil {
		cancels = dispatchercancel.NewManager()
	}
	return &Loop{jobs: jobs, runs: runs, logs: logs, queue: queue, leases: leases, cancels: cancels, executors: executors}
}

func (l *Loop) Start(ctx context.Context, interval time.Duration) {
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

func (l *Loop) tick(ctx context.Context) {
	runID, err := l.queue.Dequeue(ctx)
	if err != nil {
		return
	}
	if err := l.leases.Acquire(ctx, runID, "server"); err != nil {
		return
	}
	defer l.leases.Release(ctx, runID)

	run, err := l.runs.Get(ctx, runID)
	if err != nil {
		log.Printf("dispatcher get run %s: %v", runID, err)
		return
	}
	if run.Status != jobrundomain.StatusReady {
		return
	}
	job, err := l.jobs.Get(ctx, run.JobID)
	if err != nil {
		log.Printf("dispatcher get job %s: %v", run.JobID, err)
		return
	}

	execName, ok := executorName(job)
	if !ok {
		run.Status = jobrundomain.StatusFailed
		run.Message = "unsupported executor type"
		run.FinishedAt = time.Now().UTC()
		run.UpdatedAt = time.Now().UTC()
		_ = l.logs.Append(ctx, runlog.LogRecord{RunID: run.ID, Stream: "stderr", Content: run.Message, OccurredAt: run.UpdatedAt})
		_ = l.runs.Save(ctx, run)
		return
	}

	execImpl, err := l.executors.Get(execName)
	if err != nil {
		log.Printf("dispatcher lookup executor %s: %v", execName, err)
		return
	}

	run.Status = jobrundomain.StatusRunning
	run.StartedAt = time.Now().UTC()
	run.UpdatedAt = run.StartedAt
	if err := l.runs.Save(ctx, run); err != nil {
		log.Printf("dispatcher update running %s: %v", run.ID, err)
		return
	}

	runCtx, cancel := context.WithCancel(ctx)
	l.cancels.Register(run.ID, cancel)
	defer func() {
		l.cancels.Remove(run.ID)
		cancel()
	}()

	result, err := execImpl.Execute(runCtx, executor.Request{Job: job, Run: run})
	if runCtx.Err() == context.Canceled {
		run.Status = jobrundomain.StatusCanceled
		run.Message = "canceled while running"
		run.FinishedAt = time.Now().UTC()
		run.UpdatedAt = run.FinishedAt
		_ = l.logs.Append(ctx, runlog.LogRecord{RunID: run.ID, Stream: "stderr", Content: run.Message, OccurredAt: run.UpdatedAt})
		_ = l.runs.Save(ctx, run)
		return
	}
	if err != nil {
		run.Status = jobrundomain.StatusFailed
		run.Message = err.Error()
		run.FinishedAt = time.Now().UTC()
		run.UpdatedAt = run.FinishedAt
		_ = l.logs.Append(ctx, runlog.LogRecord{RunID: run.ID, Stream: "stderr", Content: run.Message, OccurredAt: run.UpdatedAt})
		_ = l.runs.Save(ctx, run)
		l.scheduleRetry(ctx, job, run)
		return
	}

	run.Status = result.Status
	run.Message = result.Message
	run.StartedAt = result.StartedAt
	run.FinishedAt = result.FinishedAt
	run.UpdatedAt = result.FinishedAt
	if result.Output != "" {
		stream := "stdout"
		if result.Status != jobrundomain.StatusSucceeded {
			stream = "stderr"
		}
		_ = l.logs.Append(ctx, runlog.LogRecord{RunID: run.ID, Stream: stream, Content: result.Output, OccurredAt: result.FinishedAt})
	}
	if err := l.runs.Save(ctx, run); err != nil {
		log.Printf("dispatcher finish run %s: %v", run.ID, err)
	}
	l.scheduleRetry(ctx, job, run)

	if run.Status == jobrundomain.StatusSucceeded {
		job.LastSuccessAt = run.FinishedAt
		job.UpdatedAt = run.FinishedAt
		_ = l.jobs.Save(ctx, job)
	}
}

func (l *Loop) scheduleRetry(ctx context.Context, job jobdomain.Job, run jobrundomain.JobRun) {
	if run.Status != jobrundomain.StatusFailed && run.Status != jobrundomain.StatusTimedOut {
		return
	}
	if job.RetryPolicy.MaxRetries <= 0 {
		return
	}
	currentAttempt := max(run.Attempt, 1)
	if currentAttempt > job.RetryPolicy.MaxRetries {
		return
	}
	delay := retryBackoff(job.RetryPolicy, currentAttempt)
	retryRun := jobrundomain.JobRun{
		ID:          newID(),
		JobID:       run.JobID,
		ScheduledAt: time.Now().UTC().Add(delay),
		Status:      jobrundomain.StatusReady,
		Attempt:     currentAttempt + 1,
		TriggerType: "retry",
		Message:     "scheduled automatic retry",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := l.runs.Save(ctx, retryRun); err != nil {
		log.Printf("dispatcher save retry run %s: %v", retryRun.ID, err)
		return
	}
	_ = l.logs.Append(ctx, runlog.LogRecord{RunID: retryRun.ID, Stream: "stdout", Content: "automatic retry scheduled", OccurredAt: retryRun.CreatedAt})
	go func(runID string, wait time.Duration) {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if err := l.queue.Enqueue(context.Background(), runID); err != nil {
				log.Printf("dispatcher enqueue retry run %s: %v", runID, err)
			}
		}
	}(retryRun.ID, delay)
}

func retryBackoff(policy jobdomain.RetryPolicy, attempt int) time.Duration {
	base := policy.InitialBackoff
	if base <= 0 {
		base = time.Second
	}
	multi := policy.BackoffMultiple
	if multi < 1 {
		multi = 2
	}
	pow := math.Pow(multi, float64(max(attempt-1, 0)))
	next := time.Duration(float64(base) * pow)
	if policy.MaxBackoff > 0 && next > policy.MaxBackoff {
		return policy.MaxBackoff
	}
	return next
}

func newID() string {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func executorName(job jobdomain.Job) (string, bool) {
	switch job.Executor.Kind {
	case jobdomain.ExecutorKindSDK:
		if job.Executor.SDK != nil {
			switch job.Executor.SDK.Protocol {
			case "", "http":
				return "sdk-http", true
			case "grpc":
				return "sdk-grpc", true
			}
		}
	case jobdomain.ExecutorKindBinary:
		return "binary", true
	case jobdomain.ExecutorKindShell:
		return "shell", true
	}
	return "", false
}

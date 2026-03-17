package loop

import (
	"context"
	"log"
	"time"

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
	executors *executor.Registry
}

func New(jobs jobrepo.Repository, runs jobrunrepo.Repository, logs logrepo.Repository, queue queue.Queue, leases dispatcherlease.Manager, executors *executor.Registry) *Loop {
	return &Loop{jobs: jobs, runs: runs, logs: logs, queue: queue, leases: leases, executors: executors}
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

	result, err := execImpl.Execute(ctx, executor.Request{Job: job, Run: run})
	if err != nil {
		run.Status = jobrundomain.StatusFailed
		run.Message = err.Error()
		run.FinishedAt = time.Now().UTC()
		run.UpdatedAt = run.FinishedAt
		_ = l.logs.Append(ctx, runlog.LogRecord{RunID: run.ID, Stream: "stderr", Content: run.Message, OccurredAt: run.UpdatedAt})
		_ = l.runs.Save(ctx, run)
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

	if run.Status == jobrundomain.StatusSucceeded {
		job.LastSuccessAt = run.FinishedAt
		job.UpdatedAt = run.FinishedAt
		_ = l.jobs.Save(ctx, job)
	}
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
	}
	return "", false
}

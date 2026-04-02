package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jobapp "github.com/bobacgo/cron-job/internal/app/job"
	"github.com/bobacgo/cron-job/internal/dispatcher/queue"
	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
	runlog "github.com/bobacgo/cron-job/internal/domain/log"
	"github.com/bobacgo/cron-job/internal/repository"
	"github.com/bobacgo/cron-job/internal/scheduler/planner"
	"github.com/bobacgo/cron-job/internal/testkit/repostub"
)

func TestRunLogHandlerSearch(t *testing.T) {
	svc, logs, _, _ := buildHandlerService(t)
	if err := logs.Append(context.Background(), runlog.LogRecord{RunID: "run-a", Stream: "stderr", Content: "timeout while connecting", OccurredAt: time.Now().UTC()}); err != nil {
		t.Fatalf("append: %v", err)
	}
	h := NewRunLogHandler(svc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/logs/search?q=timeout&stream=stderr", nil)
	rr := httptest.NewRecorder()
	h.Search(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var payload struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Count == 0 {
		t.Fatalf("expected search hits")
	}
}

func TestRunLogHandlerCancelAndRetry(t *testing.T) {
	svc, _, jobs, runs := buildHandlerService(t)
	h := NewRunLogHandler(svc)
	ctx := context.Background()

	job := jobdomain.Job{
		ID:      "job-x",
		Name:    "x",
		Enabled: true,
		Executor: jobdomain.ExecutorSpec{
			Kind:   jobdomain.ExecutorKindBinary,
			Binary: &jobdomain.BinaryTarget{Command: "/bin/echo", Args: []string{"ok"}},
		},
		RetryPolicy: jobdomain.RetryPolicy{MaxRetries: 2},
		CreatedAt:   time.Now().UTC().Unix(),
		UpdatedAt:   time.Now().UTC().Unix(),
	}
	if err := jobs.Save(ctx, job); err != nil {
		t.Fatalf("save job: %v", err)
	}
	if err := runs.Save(ctx, jobrundomain.JobRun{ID: "run-cancel", JobID: job.ID, Status: jobrundomain.StatusReady, Attempt: 1, TriggerType: "manual", CreatedAt: time.Now().UTC().Unix(), UpdatedAt: time.Now().UTC().Unix()}); err != nil {
		t.Fatalf("save run-cancel: %v", err)
	}
	if err := runs.Save(ctx, jobrundomain.JobRun{ID: "run-failed", JobID: job.ID, Status: jobrundomain.StatusFailed, Attempt: 1, TriggerType: "manual", CreatedAt: time.Now().UTC().Unix(), UpdatedAt: time.Now().UTC().Unix()}); err != nil {
		t.Fatalf("save run-failed: %v", err)
	}

	rrCancel := httptest.NewRecorder()
	h.Handle(rrCancel, httptest.NewRequest(http.MethodPost, "/api/v1/job-runs/run-cancel/cancel", nil))
	if rrCancel.Code != http.StatusOK {
		t.Fatalf("cancel status = %d, want 200, body=%s", rrCancel.Code, rrCancel.Body.String())
	}

	rrRetry := httptest.NewRecorder()
	h.Handle(rrRetry, httptest.NewRequest(http.MethodPost, "/api/v1/job-runs/run-failed/retry", nil))
	if rrRetry.Code != http.StatusCreated {
		t.Fatalf("retry status = %d, want 201, body=%s", rrRetry.Code, rrRetry.Body.String())
	}
}

func buildHandlerService(t *testing.T) (*jobapp.Service, *repository.FileLogRepository, *repostub.JobRepo, *repostub.JobRunRepo) {
	t.Helper()
	jobs := repostub.NewJobRepo()
	runs := repostub.NewJobRunRepo()
	deps := repostub.NewDependencyRepo()
	logs, err := repository.NewFileLogRepository(t.TempDir())
	if err != nil {
		t.Fatalf("new log repo: %v", err)
	}
	repo := &repository.Repo{Job: jobs, JobRun: runs, Dependencies: deps, Log: logs}
	svc := jobapp.NewService(repo, queue.NewInMemoryQueue(), planner.New())
	return svc, logs, jobs, runs
}

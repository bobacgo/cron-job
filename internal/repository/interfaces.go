package repository

import (
	"context"
	"time"

	dependencydomain "github.com/bobacgo/cron-job/internal/domain/dependency"
	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
	runlog "github.com/bobacgo/cron-job/internal/domain/log"
)

type JobRepository interface {
	Save(ctx context.Context, job jobdomain.Job) error
	Get(ctx context.Context, id string) (jobdomain.Job, error)
	List(ctx context.Context) ([]jobdomain.Job, error)
	ListEnabled(ctx context.Context) ([]jobdomain.Job, error)
}

type JobRunRepository interface {
	Save(ctx context.Context, run jobrundomain.JobRun) error
	Get(ctx context.Context, id string) (jobrundomain.JobRun, error)
	List(ctx context.Context) ([]jobrundomain.JobRun, error)
	ListByJob(ctx context.Context, jobID string) ([]jobrundomain.JobRun, error)
	ListByStatus(ctx context.Context, status jobrundomain.Status) ([]jobrundomain.JobRun, error)
	FindByDedupKey(ctx context.Context, key string) (jobrundomain.JobRun, bool, error)
}

type DependencyRepository interface {
	Replace(ctx context.Context, jobID string, edges []dependencydomain.Edge) error
	ListByJob(ctx context.Context, jobID string) ([]dependencydomain.Edge, error)
	ListAll(ctx context.Context) ([]dependencydomain.Edge, error)
}

type LogQuery struct {
	RunID    string
	Stream   string
	Contains string
	Limit    int
}

type LogSearchItem struct {
	RunID      string    `json:"run_id"`
	Stream     string    `json:"stream"`
	Content    string    `json:"content"`
	OccurredAt time.Time `json:"occurred_at"`
}

type LogRepository interface {
	Append(ctx context.Context, record runlog.LogRecord) error
	Read(ctx context.Context, runID string) (string, error)
	ReadStream(ctx context.Context, runID, stream string) (string, error)
	Search(ctx context.Context, query LogQuery) ([]LogSearchItem, error)
}

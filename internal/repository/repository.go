package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/bobacgo/cron-job/internal/config"
	dependencydomain "github.com/bobacgo/cron-job/internal/domain/dependency"
	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
	runlog "github.com/bobacgo/cron-job/internal/domain/log"
)

type Repo struct {
	Job          JobRepository
	JobRun       JobRunRepository
	Dependencies DependencyRepository
	Log          LogRepository
}

func NewRepo(cfg *config.Config, db *sql.DB) *Repo {
	logRepo, err := NewFileLogRepository(cfg.LogDir)
	if err != nil {
		panic(err)
	}
	return &Repo{
		Job:          &jobRepo{db: db},
		JobRun:       &jobRunRepo{db: db},
		Dependencies: &dependencyRepo{db: db},
		Log:          logRepo,
	}
}

// Job 存储相关
type JobRepository interface {
	Save(ctx context.Context, job jobdomain.Job) error
	Get(ctx context.Context, id string) (jobdomain.Job, error)
	List(ctx context.Context) ([]jobdomain.Job, error)
	ListEnabled(ctx context.Context) ([]jobdomain.Job, error)
}

// JobRun 存储相关
type JobRunRepository interface {
	Save(ctx context.Context, run jobrundomain.JobRun) error
	Get(ctx context.Context, id string) (jobrundomain.JobRun, error)
	List(ctx context.Context) ([]jobrundomain.JobRun, error)
	ListByJob(ctx context.Context, jobID string) ([]jobrundomain.JobRun, error)
	ListByStatus(ctx context.Context, status jobrundomain.Status) ([]jobrundomain.JobRun, error)
	FindByDedupKey(ctx context.Context, key string) (jobrundomain.JobRun, bool, error)
}

// Dependency 存储相关
type DependencyRepository interface {
	// Replace 会删除 jobID 相关的所有依赖关系，并替换为 edges 中的依赖关系
	Replace(ctx context.Context, jobID string, edges []dependencydomain.Edge) error
	// ListByJob 会返回 jobID 相关的所有依赖关系，即 edges 中的 JobID 都是 jobID
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

// Log 存储相关
type LogRepository interface {
	Append(ctx context.Context, record runlog.LogRecord) error
	Read(ctx context.Context, runID string) (string, error)
	ReadStream(ctx context.Context, runID, stream string) (string, error)
	Search(ctx context.Context, query LogQuery) ([]LogSearchItem, error)
}

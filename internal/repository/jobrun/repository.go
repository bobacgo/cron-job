package jobrun

import (
	"context"

	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
)

type Repository interface {
	Save(ctx context.Context, run jobrundomain.JobRun) error
	Get(ctx context.Context, id string) (jobrundomain.JobRun, error)
	List(ctx context.Context) ([]jobrundomain.JobRun, error)
	ListByJob(ctx context.Context, jobID string) ([]jobrundomain.JobRun, error)
	ListByStatus(ctx context.Context, status jobrundomain.Status) ([]jobrundomain.JobRun, error)
	FindByDedupKey(ctx context.Context, key string) (jobrundomain.JobRun, bool, error)
}

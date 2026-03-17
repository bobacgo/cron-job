package job

import (
	"context"

	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
)

type Repository interface {
	Save(ctx context.Context, job jobdomain.Job) error
	Get(ctx context.Context, id string) (jobdomain.Job, error)
	List(ctx context.Context) ([]jobdomain.Job, error)
	ListEnabled(ctx context.Context) ([]jobdomain.Job, error)
}

package log

import (
	"context"

	runlog "github.com/bobacgo/cron-job/internal/domain/log"
)

type Repository interface {
	Append(ctx context.Context, record runlog.LogRecord) error
	Read(ctx context.Context, runID string) (string, error)
}

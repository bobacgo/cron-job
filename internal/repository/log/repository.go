package log

import (
	"context"
	"time"

	runlog "github.com/bobacgo/cron-job/internal/domain/log"
)

type Query struct {
	RunID    string
	Stream   string
	Contains string
	Limit    int
}

type SearchItem struct {
	RunID      string    `json:"run_id"`
	Stream     string    `json:"stream"`
	Content    string    `json:"content"`
	OccurredAt time.Time `json:"occurred_at"`
}

type Repository interface {
	Append(ctx context.Context, record runlog.LogRecord) error
	Read(ctx context.Context, runID string) (string, error)
	ReadStream(ctx context.Context, runID, stream string) (string, error)
	Search(ctx context.Context, query Query) ([]SearchItem, error)
}

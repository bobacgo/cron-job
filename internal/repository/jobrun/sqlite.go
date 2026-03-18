package jobrun

import (
	"context"
	"database/sql"
	"time"

	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) Save(ctx context.Context, run jobrundomain.JobRun) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO job_runs (
	id, job_id, scheduled_at, started_at, finished_at, status, attempt, trigger_type, message, created_at, updated_at, dedup_key
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	job_id=excluded.job_id,
	scheduled_at=excluded.scheduled_at,
	started_at=excluded.started_at,
	finished_at=excluded.finished_at,
	status=excluded.status,
	attempt=excluded.attempt,
	trigger_type=excluded.trigger_type,
	message=excluded.message,
	updated_at=excluded.updated_at,
	dedup_key=excluded.dedup_key
`,
		run.ID,
		run.JobID,
		formatTime(run.ScheduledAt),
		formatTime(run.StartedAt),
		formatTime(run.FinishedAt),
		string(run.Status),
		run.Attempt,
		run.TriggerType,
		run.Message,
		formatTime(run.CreatedAt),
		formatTime(run.UpdatedAt),
		run.DedupKey(),
	)
	return err
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (jobrundomain.JobRun, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, job_id, scheduled_at, started_at, finished_at, status, attempt, trigger_type, message, created_at, updated_at
FROM job_runs WHERE id = ?
`, id)
	item, err := scanRun(row.Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return jobrundomain.JobRun{}, ErrNotFound
		}
		return jobrundomain.JobRun{}, err
	}
	return item, nil
}

func (r *SQLiteRepository) List(ctx context.Context) ([]jobrundomain.JobRun, error) {
	return r.listQuery(ctx, `
SELECT id, job_id, scheduled_at, started_at, finished_at, status, attempt, trigger_type, message, created_at, updated_at
FROM job_runs ORDER BY created_at DESC
`)
}

func (r *SQLiteRepository) ListByJob(ctx context.Context, jobID string) ([]jobrundomain.JobRun, error) {
	return r.listQuery(ctx, `
SELECT id, job_id, scheduled_at, started_at, finished_at, status, attempt, trigger_type, message, created_at, updated_at
FROM job_runs WHERE job_id = ? ORDER BY created_at DESC
`, jobID)
}

func (r *SQLiteRepository) ListByStatus(ctx context.Context, status jobrundomain.Status) ([]jobrundomain.JobRun, error) {
	return r.listQuery(ctx, `
SELECT id, job_id, scheduled_at, started_at, finished_at, status, attempt, trigger_type, message, created_at, updated_at
FROM job_runs WHERE status = ? ORDER BY created_at DESC
`, string(status))
}

func (r *SQLiteRepository) FindByDedupKey(ctx context.Context, key string) (jobrundomain.JobRun, bool, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, job_id, scheduled_at, started_at, finished_at, status, attempt, trigger_type, message, created_at, updated_at
FROM job_runs WHERE dedup_key = ? LIMIT 1
`, key)
	item, err := scanRun(row.Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return jobrundomain.JobRun{}, false, nil
		}
		return jobrundomain.JobRun{}, false, err
	}
	return item, true, nil
}

func (r *SQLiteRepository) listQuery(ctx context.Context, query string, args ...any) ([]jobrundomain.JobRun, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]jobrundomain.JobRun, 0)
	for rows.Next() {
		item, err := scanRun(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

type scanner func(dest ...any) error

func scanRun(scan scanner) (jobrundomain.JobRun, error) {
	var (
		id          string
		jobID       string
		scheduledAt string
		startedAt   string
		finishedAt  string
		status      string
		attempt     int
		triggerType string
		message     string
		createdAt   string
		updatedAt   string
	)
	if err := scan(
		&id,
		&jobID,
		&scheduledAt,
		&startedAt,
		&finishedAt,
		&status,
		&attempt,
		&triggerType,
		&message,
		&createdAt,
		&updatedAt,
	); err != nil {
		return jobrundomain.JobRun{}, err
	}
	return jobrundomain.JobRun{
		ID:          id,
		JobID:       jobID,
		ScheduledAt: parseTime(scheduledAt),
		StartedAt:   parseTime(startedAt),
		FinishedAt:  parseTime(finishedAt),
		Status:      jobrundomain.Status(status),
		Attempt:     attempt,
		TriggerType: triggerType,
		Message:     message,
		CreatedAt:   parseTime(createdAt),
		UpdatedAt:   parseTime(updatedAt),
	}, nil
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func parseTime(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}
	return parsed.UTC()
}

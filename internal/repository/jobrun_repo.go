package repository

import (
	"context"
	"database/sql"

	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
)

type jobRunRepo struct{ db *sql.DB }

func (r *jobRunRepo) Save(ctx context.Context, run jobrundomain.JobRun) error {
	_, err := r.db.ExecContext(ctx, `
INSERT INTO job_runs (
	id, job_id, scheduled_at, started_at, finished_at, status, attempt, trigger_type, message, created_at, updated_at, dedup_key
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
	job_id=VALUES(job_id),
	scheduled_at=VALUES(scheduled_at),
	started_at=VALUES(started_at),
	finished_at=VALUES(finished_at),
	status=VALUES(status),
	attempt=VALUES(attempt),
	trigger_type=VALUES(trigger_type),
	message=VALUES(message),
	updated_at=VALUES(updated_at),
	dedup_key=VALUES(dedup_key)
`,
		run.ID, run.JobID,
		formatTime(run.ScheduledAt), formatTime(run.StartedAt), formatTime(run.FinishedAt),
		string(run.Status), run.Attempt, run.TriggerType, run.Message,
		formatTime(run.CreatedAt), formatTime(run.UpdatedAt), run.DedupKey(),
	)
	return err
}

func (r *jobRunRepo) Get(ctx context.Context, id string) (jobrundomain.JobRun, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, job_id, scheduled_at, started_at, finished_at, status, attempt, trigger_type, message, created_at, updated_at
FROM job_runs WHERE id = ?
`, id)
	item, err := r.scanRun(row.Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return jobrundomain.JobRun{}, ErrNotFound
		}
		return jobrundomain.JobRun{}, err
	}
	return *item, nil
}

func (r *jobRunRepo) List(ctx context.Context) ([]jobrundomain.JobRun, error) {
	return r.listQuery(ctx, `
SELECT id, job_id, scheduled_at, started_at, finished_at, status, attempt, trigger_type, message, created_at, updated_at
FROM job_runs ORDER BY created_at DESC
`)
}

func (r *jobRunRepo) ListByJob(ctx context.Context, jobID string) ([]jobrundomain.JobRun, error) {
	return r.listQuery(ctx, `
SELECT id, job_id, scheduled_at, started_at, finished_at, status, attempt, trigger_type, message, created_at, updated_at
FROM job_runs WHERE job_id = ? ORDER BY created_at DESC
`, jobID)
}

func (r *jobRunRepo) ListByStatus(ctx context.Context, status jobrundomain.Status) ([]jobrundomain.JobRun, error) {
	return r.listQuery(ctx, `
SELECT id, job_id, scheduled_at, started_at, finished_at, status, attempt, trigger_type, message, created_at, updated_at
FROM job_runs WHERE status = ? ORDER BY created_at DESC
`, string(status))
}

func (r *jobRunRepo) FindByDedupKey(ctx context.Context, key string) (jobrundomain.JobRun, bool, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT id, job_id, scheduled_at, started_at, finished_at, status, attempt, trigger_type, message, created_at, updated_at
FROM job_runs WHERE dedup_key = ? LIMIT 1
`, key)
	item, err := r.scanRun(row.Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return jobrundomain.JobRun{}, false, nil
		}
		return jobrundomain.JobRun{}, false, err
	}
	return *item, true, nil
}

func (r *jobRunRepo) listQuery(ctx context.Context, query string, args ...any) ([]jobrundomain.JobRun, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]jobrundomain.JobRun, 0)
	for rows.Next() {
		item, err := r.scanRun(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *jobRunRepo) scanRun(scan scanFunc) (*jobrundomain.JobRun, error) {
	row := &jobrundomain.JobRun{}
	err := scan(
		&row.ID,
		&row.JobID,
		&row.ScheduledAt,
		&row.StartedAt,
		&row.FinishedAt,
		&row.Status,
		&row.Attempt,
		&row.TriggerType,
		&row.Message,
		&row.CreatedAt,
		&row.UpdatedAt,
	)
	return row, err
}

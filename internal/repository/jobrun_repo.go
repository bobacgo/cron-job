package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
	"github.com/bobacgo/cron-job/kit/sqlx"
)

type jobRunRepo struct{ db *sqlx.DB }

// jobRunFields 是 job_runs 表的 SELECT 字段列表（不含 dedup_key）
var jobRunFields = []string{
	"id",
	"job_id",
	"scheduled_at",
	"started_at",
	"finished_at",
	"status",
	"attempt",
	"trigger_type",
	"message",
	"created_at",
	"updated_at",
}

var jobRunSelectCols = strings.Join(jobRunFields, ", ")

func (r *jobRunRepo) Save(ctx context.Context, run jobrundomain.JobRun) error {
	err := r.existsByID(ctx, run.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return r.insert(ctx, run)
		}
		return err
	}
	return r.update(ctx, run)
}

func (r *jobRunRepo) existsByID(ctx context.Context, id string) error {
	return r.db.QueryRowContext(ctx, `SELECT 1 FROM job_runs WHERE id = ? LIMIT 1`, id).Scan(new(int))
}

func (r *jobRunRepo) insert(ctx context.Context, run jobrundomain.JobRun) error {
	insertFields := append(append([]string{}, jobRunFields...), "dedup_key")
	cols := strings.Join(insertFields, ", ")
	placeholders := strings.Trim(strings.Repeat("?, ", len(insertFields)), ", ")

	_, err := r.db.ExecContext(ctx,
		fmt.Sprintf(`INSERT INTO job_runs (%s) VALUES (%s)`, cols, placeholders),
		run.ID,
		run.JobID,
		run.ScheduledAt,
		run.StartedAt,
		run.FinishedAt,
		string(run.Status),
		run.Attempt,
		run.TriggerType,
		run.Message,
		run.CreatedAt,
		run.UpdatedAt,
		run.DedupKey(),
	)
	return err
}

func (r *jobRunRepo) update(ctx context.Context, run jobrundomain.JobRun) error {
	updateFields := append(append([]string{}, jobRunFields...), "dedup_key")
	skipUpdate := map[string]bool{"id": true, "created_at": true}
	setClauses := make([]string, 0, len(updateFields))
	for _, field := range updateFields {
		if !skipUpdate[field] {
			setClauses = append(setClauses, field+" = ?")
		}
	}

	_, err := r.db.ExecContext(ctx,
		fmt.Sprintf(`UPDATE job_runs SET %s WHERE id = ?`, strings.Join(setClauses, ", ")),
		run.JobID,
		run.ScheduledAt,
		run.StartedAt,
		run.FinishedAt,
		string(run.Status),
		run.Attempt,
		run.TriggerType,
		run.Message,
		run.UpdatedAt,
		run.DedupKey(),
		run.ID,
	)
	return err
}

func (r *jobRunRepo) Get(ctx context.Context, id string) (jobrundomain.JobRun, error) {
	row := r.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT %s FROM job_runs WHERE id = ?`, jobRunSelectCols), id)
	item, err := r.scanRun(row.Scan)
	if err != nil {
		return jobrundomain.JobRun{}, err
	}
	return *item, nil
}

func (r *jobRunRepo) List(ctx context.Context) ([]jobrundomain.JobRun, error) {
	return r.listQuery(ctx,
		fmt.Sprintf(`SELECT %s FROM job_runs ORDER BY created_at DESC`, jobRunSelectCols))
}

func (r *jobRunRepo) ListByJob(ctx context.Context, jobID string) ([]jobrundomain.JobRun, error) {
	return r.listQuery(ctx,
		fmt.Sprintf(`SELECT %s FROM job_runs WHERE job_id = ? ORDER BY created_at DESC`, jobRunSelectCols),
		jobID)
}

func (r *jobRunRepo) ListByStatus(ctx context.Context, status jobrundomain.Status) ([]jobrundomain.JobRun, error) {
	return r.listQuery(ctx,
		fmt.Sprintf(`SELECT %s FROM job_runs WHERE status = ? ORDER BY created_at DESC`, jobRunSelectCols),
		string(status))
}

func (r *jobRunRepo) FindByDedupKey(ctx context.Context, key string) (jobrundomain.JobRun, bool, error) {
	row := r.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT %s FROM job_runs WHERE dedup_key = ? LIMIT 1`, jobRunSelectCols), key)
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

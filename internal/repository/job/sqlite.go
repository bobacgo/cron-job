package job

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) Save(ctx context.Context, job jobdomain.Job) error {
	binaryArgs, err := json.Marshal(binaryArgs(job))
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
INSERT INTO jobs (
	id, name, description, enabled,
	schedule_cron, schedule_interval_seconds, schedule_time_zone, schedule_starting_deadline_seconds,
	executor_kind, sdk_protocol, sdk_url, sdk_method, sdk_timeout_seconds,
	binary_command, binary_args_json, binary_timeout_seconds,
	retry_max_retries, retry_initial_backoff_seconds, retry_max_backoff_seconds, retry_backoff_multiple,
	concurrency_policy, next_run_at, last_run_at, last_success_at, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	name=excluded.name,
	description=excluded.description,
	enabled=excluded.enabled,
	schedule_cron=excluded.schedule_cron,
	schedule_interval_seconds=excluded.schedule_interval_seconds,
	schedule_time_zone=excluded.schedule_time_zone,
	schedule_starting_deadline_seconds=excluded.schedule_starting_deadline_seconds,
	executor_kind=excluded.executor_kind,
	sdk_protocol=excluded.sdk_protocol,
	sdk_url=excluded.sdk_url,
	sdk_method=excluded.sdk_method,
	sdk_timeout_seconds=excluded.sdk_timeout_seconds,
	binary_command=excluded.binary_command,
	binary_args_json=excluded.binary_args_json,
	binary_timeout_seconds=excluded.binary_timeout_seconds,
	retry_max_retries=excluded.retry_max_retries,
	retry_initial_backoff_seconds=excluded.retry_initial_backoff_seconds,
	retry_max_backoff_seconds=excluded.retry_max_backoff_seconds,
	retry_backoff_multiple=excluded.retry_backoff_multiple,
	concurrency_policy=excluded.concurrency_policy,
	next_run_at=excluded.next_run_at,
	last_run_at=excluded.last_run_at,
	last_success_at=excluded.last_success_at,
	updated_at=excluded.updated_at
`,
		job.ID,
		job.Name,
		job.Description,
		boolToInt(job.Enabled),
		job.Schedule.Cron,
		int64(job.Schedule.Interval.Seconds()),
		job.Schedule.TimeZone,
		job.Schedule.StartingDeadlineSeconds,
		job.Executor.Kind,
		sdkProtocol(job),
		sdkURL(job),
		sdkMethod(job),
		int64(sdkTimeout(job).Seconds()),
		binaryCommand(job),
		string(binaryArgs),
		int64(binaryTimeout(job).Seconds()),
		job.RetryPolicy.MaxRetries,
		int64(job.RetryPolicy.InitialBackoff.Seconds()),
		int64(job.RetryPolicy.MaxBackoff.Seconds()),
		job.RetryPolicy.BackoffMultiple,
		job.ConcurrencyPolicy,
		formatTime(job.NextRunAt),
		formatTime(job.LastRunAt),
		formatTime(job.LastSuccessAt),
		formatTime(job.CreatedAt),
		formatTime(job.UpdatedAt),
	)
	return err
}

func (r *SQLiteRepository) Get(ctx context.Context, id string) (jobdomain.Job, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT
	id, name, description, enabled,
	schedule_cron, schedule_interval_seconds, schedule_time_zone, schedule_starting_deadline_seconds,
	executor_kind, sdk_protocol, sdk_url, sdk_method, sdk_timeout_seconds,
	binary_command, binary_args_json, binary_timeout_seconds,
	retry_max_retries, retry_initial_backoff_seconds, retry_max_backoff_seconds, retry_backoff_multiple,
	concurrency_policy, next_run_at, last_run_at, last_success_at, created_at, updated_at
FROM jobs WHERE id = ?
`, id)
	job, err := scanJob(row.Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return jobdomain.Job{}, ErrNotFound
		}
		return jobdomain.Job{}, err
	}
	return job, nil
}

func (r *SQLiteRepository) List(ctx context.Context) ([]jobdomain.Job, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT
	id, name, description, enabled,
	schedule_cron, schedule_interval_seconds, schedule_time_zone, schedule_starting_deadline_seconds,
	executor_kind, sdk_protocol, sdk_url, sdk_method, sdk_timeout_seconds,
	binary_command, binary_args_json, binary_timeout_seconds,
	retry_max_retries, retry_initial_backoff_seconds, retry_max_backoff_seconds, retry_backoff_multiple,
	concurrency_policy, next_run_at, last_run_at, last_success_at, created_at, updated_at
FROM jobs ORDER BY created_at ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]jobdomain.Job, 0)
	for rows.Next() {
		item, err := scanJob(rows.Scan)
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

func (r *SQLiteRepository) ListEnabled(ctx context.Context) ([]jobdomain.Job, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT
	id, name, description, enabled,
	schedule_cron, schedule_interval_seconds, schedule_time_zone, schedule_starting_deadline_seconds,
	executor_kind, sdk_protocol, sdk_url, sdk_method, sdk_timeout_seconds,
	binary_command, binary_args_json, binary_timeout_seconds,
	retry_max_retries, retry_initial_backoff_seconds, retry_max_backoff_seconds, retry_backoff_multiple,
	concurrency_policy, next_run_at, last_run_at, last_success_at, created_at, updated_at
FROM jobs WHERE enabled = 1 ORDER BY created_at ASC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]jobdomain.Job, 0)
	for rows.Next() {
		item, err := scanJob(rows.Scan)
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

func scanJob(scan scanner) (jobdomain.Job, error) {
	var (
		id                              string
		name                            string
		description                     string
		enabled                         int
		scheduleCron                    string
		scheduleIntervalSeconds         int64
		scheduleTimeZone                string
		scheduleStartingDeadlineSeconds int
		executorKind                    string
		sdkProtocol                     string
		sdkURL                          string
		sdkMethod                       string
		sdkTimeoutSeconds               int64
		binaryCommand                   string
		binaryArgsJSON                  string
		binaryTimeoutSeconds            int64
		retryMaxRetries                 int
		retryInitialBackoffSeconds      int64
		retryMaxBackoffSeconds          int64
		retryBackoffMultiple            float64
		concurrencyPolicy               string
		nextRunAt                       string
		lastRunAt                       string
		lastSuccessAt                   string
		createdAt                       string
		updatedAt                       string
	)
	if err := scan(
		&id,
		&name,
		&description,
		&enabled,
		&scheduleCron,
		&scheduleIntervalSeconds,
		&scheduleTimeZone,
		&scheduleStartingDeadlineSeconds,
		&executorKind,
		&sdkProtocol,
		&sdkURL,
		&sdkMethod,
		&sdkTimeoutSeconds,
		&binaryCommand,
		&binaryArgsJSON,
		&binaryTimeoutSeconds,
		&retryMaxRetries,
		&retryInitialBackoffSeconds,
		&retryMaxBackoffSeconds,
		&retryBackoffMultiple,
		&concurrencyPolicy,
		&nextRunAt,
		&lastRunAt,
		&lastSuccessAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return jobdomain.Job{}, err
	}
	var args []string
	if err := json.Unmarshal([]byte(binaryArgsJSON), &args); err != nil {
		args = nil
	}

	job := jobdomain.Job{
		ID:          id,
		Name:        name,
		Description: description,
		Enabled:     enabled == 1,
		Schedule: jobdomain.Schedule{
			Cron:                    scheduleCron,
			Interval:                time.Duration(scheduleIntervalSeconds) * time.Second,
			TimeZone:                scheduleTimeZone,
			StartingDeadlineSeconds: scheduleStartingDeadlineSeconds,
		},
		RetryPolicy: jobdomain.RetryPolicy{
			MaxRetries:      retryMaxRetries,
			InitialBackoff:  time.Duration(retryInitialBackoffSeconds) * time.Second,
			MaxBackoff:      time.Duration(retryMaxBackoffSeconds) * time.Second,
			BackoffMultiple: retryBackoffMultiple,
		},
		ConcurrencyPolicy: jobdomain.ConcurrencyPolicy(concurrencyPolicy),
		NextRunAt:         parseTime(nextRunAt),
		LastRunAt:         parseTime(lastRunAt),
		LastSuccessAt:     parseTime(lastSuccessAt),
		CreatedAt:         parseTime(createdAt),
		UpdatedAt:         parseTime(updatedAt),
	}
	job.Executor.Kind = jobdomain.ExecutorKind(executorKind)
	if job.Executor.Kind == jobdomain.ExecutorKindSDK {
		job.Executor.SDK = &jobdomain.SDKTarget{
			Protocol: sdkProtocol,
			URL:      sdkURL,
			Method:   sdkMethod,
			Timeout:  time.Duration(sdkTimeoutSeconds) * time.Second,
		}
	}
	if job.Executor.Kind == jobdomain.ExecutorKindBinary {
		job.Executor.Binary = &jobdomain.BinaryTarget{
			Command: binaryCommand,
			Args:    args,
			Timeout: time.Duration(binaryTimeoutSeconds) * time.Second,
		}
	}
	return job, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
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

func sdkProtocol(job jobdomain.Job) string {
	if job.Executor.SDK == nil {
		return ""
	}
	return job.Executor.SDK.Protocol
}

func sdkURL(job jobdomain.Job) string {
	if job.Executor.SDK == nil {
		return ""
	}
	return job.Executor.SDK.URL
}

func sdkMethod(job jobdomain.Job) string {
	if job.Executor.SDK == nil {
		return ""
	}
	return job.Executor.SDK.Method
}

func sdkTimeout(job jobdomain.Job) time.Duration {
	if job.Executor.SDK == nil {
		return 0
	}
	return job.Executor.SDK.Timeout
}

func binaryCommand(job jobdomain.Job) string {
	if job.Executor.Binary == nil {
		return ""
	}
	return job.Executor.Binary.Command
}

func binaryArgs(job jobdomain.Job) []string {
	if job.Executor.Binary == nil {
		return nil
	}
	return job.Executor.Binary.Args
}

func binaryTimeout(job jobdomain.Job) time.Duration {
	if job.Executor.Binary == nil {
		return 0
	}
	return job.Executor.Binary.Timeout
}

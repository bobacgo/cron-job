package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
)

type jobRepo struct{ db *sql.DB }

const selectJobColumns = `
	id,
	name,
	description,
	enabled,
	schedule_cron,
	schedule_interval_seconds,
	schedule_time_zone,
	schedule_starting_deadline_seconds,
	executor_kind,
	sdk_protocol,
	sdk_url,
	sdk_method,
	sdk_timeout_seconds,
	binary_command,
	binary_args_json,
	binary_timeout_seconds,
	shell_script,
	shell_shell,
	shell_timeout_seconds,
	retry_max_retries,
	retry_initial_backoff_seconds,
	retry_max_backoff_seconds,
	retry_backoff_multiple,
	concurrency_policy,
	next_run_at,
	last_run_at,
	last_success_at,
	created_at,
	updated_at`

func (r *jobRepo) Save(ctx context.Context, job jobdomain.Job) error {
	binaryArgsJSON, err := json.Marshal(jobBinaryArgs(job))
	if err != nil {
		return err
	}
	_, err = r.db.ExecContext(ctx, `
INSERT INTO jobs (
	id,
	name,
	description,
	enabled,
	schedule_cron,
	schedule_interval_seconds,
	schedule_time_zone,
	schedule_starting_deadline_seconds,
	executor_kind,
	sdk_protocol,
	sdk_url,
	sdk_method,
	sdk_timeout_seconds,
	binary_command,
	binary_args_json,
	binary_timeout_seconds,
	shell_script,
	shell_shell,
	shell_timeout_seconds,
	retry_max_retries,
	retry_initial_backoff_seconds,
	retry_max_backoff_seconds,
	retry_backoff_multiple,
	concurrency_policy,
	next_run_at,
	last_run_at,
	last_success_at,
	created_at,
	updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
	name=VALUES(name),
	description=VALUES(description),
	enabled=VALUES(enabled),
	schedule_cron=VALUES(schedule_cron),
	schedule_interval_seconds=VALUES(schedule_interval_seconds),
	schedule_time_zone=VALUES(schedule_time_zone),
	schedule_starting_deadline_seconds=VALUES(schedule_starting_deadline_seconds),
	executor_kind=VALUES(executor_kind),
	sdk_protocol=VALUES(sdk_protocol),
	sdk_url=VALUES(sdk_url),
	sdk_method=VALUES(sdk_method),
	sdk_timeout_seconds=VALUES(sdk_timeout_seconds),
	binary_command=VALUES(binary_command),
	binary_args_json=VALUES(binary_args_json),
	binary_timeout_seconds=VALUES(binary_timeout_seconds),
	shell_script=VALUES(shell_script),
	shell_shell=VALUES(shell_shell),
	shell_timeout_seconds=VALUES(shell_timeout_seconds),
	retry_max_retries=VALUES(retry_max_retries),
	retry_initial_backoff_seconds=VALUES(retry_initial_backoff_seconds),
	retry_max_backoff_seconds=VALUES(retry_max_backoff_seconds),
	retry_backoff_multiple=VALUES(retry_backoff_multiple),
	concurrency_policy=VALUES(concurrency_policy),
	next_run_at=VALUES(next_run_at),
	last_run_at=VALUES(last_run_at),
	last_success_at=VALUES(last_success_at),
	updated_at=VALUES(updated_at)
`,
		job.ID, job.Name, job.Description, boolToInt(job.Enabled),
		job.Schedule.Cron, int64(job.Schedule.Interval.Seconds()), job.Schedule.TimeZone, job.Schedule.StartingDeadlineSeconds,
		job.Executor.Kind, jobSDKProtocol(job), jobSDKURL(job), jobSDKMethod(job), int64(jobSDKTimeout(job).Seconds()),
		jobBinaryCommand(job), string(binaryArgsJSON), int64(jobBinaryTimeout(job).Seconds()),
		jobShellScript(job), jobShellShell(job), int64(jobShellTimeout(job).Seconds()),
		job.RetryPolicy.MaxRetries, int64(job.RetryPolicy.InitialBackoff.Seconds()), int64(job.RetryPolicy.MaxBackoff.Seconds()), job.RetryPolicy.BackoffMultiple,
		job.ConcurrencyPolicy, formatTime(job.NextRunAt), formatTime(job.LastRunAt), formatTime(job.LastSuccessAt),
		formatTime(job.CreatedAt), formatTime(job.UpdatedAt),
	)
	return err
}

func (r *jobRepo) Get(ctx context.Context, id string) (jobdomain.Job, error) {
	row := r.db.QueryRowContext(ctx, `SELECT`+selectJobColumns+` FROM jobs WHERE id = ?`, id)
	job, err := scanJob(row.Scan)
	if err != nil {
		if err == sql.ErrNoRows {
			return jobdomain.Job{}, ErrNotFound
		}
		return jobdomain.Job{}, err
	}
	return job, nil
}

func (r *jobRepo) List(ctx context.Context) ([]jobdomain.Job, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT`+selectJobColumns+` FROM jobs ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectJobs(rows)
}

func (r *jobRepo) ListEnabled(ctx context.Context) ([]jobdomain.Job, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT`+selectJobColumns+` FROM jobs WHERE enabled = 1 ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectJobs(rows)
}

func collectJobs(rows *sql.Rows) ([]jobdomain.Job, error) {
	items := make([]jobdomain.Job, 0)
	for rows.Next() {
		item, err := scanJob(rows.Scan)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanJob(scan scanFunc) (jobdomain.Job, error) {
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
		shellScript                     string
		shellShell                      string
		shellTimeoutSeconds             int64
		retryMaxRetries                 int
		retryInitialBackoffSeconds      int64
		retryMaxBackoffSeconds          int64
		retryBackoffMultiple            float64
		concurrencyPolicy               string
		nextRunAt                       int64
		lastRunAt                       int64
		lastSuccessAt                   int64
		createdAt                       int64
		updatedAt                       int64
	)
	if err := scan(
		&id, &name, &description, &enabled,
		&scheduleCron, &scheduleIntervalSeconds, &scheduleTimeZone, &scheduleStartingDeadlineSeconds,
		&executorKind, &sdkProtocol, &sdkURL, &sdkMethod, &sdkTimeoutSeconds,
		&binaryCommand, &binaryArgsJSON, &binaryTimeoutSeconds,
		&shellScript, &shellShell, &shellTimeoutSeconds,
		&retryMaxRetries, &retryInitialBackoffSeconds, &retryMaxBackoffSeconds, &retryBackoffMultiple,
		&concurrencyPolicy, &nextRunAt, &lastRunAt, &lastSuccessAt, &createdAt, &updatedAt,
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
	if job.Executor.Kind == jobdomain.ExecutorKindShell {
		job.Executor.Shell = &jobdomain.ShellTarget{
			Script:  shellScript,
			Shell:   shellShell,
			Timeout: time.Duration(shellTimeoutSeconds) * time.Second,
		}
	}
	return job, nil
}

func jobSDKProtocol(job jobdomain.Job) string {
	if job.Executor.SDK == nil {
		return ""
	}
	return job.Executor.SDK.Protocol
}

func jobSDKURL(job jobdomain.Job) string {
	if job.Executor.SDK == nil {
		return ""
	}
	return job.Executor.SDK.URL
}

func jobSDKMethod(job jobdomain.Job) string {
	if job.Executor.SDK == nil {
		return ""
	}
	return job.Executor.SDK.Method
}

func jobSDKTimeout(job jobdomain.Job) time.Duration {
	if job.Executor.SDK == nil {
		return 0
	}
	return job.Executor.SDK.Timeout
}

func jobBinaryCommand(job jobdomain.Job) string {
	if job.Executor.Binary == nil {
		return ""
	}
	return job.Executor.Binary.Command
}

func jobBinaryArgs(job jobdomain.Job) []string {
	if job.Executor.Binary == nil {
		return nil
	}
	return job.Executor.Binary.Args
}

func jobBinaryTimeout(job jobdomain.Job) time.Duration {
	if job.Executor.Binary == nil {
		return 0
	}
	return job.Executor.Binary.Timeout
}

func jobShellScript(job jobdomain.Job) string {
	if job.Executor.Shell == nil {
		return ""
	}
	return job.Executor.Shell.Script
}

func jobShellShell(job jobdomain.Job) string {
	if job.Executor.Shell == nil {
		return ""
	}
	return job.Executor.Shell.Shell
}

func jobShellTimeout(job jobdomain.Job) time.Duration {
	if job.Executor.Shell == nil {
		return 0
	}
	return job.Executor.Shell.Timeout
}

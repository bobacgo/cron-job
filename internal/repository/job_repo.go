package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
	"github.com/bobacgo/cron-job/kit/sqlx"
)

type jobRepo struct{ db *sqlx.DB }

// jobFields 是 jobs 表的字段列表
var jobFields = []string{
	"id",
	"name",
	"description",
	"enabled",
	"schedule_cron",
	"schedule_interval_seconds",
	"schedule_time_zone",
	"schedule_starting_deadline_seconds",
	"executor_kind",
	"sdk_protocol",
	"sdk_url",
	"sdk_method",
	"sdk_timeout_seconds",
	"binary_command",
	"binary_args_json",
	"binary_timeout_seconds",
	"shell_script",
	"shell_shell",
	"shell_timeout_seconds",
	"retry_max_retries",
	"retry_initial_backoff_seconds",
	"retry_max_backoff_seconds",
	"retry_backoff_multiple",
	"concurrency_policy",
	"next_run_at",
	"last_run_at",
	"last_success_at",
	"created_at",
	"updated_at",
}

var jobSelectCols = strings.Join(jobFields, ", ")

func (r *jobRepo) Save(ctx context.Context, job jobdomain.Job) error {
	binaryArgsJSON, err := json.Marshal(jobBinaryArgs(job))
	if err != nil {
		return err
	}
	values := buildJobFieldValues(job, string(binaryArgsJSON))

	err = r.existsByID(ctx, job.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return r.insert(ctx, values)
		}
		return err
	}
	return r.update(ctx, values)
}

func (r *jobRepo) existsByID(ctx context.Context, id string) error {
	return r.db.QueryRowContext(ctx, `SELECT 1 FROM jobs WHERE id = ? LIMIT 1`, id).Scan(new(int))
}

func (r *jobRepo) insert(ctx context.Context, values map[string]any) error {
	insertFields := append([]string{}, jobFields...)
	cols := strings.Join(insertFields, ", ")
	placeholders := strings.Trim(strings.Repeat("?, ", len(insertFields)), ", ")
	args := make([]any, 0, len(insertFields))
	for _, field := range insertFields {
		args = append(args, values[field])
	}

	_, err := r.db.ExecContext(ctx,
		fmt.Sprintf(`INSERT INTO jobs (%s) VALUES (%s)`, cols, placeholders),
		args...,
	)
	return err
}

func (r *jobRepo) update(ctx context.Context, values map[string]any) error {
	updateFields := append([]string{}, jobFields...)
	skipUpdate := map[string]bool{"id": true, "created_at": true}
	setClauses := make([]string, 0, len(updateFields))
	args := make([]any, 0, len(updateFields))
	for _, field := range updateFields {
		if skipUpdate[field] {
			continue
		}
		setClauses = append(setClauses, field+" = ?")
		args = append(args, values[field])
	}
	args = append(args, values["id"])

	_, err := r.db.ExecContext(ctx,
		fmt.Sprintf(`UPDATE jobs SET %s WHERE id = ?`, strings.Join(setClauses, ", ")),
		args...,
	)
	return err
}

func buildJobFieldValues(job jobdomain.Job, binaryArgsJSON string) map[string]any {
	return map[string]any{
		"id":                                 job.ID,
		"name":                               job.Name,
		"description":                        job.Description,
		"enabled":                            boolToInt(job.Enabled),
		"schedule_cron":                      job.Schedule.Cron,
		"schedule_interval_seconds":          int64(job.Schedule.Interval.Seconds()),
		"schedule_time_zone":                 job.Schedule.TimeZone,
		"schedule_starting_deadline_seconds": job.Schedule.StartingDeadlineSeconds,
		"executor_kind":                      job.Executor.Kind,
		"sdk_protocol":                       jobSDKProtocol(job),
		"sdk_url":                            jobSDKURL(job),
		"sdk_method":                         jobSDKMethod(job),
		"sdk_timeout_seconds":                int64(jobSDKTimeout(job).Seconds()),
		"binary_command":                     jobBinaryCommand(job),
		"binary_args_json":                   binaryArgsJSON,
		"binary_timeout_seconds":             int64(jobBinaryTimeout(job).Seconds()),
		"shell_script":                       jobShellScript(job),
		"shell_shell":                        jobShellShell(job),
		"shell_timeout_seconds":              int64(jobShellTimeout(job).Seconds()),
		"retry_max_retries":                  job.RetryPolicy.MaxRetries,
		"retry_initial_backoff_seconds":      int64(job.RetryPolicy.InitialBackoff.Seconds()),
		"retry_max_backoff_seconds":          int64(job.RetryPolicy.MaxBackoff.Seconds()),
		"retry_backoff_multiple":             job.RetryPolicy.BackoffMultiple,
		"concurrency_policy":                 job.ConcurrencyPolicy,
		"next_run_at":                        job.NextRunAt,
		"last_run_at":                        job.LastRunAt,
		"last_success_at":                    job.LastSuccessAt,
		"created_at":                         job.CreatedAt,
		"updated_at":                         job.UpdatedAt,
	}
}

func (r *jobRepo) Get(ctx context.Context, id string) (jobdomain.Job, error) {
	row := r.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT %s FROM jobs WHERE id = ?`, jobSelectCols), id)
	job, err := scanJob(row.Scan)
	if err != nil {
		return jobdomain.Job{}, err
	}
	return job, nil
}

func (r *jobRepo) List(ctx context.Context) ([]jobdomain.Job, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`SELECT %s FROM jobs ORDER BY created_at ASC`, jobSelectCols))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectJobs(rows)
}

func (r *jobRepo) ListEnabled(ctx context.Context) ([]jobdomain.Job, error) {
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf(`SELECT %s FROM jobs WHERE enabled = 1 ORDER BY created_at ASC`, jobSelectCols))
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
		NextRunAt:         nextRunAt,
		LastRunAt:         lastRunAt,
		LastSuccessAt:     lastSuccessAt,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
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

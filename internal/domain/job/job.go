package job

import "time"

type ExecutorKind string

const (
	ExecutorKindSDK    ExecutorKind = "sdk"
	ExecutorKindBinary ExecutorKind = "binary"
	ExecutorKindShell  ExecutorKind = "shell"
)

type ConcurrencyPolicy string

const (
	ConcurrencyAllow   ConcurrencyPolicy = "Allow"
	ConcurrencyForbid  ConcurrencyPolicy = "Forbid"
	ConcurrencyReplace ConcurrencyPolicy = "Replace"
)

type Job struct {
	ID                string
	Name              string
	Description       string
	Enabled           bool
	Schedule          Schedule
	Executor          ExecutorSpec
	RetryPolicy       RetryPolicy
	ConcurrencyPolicy ConcurrencyPolicy
	NextRunAt         time.Time
	LastRunAt         time.Time
	LastSuccessAt     time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Schedule struct {
	Cron                    string
	Interval                time.Duration
	TimeZone                string
	StartingDeadlineSeconds int
}

type RetryPolicy struct {
	MaxRetries      int
	InitialBackoff  time.Duration
	MaxBackoff      time.Duration
	BackoffMultiple float64
}

type ExecutorSpec struct {
	Kind   ExecutorKind
	SDK    *SDKTarget
	Binary *BinaryTarget
	Shell  *ShellTarget
}

type SDKTarget struct {
	Protocol string
	URL      string
	Method   string
	Timeout  time.Duration
}

type BinaryTarget struct {
	Command string
	Args    []string
	Timeout time.Duration
}

type ShellTarget struct {
	// Script is an inline shell script (passed to Shell via -c).
	Script string
	// Shell is the interpreter to use; defaults to /bin/sh.
	Shell   string
	Timeout time.Duration
}

func (j Job) Validate() error {
	if j.Name == "" {
		return ErrInvalidJob("name is required")
	}
	if j.Schedule.Interval <= 0 && j.Schedule.Cron == "" {
		return ErrInvalidJob("interval or cron is required")
	}
	if j.Executor.Kind == ExecutorKindSDK && j.Executor.SDK == nil {
		return ErrInvalidJob("sdk target is required")
	}
	if j.Executor.Kind == ExecutorKindBinary && j.Executor.Binary == nil {
		return ErrInvalidJob("binary target is required")
	}
	if j.Executor.Kind == ExecutorKindShell && (j.Executor.Shell == nil || j.Executor.Shell.Script == "") {
		return ErrInvalidJob("shell script is required")
	}
	return nil
}

type invalidJobError string

func (e invalidJobError) Error() string { return string(e) }

func ErrInvalidJob(message string) error { return invalidJobError(message) }

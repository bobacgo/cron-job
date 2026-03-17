package jobrun

import "time"

type Status string

const (
	StatusPending   Status = "Pending"
	StatusBlocked   Status = "Blocked"
	StatusReady     Status = "Ready"
	StatusRunning   Status = "Running"
	StatusSucceeded Status = "Succeeded"
	StatusFailed    Status = "Failed"
	StatusTimedOut  Status = "TimedOut"
	StatusCanceled  Status = "Canceled"
	StatusSkipped   Status = "Skipped"
)

type JobRun struct {
	ID          string
	JobID       string
	ScheduledAt time.Time
	StartedAt   time.Time
	FinishedAt  time.Time
	Status      Status
	Attempt     int
	TriggerType string
	Message     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (r JobRun) DedupKey() string {
	return r.JobID + "/" + r.ScheduledAt.UTC().Format(time.RFC3339Nano)
}

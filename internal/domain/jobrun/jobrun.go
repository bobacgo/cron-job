package jobrun

import "strconv"

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
	ScheduledAt int64
	StartedAt   int64
	FinishedAt  int64
	Status      Status
	Attempt     int
	TriggerType string
	Message     string
	CreatedAt   int64
	UpdatedAt   int64
}

func (r JobRun) DedupKey() string {
	return r.JobID + "/" + strconv.FormatInt(r.ScheduledAt, 10)
}

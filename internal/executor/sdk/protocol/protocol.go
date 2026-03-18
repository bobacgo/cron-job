package protocol

import (
	"strings"
	"time"

	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
)

type RunRequest struct {
	JobID       string    `json:"job_id"`
	JobName     string    `json:"job_name"`
	RunID       string    `json:"run_id"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Attempt     int       `json:"attempt"`
	TriggerType string    `json:"trigger_type"`
}

type RunResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Output  string `json:"output"`
}

func NormalizeStatus(raw string) jobrundomain.Status {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "ok", "succeeded", "success":
		return jobrundomain.StatusSucceeded
	case "failed", "error":
		return jobrundomain.StatusFailed
	case "timeout", "timedout":
		return jobrundomain.StatusTimedOut
	case "canceled", "cancelled":
		return jobrundomain.StatusCanceled
	default:
		return jobrundomain.StatusSucceeded
	}
}

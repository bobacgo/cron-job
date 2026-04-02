package viewmodel

import (
	"time"

	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
)

type Dashboard struct {
	JobCount      int
	EnabledJobs   int
	DisabledJobs  int
	RunCount      int
	RunningRuns   int
	WaitingRuns   int
	SucceededRuns int
	FailedRuns    int
	EnabledRate   int
	SuccessRate   int
	AttentionRate int
	GeneratedAt   time.Time
}

type JobItem struct {
	ID          string
	Name        string
	Description string
	Enabled     bool
	Executor    string
	Schedule    string
	NextRunAt   string
}

type DependencyOption struct {
	ID   string
	Name string
}

type JobDetail struct {
	ID                string
	Name              string
	Description       string
	Enabled           bool
	Executor          string
	Schedule          string
	NextRunAt         string
	LastSuccessAt     string
	DependencyOptions []DependencyOption
	Runs              []RunItem
}

type RunItem struct {
	ID          string
	LogURL      string
	Status      string
	TriggerType string
	ScheduledAt string
	StartedAt   string
	FinishedAt  string
	Message     string
}

func JobItems(items []jobdomain.Job) []JobItem {
	result := make([]JobItem, 0, len(items))
	for _, item := range items {
		schedule := item.Schedule.Cron
		if schedule == "" {
			schedule = item.Schedule.Interval.String()
		}
		nextRun := "-"
		if item.NextRunAt > 0 {
			nextRun = time.Unix(item.NextRunAt, 0).UTC().Format(time.RFC3339)
		}
		result = append(result, JobItem{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Enabled:     item.Enabled,
			Executor:    string(item.Executor.Kind),
			Schedule:    schedule,
			NextRunAt:   nextRun,
		})
	}
	return result
}

func DependencyOptions(items []jobdomain.Job) []DependencyOption {
	result := make([]DependencyOption, 0, len(items))
	for _, item := range items {
		result = append(result, DependencyOption{ID: item.ID, Name: item.Name})
	}
	return result
}

func NewJobDetail(item jobdomain.Job, deps []jobdomain.Job, runs []jobrundomain.JobRun) JobDetail {
	lastSuccess := "-"
	if item.LastSuccessAt > 0 {
		lastSuccess = time.Unix(item.LastSuccessAt, 0).UTC().Format(time.RFC3339)
	}
	nextRun := "-"
	if item.NextRunAt > 0 {
		nextRun = time.Unix(item.NextRunAt, 0).UTC().Format(time.RFC3339)
	}
	schedule := item.Schedule.Cron
	if schedule == "" {
		schedule = item.Schedule.Interval.String()
	}
	runItems := make([]RunItem, 0, len(runs))
	for _, run := range runs {
		runItems = append(runItems, RunItem{
			ID:          run.ID,
			LogURL:      "/job-runs/" + run.ID + "/logs",
			Status:      string(run.Status),
			TriggerType: run.TriggerType,
			ScheduledAt: formatTime(run.ScheduledAt),
			StartedAt:   formatTime(run.StartedAt),
			FinishedAt:  formatTime(run.FinishedAt),
			Message:     run.Message,
		})
	}
	return JobDetail{
		ID:                item.ID,
		Name:              item.Name,
		Description:       item.Description,
		Enabled:           item.Enabled,
		Executor:          string(item.Executor.Kind),
		Schedule:          schedule,
		NextRunAt:         nextRun,
		LastSuccessAt:     lastSuccess,
		DependencyOptions: DependencyOptions(deps),
		Runs:              runItems,
	}
}

func formatTime(value int64) string {
	if value <= 0 {
		return "-"
	}
	return time.Unix(value, 0).UTC().Format(time.RFC3339)
}

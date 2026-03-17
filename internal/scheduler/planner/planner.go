package planner

import (
	"fmt"
	"time"

	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
	cron "github.com/robfig/cron/v3"
)

type Planner struct {
	parser cron.Parser
}

func New() *Planner {
	return &Planner{
		parser: cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
	}
}

func (p *Planner) Due(job jobdomain.Job, now time.Time) (time.Time, bool, error) {
	if !job.Enabled {
		return time.Time{}, false, nil
	}
	if job.NextRunAt.IsZero() {
		return time.Time{}, false, nil
	}
	if !job.NextRunAt.After(now) {
		return job.NextRunAt, true, nil
	}
	return time.Time{}, false, nil
}

func (p *Planner) Next(job jobdomain.Job, base time.Time) (time.Time, error) {
	if job.Schedule.Cron != "" {
		location, err := p.location(job.Schedule.TimeZone)
		if err != nil {
			return time.Time{}, err
		}
		schedule, err := p.parser.Parse(job.Schedule.Cron)
		if err != nil {
			return time.Time{}, err
		}
		return schedule.Next(base.In(location)).UTC(), nil
	}
	if job.Schedule.Interval > 0 {
		return base.Add(job.Schedule.Interval), nil
	}
	return time.Time{}, fmt.Errorf("job has no schedule")
}

func (p *Planner) location(name string) (*time.Location, error) {
	if name == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q: %w", name, err)
	}
	return loc, nil
}

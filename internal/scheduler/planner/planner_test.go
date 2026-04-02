package planner

import (
	"testing"
	"time"

	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
)

func TestNextInterval(t *testing.T) {
	p := New()
	base := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)
	job := jobdomain.Job{Schedule: jobdomain.Schedule{Interval: 30 * time.Second}}

	next, err := p.Next(job, base)
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	want := base.Add(30 * time.Second)
	if !next.Equal(want) {
		t.Fatalf("Next() = %s, want %s", next, want)
	}
}

func TestNextCronWithTimezone(t *testing.T) {
	p := New()
	base := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)
	job := jobdomain.Job{Schedule: jobdomain.Schedule{Cron: "0 20 * * *", TimeZone: "Asia/Shanghai"}}

	next, err := p.Next(job, base)
	if err != nil {
		t.Fatalf("Next() error = %v", err)
	}
	// 20:00 at UTC+8 is 12:00 UTC.
	want := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("Next() = %s, want %s", next, want)
	}
}

func TestDue(t *testing.T) {
	p := New()
	now := time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC)
	job := jobdomain.Job{Enabled: true, NextRunAt: now.Add(-time.Second).Unix()}

	dueAt, due, err := p.Due(job, now)
	if err != nil {
		t.Fatalf("Due() error = %v", err)
	}
	if !due {
		t.Fatalf("Due() due = false, want true")
	}
	if !dueAt.Equal(time.Unix(job.NextRunAt, 0).UTC()) {
		t.Fatalf("Due() dueAt = %s, want %s", dueAt, time.Unix(job.NextRunAt, 0).UTC())
	}
}

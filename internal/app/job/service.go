package job

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/bobacgo/cron-job/internal/dispatcher/queue"
	dependencydomain "github.com/bobacgo/cron-job/internal/domain/dependency"
	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
	runlog "github.com/bobacgo/cron-job/internal/domain/log"
	dependencyrepo "github.com/bobacgo/cron-job/internal/repository/dependency"
	jobrepo "github.com/bobacgo/cron-job/internal/repository/job"
	jobrunrepo "github.com/bobacgo/cron-job/internal/repository/jobrun"
	logrepo "github.com/bobacgo/cron-job/internal/repository/log"
	"github.com/bobacgo/cron-job/internal/scheduler/planner"
)

type Service struct {
	jobs         jobrepo.Repository
	runs         jobrunrepo.Repository
	dependencies dependencyrepo.Repository
	logs         logrepo.Repository
	queue        queue.Queue
	planner      *planner.Planner
}

type Detail struct {
	Job            jobdomain.Job
	Dependencies   []dependencydomain.Edge
	DependencyJobs []jobdomain.Job
	Runs           []jobrundomain.JobRun
}

func NewService(jobs jobrepo.Repository, runs jobrunrepo.Repository, dependencies dependencyrepo.Repository, logs logrepo.Repository, queue queue.Queue, planner *planner.Planner) *Service {
	return &Service{jobs: jobs, runs: runs, dependencies: dependencies, logs: logs, queue: queue, planner: planner}
}

func (s *Service) Create(ctx context.Context, job jobdomain.Job, dependencyIDs []string) (jobdomain.Job, error) {
	if err := job.Validate(); err != nil {
		return jobdomain.Job{}, err
	}

	now := time.Now().UTC()
	job.ID = newID()
	job.CreatedAt = now
	job.UpdatedAt = now
	nextRunAt, err := s.planner.Next(job, now)
	if err != nil {
		return jobdomain.Job{}, err
	}
	job.NextRunAt = nextRunAt
	if job.ConcurrencyPolicy == "" {
		job.ConcurrencyPolicy = jobdomain.ConcurrencyForbid
	}
	edges, err := s.buildEdges(ctx, job.ID, dependencyIDs)
	if err != nil {
		return jobdomain.Job{}, err
	}

	if err := s.jobs.Save(ctx, job); err != nil {
		return jobdomain.Job{}, err
	}
	if len(edges) > 0 {
		for i := range edges {
			edges[i].JobID = job.ID
		}
		if err := s.dependencies.Replace(ctx, job.ID, edges); err != nil {
			return jobdomain.Job{}, err
		}
	}

	return job, nil
}

func (s *Service) List(ctx context.Context) ([]jobdomain.Job, error) {
	return s.jobs.List(ctx)
}

func (s *Service) Get(ctx context.Context, id string) (jobdomain.Job, error) {
	return s.jobs.Get(ctx, id)
}

func (s *Service) GetDetail(ctx context.Context, id string) (Detail, error) {
	job, err := s.jobs.Get(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	edges, err := s.dependencies.ListByJob(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	runs, err := s.runs.ListByJob(ctx, id)
	if err != nil {
		return Detail{}, err
	}

	depJobs := make([]jobdomain.Job, 0, len(edges))
	for _, edge := range edges {
		depJob, err := s.jobs.Get(ctx, edge.DependsOnJobID)
		if err == nil {
			depJobs = append(depJobs, depJob)
		}
	}

	return Detail{Job: job, Dependencies: edges, DependencyJobs: depJobs, Runs: runs}, nil
}

func (s *Service) Trigger(ctx context.Context, jobID string) (jobrundomain.JobRun, error) {
	job, err := s.jobs.Get(ctx, jobID)
	if err != nil {
		return jobrundomain.JobRun{}, err
	}
	if !job.Enabled {
		return jobrundomain.JobRun{}, fmt.Errorf("job is paused")
	}
	now := time.Now().UTC()
	status := jobrundomain.StatusReady
	edges, err := s.dependencies.ListByJob(ctx, jobID)
	if err != nil {
		return jobrundomain.JobRun{}, err
	}
	if len(edges) > 0 {
		status = jobrundomain.StatusBlocked
	}
	run := jobrundomain.JobRun{
		ID:          newID(),
		JobID:       job.ID,
		ScheduledAt: now,
		Status:      status,
		TriggerType: "manual",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.runs.Save(ctx, run); err != nil {
		return jobrundomain.JobRun{}, err
	}
	if status == jobrundomain.StatusReady {
		if err := s.queue.Enqueue(ctx, run.ID); err != nil {
			return jobrundomain.JobRun{}, err
		}
	}
	return run, nil
}

func (s *Service) Pause(ctx context.Context, jobID string) (jobdomain.Job, error) {
	job, err := s.jobs.Get(ctx, jobID)
	if err != nil {
		return jobdomain.Job{}, err
	}
	job.Enabled = false
	job.UpdatedAt = time.Now().UTC()
	if err := s.jobs.Save(ctx, job); err != nil {
		return jobdomain.Job{}, err
	}
	return job, nil
}

func (s *Service) Resume(ctx context.Context, jobID string) (jobdomain.Job, error) {
	job, err := s.jobs.Get(ctx, jobID)
	if err != nil {
		return jobdomain.Job{}, err
	}
	job.Enabled = true
	nextRunAt, err := s.planner.Next(job, time.Now().UTC())
	if err != nil {
		return jobdomain.Job{}, err
	}
	job.NextRunAt = nextRunAt
	job.UpdatedAt = time.Now().UTC()
	if err := s.jobs.Save(ctx, job); err != nil {
		return jobdomain.Job{}, err
	}
	return job, nil
}

func (s *Service) ReadRunLog(ctx context.Context, runID string) (string, error) {
	return s.logs.Read(ctx, runID)
}

func (s *Service) AppendRunLog(ctx context.Context, runID, stream, content string) error {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	return s.logs.Append(ctx, runlog.LogRecord{
		RunID:      runID,
		Stream:     stream,
		Content:    content,
		OccurredAt: time.Now().UTC(),
	})
}

func (s *Service) buildEdges(ctx context.Context, jobID string, dependencyIDs []string) ([]dependencydomain.Edge, error) {
	if len(dependencyIDs) == 0 {
		return nil, nil
	}
	allJobs, err := s.jobs.List(ctx)
	if err != nil {
		return nil, err
	}
	known := make(map[string]struct{}, len(allJobs))
	for _, item := range allJobs {
		known[item.ID] = struct{}{}
	}

	uniqueIDs := make([]string, 0, len(dependencyIDs))
	for _, depID := range dependencyIDs {
		if depID == "" {
			continue
		}
		if depID == jobID {
			return nil, fmt.Errorf("job cannot depend on itself")
		}
		if _, ok := known[depID]; !ok {
			return nil, fmt.Errorf("dependency job %s not found", depID)
		}
		if !slices.Contains(uniqueIDs, depID) {
			uniqueIDs = append(uniqueIDs, depID)
		}
	}

	edges := make([]dependencydomain.Edge, 0, len(uniqueIDs))
	for _, depID := range uniqueIDs {
		edges = append(edges, dependencydomain.Edge{JobID: jobID, DependsOnJobID: depID})
	}

	existingEdges, err := s.dependencies.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	allEdges := append(existingEdges, edges...)
	if err := dependencydomain.ValidateGraph(allEdges); err != nil {
		return nil, err
	}
	return edges, nil
}

func newID() string {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

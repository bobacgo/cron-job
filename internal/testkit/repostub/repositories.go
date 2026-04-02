package repostub

import (
	"context"
	"errors"
	"sort"
	"sync"

	dependencydomain "github.com/bobacgo/cron-job/internal/domain/dependency"
	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
)

var errNotFound = errors.New("not found")

type JobRepo struct {
	mu   sync.RWMutex
	jobs map[string]jobdomain.Job
}

func NewJobRepo() *JobRepo {
	return &JobRepo{jobs: make(map[string]jobdomain.Job)}
}

func (r *JobRepo) Save(_ context.Context, job jobdomain.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs[job.ID] = job
	return nil
}

func (r *JobRepo) Get(_ context.Context, id string) (jobdomain.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.jobs[id]
	if !ok {
		return jobdomain.Job{}, errNotFound
	}
	return item, nil
}

func (r *JobRepo) List(_ context.Context) ([]jobdomain.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]jobdomain.Job, 0, len(r.jobs))
	for _, item := range r.jobs {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt < items[j].CreatedAt
	})
	return items, nil
}

func (r *JobRepo) ListEnabled(ctx context.Context) ([]jobdomain.Job, error) {
	items, err := r.List(ctx)
	if err != nil {
		return nil, err
	}
	result := items[:0]
	for _, item := range items {
		if item.Enabled {
			result = append(result, item)
		}
	}
	return result, nil
}

type JobRunRepo struct {
	mu      sync.RWMutex
	runs    map[string]jobrundomain.JobRun
	dedupBy map[string]string
}

func NewJobRunRepo() *JobRunRepo {
	return &JobRunRepo{
		runs:    make(map[string]jobrundomain.JobRun),
		dedupBy: make(map[string]string),
	}
}

func (r *JobRunRepo) Save(_ context.Context, run jobrundomain.JobRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runs[run.ID] = run
	r.dedupBy[run.DedupKey()] = run.ID
	return nil
}

func (r *JobRunRepo) Get(_ context.Context, id string) (jobrundomain.JobRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.runs[id]
	if !ok {
		return jobrundomain.JobRun{}, errNotFound
	}
	return item, nil
}

func (r *JobRunRepo) List(_ context.Context) ([]jobrundomain.JobRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]jobrundomain.JobRun, 0, len(r.runs))
	for _, item := range r.runs {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items, nil
}

func (r *JobRunRepo) ListByJob(_ context.Context, jobID string) ([]jobrundomain.JobRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]jobrundomain.JobRun, 0)
	for _, item := range r.runs {
		if item.JobID == jobID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items, nil
}

func (r *JobRunRepo) ListByStatus(_ context.Context, status jobrundomain.Status) ([]jobrundomain.JobRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]jobrundomain.JobRun, 0)
	for _, item := range r.runs {
		if item.Status == status {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items, nil
}

func (r *JobRunRepo) FindByDedupKey(_ context.Context, key string) (jobrundomain.JobRun, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.dedupBy[key]
	if !ok {
		return jobrundomain.JobRun{}, false, nil
	}
	return r.runs[id], true, nil
}

type DependencyRepo struct {
	mu    sync.RWMutex
	edges map[string][]dependencydomain.Edge
}

func NewDependencyRepo() *DependencyRepo {
	return &DependencyRepo{edges: make(map[string][]dependencydomain.Edge)}
}

func (r *DependencyRepo) Replace(_ context.Context, jobID string, edges []dependencydomain.Edge) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cloned := make([]dependencydomain.Edge, len(edges))
	copy(cloned, edges)
	r.edges[jobID] = cloned
	return nil
}

func (r *DependencyRepo) ListByJob(_ context.Context, jobID string) ([]dependencydomain.Edge, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := r.edges[jobID]
	cloned := make([]dependencydomain.Edge, len(items))
	copy(cloned, items)
	return cloned, nil
}

func (r *DependencyRepo) ListAll(_ context.Context) ([]dependencydomain.Edge, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	all := make([]dependencydomain.Edge, 0)
	for _, items := range r.edges {
		all = append(all, items...)
	}
	cloned := make([]dependencydomain.Edge, len(all))
	copy(cloned, all)
	return cloned, nil
}

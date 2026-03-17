package jobrun

import (
	"context"
	"errors"
	"sort"
	"sync"

	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
)

var ErrNotFound = errors.New("job run not found")

type InMemoryRepository struct {
	mu      sync.RWMutex
	runs    map[string]jobrundomain.JobRun
	dedupBy map[string]string
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		runs:    make(map[string]jobrundomain.JobRun),
		dedupBy: make(map[string]string),
	}
}

func (r *InMemoryRepository) Save(_ context.Context, run jobrundomain.JobRun) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runs[run.ID] = run
	r.dedupBy[run.DedupKey()] = run.ID
	return nil
}

func (r *InMemoryRepository) Get(_ context.Context, id string) (jobrundomain.JobRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	run, ok := r.runs[id]
	if !ok {
		return jobrundomain.JobRun{}, ErrNotFound
	}
	return run, nil
}

func (r *InMemoryRepository) List(_ context.Context) ([]jobrundomain.JobRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]jobrundomain.JobRun, 0, len(r.runs))
	for _, item := range r.runs {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

func (r *InMemoryRepository) ListByJob(_ context.Context, jobID string) ([]jobrundomain.JobRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]jobrundomain.JobRun, 0)
	for _, item := range r.runs {
		if item.JobID == jobID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

func (r *InMemoryRepository) ListByStatus(_ context.Context, status jobrundomain.Status) ([]jobrundomain.JobRun, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]jobrundomain.JobRun, 0)
	for _, item := range r.runs {
		if item.Status == status {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

func (r *InMemoryRepository) FindByDedupKey(_ context.Context, key string) (jobrundomain.JobRun, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.dedupBy[key]
	if !ok {
		return jobrundomain.JobRun{}, false, nil
	}
	run := r.runs[id]
	return run, true, nil
}

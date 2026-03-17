package job

import (
	"context"
	"errors"
	"sort"
	"sync"

	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
)

var ErrNotFound = errors.New("job not found")

type InMemoryRepository struct {
	mu   sync.RWMutex
	jobs map[string]jobdomain.Job
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{jobs: make(map[string]jobdomain.Job)}
}

func (r *InMemoryRepository) Save(_ context.Context, job jobdomain.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs[job.ID] = job
	return nil
}

func (r *InMemoryRepository) Get(_ context.Context, id string) (jobdomain.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	job, ok := r.jobs[id]
	if !ok {
		return jobdomain.Job{}, ErrNotFound
	}
	return job, nil
}

func (r *InMemoryRepository) List(_ context.Context) ([]jobdomain.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]jobdomain.Job, 0, len(r.jobs))
	for _, item := range r.jobs {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.Before(items[j].CreatedAt)
	})
	return items, nil
}

func (r *InMemoryRepository) ListEnabled(ctx context.Context) ([]jobdomain.Job, error) {
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

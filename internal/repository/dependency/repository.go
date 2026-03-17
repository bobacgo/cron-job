package dependency

import (
	"context"
	"sync"

	dependencydomain "github.com/bobacgo/cron-job/internal/domain/dependency"
)

type Repository interface {
	Replace(ctx context.Context, jobID string, edges []dependencydomain.Edge) error
	ListByJob(ctx context.Context, jobID string) ([]dependencydomain.Edge, error)
	ListAll(ctx context.Context) ([]dependencydomain.Edge, error)
}

type InMemoryRepository struct {
	mu    sync.RWMutex
	edges map[string][]dependencydomain.Edge
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{edges: make(map[string][]dependencydomain.Edge)}
}

func (r *InMemoryRepository) Replace(_ context.Context, jobID string, edges []dependencydomain.Edge) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	cloned := make([]dependencydomain.Edge, len(edges))
	copy(cloned, edges)
	r.edges[jobID] = cloned
	return nil
}

func (r *InMemoryRepository) ListByJob(_ context.Context, jobID string) ([]dependencydomain.Edge, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := r.edges[jobID]
	cloned := make([]dependencydomain.Edge, len(items))
	copy(cloned, items)
	return cloned, nil
}

func (r *InMemoryRepository) ListAll(_ context.Context) ([]dependencydomain.Edge, error) {
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

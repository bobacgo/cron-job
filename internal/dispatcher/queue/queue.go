package queue

import (
	"context"
	"errors"
	"sync"
)

var ErrEmpty = errors.New("queue empty")

type Queue interface {
	Enqueue(ctx context.Context, runID string) error
	Dequeue(ctx context.Context) (string, error)
}

type InMemoryQueue struct {
	mu      sync.Mutex
	items   []string
	knownID map[string]struct{}
}

func NewInMemoryQueue() *InMemoryQueue {
	return &InMemoryQueue{knownID: make(map[string]struct{})}
}

func (q *InMemoryQueue) Enqueue(_ context.Context, runID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if _, exists := q.knownID[runID]; exists {
		return nil
	}
	q.items = append(q.items, runID)
	q.knownID[runID] = struct{}{}
	return nil
}

func (q *InMemoryQueue) Dequeue(_ context.Context) (string, error) {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return "", ErrEmpty
	}
	item := q.items[0]
	q.items = q.items[1:]
	delete(q.knownID, item)
	return item, nil
}

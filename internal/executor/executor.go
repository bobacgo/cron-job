package executor

import (
	"context"
	"errors"
	"sync"
	"time"

	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
)

var ErrExecutorNotFound = errors.New("executor not found")

type Request struct {
	Job jobdomain.Job
	Run jobrundomain.JobRun
}

type Result struct {
	Status     jobrundomain.Status
	Message    string
	Output     string
	StartedAt  time.Time
	FinishedAt time.Time
}

type Executor interface {
	Execute(ctx context.Context, req Request) (Result, error)
}

type Registry struct {
	mu        sync.RWMutex
	executors map[string]Executor
}

func NewRegistry() *Registry {
	return &Registry{executors: make(map[string]Executor)}
}

func (r *Registry) Register(name string, exec Executor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.executors[name] = exec
}

func (r *Registry) Get(name string) (Executor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	exec, ok := r.executors[name]
	if !ok {
		return nil, ErrExecutorNotFound
	}
	return exec, nil
}

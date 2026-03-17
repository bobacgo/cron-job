package lease

import (
	"context"
	"errors"
	"sync"
	"time"

	leasedomain "github.com/bobacgo/cron-job/internal/domain/lease"
)

var ErrLeaseHeld = errors.New("lease already held")

type Manager interface {
	Acquire(ctx context.Context, runID, holderID string) error
	Renew(ctx context.Context, runID string) error
	Release(ctx context.Context, runID string) error
}

type MemoryManager struct {
	mu     sync.Mutex
	ttl    time.Duration
	leases map[string]leasedomain.Lease
}

func NewMemoryManager(ttl time.Duration) *MemoryManager {
	return &MemoryManager{ttl: ttl, leases: make(map[string]leasedomain.Lease)}
}

func (m *MemoryManager) Acquire(_ context.Context, runID, holderID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if lease, ok := m.leases[runID]; ok && !lease.Expired(time.Now()) {
		return ErrLeaseHeld
	}
	now := time.Now()
	m.leases[runID] = leasedomain.Lease{
		RunID:       runID,
		HolderID:    holderID,
		HeartbeatAt: now,
		ExpiresAt:   now.Add(m.ttl),
	}
	return nil
}

func (m *MemoryManager) Renew(_ context.Context, runID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	lease, ok := m.leases[runID]
	if !ok {
		return nil
	}
	now := time.Now()
	lease.HeartbeatAt = now
	lease.ExpiresAt = now.Add(m.ttl)
	m.leases[runID] = lease
	return nil
}

func (m *MemoryManager) Release(_ context.Context, runID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.leases, runID)
	return nil
}

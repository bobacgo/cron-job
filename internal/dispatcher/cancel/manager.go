package cancel

import "sync"

type Manager struct {
	mu      sync.Mutex
	cancels map[string]func()
}

func NewManager() *Manager {
	return &Manager{cancels: make(map[string]func())}
}

func (m *Manager) Register(runID string, cancel func()) {
	if runID == "" || cancel == nil {
		return
	}
	m.mu.Lock()
	m.cancels[runID] = cancel
	m.mu.Unlock()
}

func (m *Manager) Remove(runID string) {
	m.mu.Lock()
	delete(m.cancels, runID)
	m.mu.Unlock()
}

func (m *Manager) Cancel(runID string) bool {
	m.mu.Lock()
	cancel, ok := m.cancels[runID]
	m.mu.Unlock()
	if !ok {
		return false
	}
	cancel()
	return true
}

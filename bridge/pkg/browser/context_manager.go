package browser

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// BrowserContextManager maps agent IDs to isolated browser context IDs.
// It is a lightweight bookkeeping layer — actual browser context creation
// happens in Jetski; this manager only tracks the agent→context mapping.
type BrowserContextManager struct {
	mu       sync.RWMutex
	contexts map[string]string // agentID → contextID
	nextID   atomic.Int64
}

// NewBrowserContextManager creates a new BrowserContextManager.
func NewBrowserContextManager() *BrowserContextManager {
	return &BrowserContextManager{
		contexts: make(map[string]string),
	}
}

// AllocateContext assigns a browser context to an agent.
// If the agent already has a context, it returns the existing one.
// Otherwise it creates a new contextID, stores it, and returns it.
func (m *BrowserContextManager) AllocateContext(agentID string) string {
	// Fast path: read lock to check existing
	m.mu.RLock()
	if cid, ok := m.contexts[agentID]; ok {
		m.mu.RUnlock()
		return cid
	}
	m.mu.RUnlock()

	// Slow path: write lock to allocate
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if cid, ok := m.contexts[agentID]; ok {
		return cid
	}

	id := m.nextID.Add(1)
	cid := fmt.Sprintf("ctx-%d", id)
	m.contexts[agentID] = cid
	return cid
}

// ReleaseContext removes the context mapping for an agent.
// Returns true if the agent had an active context, false otherwise.
func (m *BrowserContextManager) ReleaseContext(agentID string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	_, existed := m.contexts[agentID]
	delete(m.contexts, agentID)
	return existed
}

// GetContext returns the contextID for an agent.
// Returns (contextID, true) if found, ("", false) otherwise.
func (m *BrowserContextManager) GetContext(agentID string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cid, ok := m.contexts[agentID]
	return cid, ok
}

// ActiveCount returns the number of active browser contexts.
func (m *BrowserContextManager) ActiveCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.contexts)
}

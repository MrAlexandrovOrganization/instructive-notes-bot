package state

import "sync"

// Manager manages user states in a thread-safe way.
type Manager struct {
	mu     sync.RWMutex
	states map[int64]*UserContext
}

// NewManager creates a new state Manager.
func NewManager() *Manager {
	return &Manager{states: make(map[int64]*UserContext)}
}

// Get returns the UserContext for a user, creating a default one if absent.
func (m *Manager) Get(userID int64) *UserContext {
	m.mu.RLock()
	ctx, ok := m.states[userID]
	m.mu.RUnlock()
	if !ok {
		return &UserContext{State: StateIdle}
	}
	return ctx
}

// Set updates the UserContext for a user.
func (m *Manager) Set(userID int64, ctx *UserContext) {
	m.mu.Lock()
	m.states[userID] = ctx
	m.mu.Unlock()
}

// Reset clears the UserContext for a user (sets to Idle).
func (m *Manager) Reset(userID int64) {
	m.mu.Lock()
	m.states[userID] = &UserContext{State: StateIdle}
	m.mu.Unlock()
}

// SetState is a convenience method to just update the state.
func (m *Manager) SetState(userID int64, s UserState) {
	m.mu.Lock()
	if m.states[userID] == nil {
		m.states[userID] = &UserContext{}
	}
	m.states[userID].State = s
	m.mu.Unlock()
}

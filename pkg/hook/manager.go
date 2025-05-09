// pkg/hook/manager.go
// Package hook provides a lightweight lifecycle extension mechanism.
// Hooks can be registered for specific event names and triggered manually or by the AppManager.
package hook

import (
	"context"
	"sync"
)

// HookFunc represents a function that can be triggered by a hook event.
type HookFunc func(ctx context.Context)

// Manager stores and manages hooks for different named events.
type Manager struct {
	mu        sync.RWMutex
	hooks     map[string][]HookFunc
	triggered map[string]bool
}

// NewManager creates and returns a new hook manager.
func NewManager() *Manager {
	return &Manager{
		hooks:     make(map[string][]HookFunc),
		triggered: make(map[string]bool),
	}
}

// Register adds a hook function to a named event.
func (m *Manager) Register(event string, fn HookFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hooks[event] = append(m.hooks[event], fn)
}

// Trigger calls all hooks registered to a named event.
func (m *Manager) Trigger(ctx context.Context, event string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.triggered[event] = true
	for _, fn := range m.hooks[event] {
		go fn(ctx) // trigger hooks asynchronously
	}
}

// IsTriggered checks if a specific event has been triggered.
func (m *Manager) IsTriggered(event string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.triggered[event]
}

// pkg/event/event.go
// Package event provides a simple publish-subscribe event bus for decoupled communication.
package event

import (
	"context"
	"sync"
)

// Handler is a function that handles an event.
type Handler func(ctx context.Context, data any)

// EventBus defines the interface for an event system.
type EventBus interface {
	Subscribe(event string, handler Handler)
	Publish(ctx context.Context, event string, data any)
}

// Bus represents the event bus.
type Bus struct {
	mu          sync.RWMutex
	subscribers map[string][]Handler
}

// New creates a new event bus.
func New() *Bus {
	return &Bus{
		subscribers: make(map[string][]Handler),
	}
}

// Subscribe adds a handler for a specific event.
func (b *Bus) Subscribe(event string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[event] = append(b.subscribers[event], handler)
}

// Publish triggers all handlers subscribed to the event.
func (b *Bus) Publish(ctx context.Context, event string, data any) {
	b.mu.RLock()
	handlers := append([]Handler{}, b.subscribers[event]...) // copy to avoid race
	b.mu.RUnlock()
	for _, handler := range handlers {
		go handler(ctx, data) // async execution
	}
}

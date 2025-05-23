// pkg/event/event_test.go

package event

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestBus_SubscribeAndPublish(t *testing.T) {
	bus := NewManager()
	var received int32

	bus.Subscribe("test_event", func(ctx context.Context, data any) {
		if msg, ok := data.(string); ok && msg == "hello" {
			atomic.AddInt32(&received, 1)
		}
	})

	ctx := context.Background()
	bus.Publish(ctx, "test_event", "hello")

	// Allow some time for the async handler to execute
	time.Sleep(100 * time.Millisecond)

	if received != 1 {
		t.Errorf("handler should have been called once, got %d", received)
	}
}

func TestBus_MultipleHandlers(t *testing.T) {
	bus := NewManager()
	var count int32

	bus.Subscribe("test_event", func(ctx context.Context, data any) {
		atomic.AddInt32(&count, 1)
	})
	bus.Subscribe("test_event", func(ctx context.Context, data any) {
		atomic.AddInt32(&count, 1)
	})

	ctx := context.Background()
	bus.Publish(ctx, "test_event", nil)

	// Allow some time for the async handlers to execute
	time.Sleep(100 * time.Millisecond)

	if count != 2 {
		t.Errorf("both handlers should have been called, got %d", count)
	}
}

func TestBus_NoSubscribers(t *testing.T) {
	bus := NewManager()

	// Publish an event with no subscribers
	ctx := context.Background()
	bus.Publish(ctx, "nonexistent_event", nil)

	// No panic or error should occur
	if false {
		t.Errorf("publishing to an event with no subscribers should not fail")
	}
}

func TestBus_ConcurrentAccess(t *testing.T) {
	bus := NewManager()
	var count int32

	bus.Subscribe("test_event", func(ctx context.Context, data any) {
		atomic.AddInt32(&count, 1)
	})

	ctx := context.Background()
	for i := 0; i < 100; i++ {
		go bus.Publish(ctx, "test_event", nil)
	}

	// Allow some time for the async handlers to execute
	time.Sleep(500 * time.Millisecond)

	if count != 100 {
		t.Errorf("all handlers should have been called, got %d", count)
	}
}

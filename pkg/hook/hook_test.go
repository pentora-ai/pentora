// pkg/hook/manager_test.go
package hook

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestHookManager_OnShutdown(t *testing.T) {
	t.Parallel() // Parallel test

	mgr := NewManager()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var (
		hookCalled bool
		wg         sync.WaitGroup
	)

	wg.Add(1)
	mgr.Register("onShutdown", func(ctx context.Context) {
		defer wg.Done()
		hookCalled = true

		if ctx.Err() != nil {
			t.Error("context should not be canceled")
		}
	})

	mgr.Trigger(ctx, "onShutdown")

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		if !hookCalled {
			t.Error("hook was not executed")
		}
	case <-ctx.Done():
		t.Fatal("test timed out waiting for hook")
	}
}

func TestMultipleHooks(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	var callCount int
	var mu sync.Mutex
	var wg sync.WaitGroup

	const numHooks = 5
	wg.Add(numHooks)

	for i := 0; i < numHooks; i++ {
		mgr.Register("test", func(ctx context.Context) {
			defer wg.Done()
			mu.Lock()
			callCount++
			mu.Unlock()
		})
	}

	mgr.Trigger(ctx, "test")
	wg.Wait()

	if callCount != numHooks {
		t.Errorf("expected %d hooks to be called, got %d", numHooks, callCount)
	}
}

func TestHookWithCanceledContext(t *testing.T) {
	mgr := NewManager()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var called bool
	var wg sync.WaitGroup
	wg.Add(1)

	mgr.Register("test", func(ctx context.Context) {
		defer wg.Done()
		called = true
		if ctx.Err() == nil {
			t.Error("expected context to be canceled")
		}
	})

	mgr.Trigger(ctx, "test")
	wg.Wait()

	if !called {
		t.Error("hook should be called even with canceled context")
	}
}

func TestIsTriggered(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()

	if mgr.IsTriggered("test") {
		t.Error("expected 'test' to not be triggered initially")
	}

	mgr.Trigger(ctx, "test")

	if !mgr.IsTriggered("test") {
		t.Error("expected 'test' to be triggered after calling Trigger")
	}
}

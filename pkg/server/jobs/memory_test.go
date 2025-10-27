// pkg/server/jobs/memory_test.go
package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewMemoryManager(t *testing.T) {
	t.Run("default concurrency", func(t *testing.T) {
		m := NewMemoryManager(0)
		require.NotNil(t, m)
		require.Equal(t, 4, m.concurrency, "Should default to 4 workers")
	})

	t.Run("custom concurrency", func(t *testing.T) {
		m := NewMemoryManager(8)
		require.NotNil(t, m)
		require.Equal(t, 8, m.concurrency, "Should use custom worker count")
	})

	t.Run("negative concurrency defaults to 4", func(t *testing.T) {
		m := NewMemoryManager(-5)
		require.NotNil(t, m)
		require.Equal(t, 4, m.concurrency, "Negative concurrency should default to 4")
	})
}

func TestMemoryManager_StartStop(t *testing.T) {
	t.Run("start and stop lifecycle", func(t *testing.T) {
		m := NewMemoryManager(2)
		ctx := context.Background()

		// Start manager
		err := m.Start(ctx)
		require.NoError(t, err, "Should start successfully")
		require.True(t, m.started, "Should be marked as started")

		// Give workers time to start
		time.Sleep(10 * time.Millisecond)

		// Stop manager
		stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err = m.Stop(stopCtx)
		require.NoError(t, err, "Should stop successfully")
		require.False(t, m.started, "Should be marked as stopped")
	})

	t.Run("double start returns error", func(t *testing.T) {
		m := NewMemoryManager(2)
		ctx := context.Background()

		err := m.Start(ctx)
		require.NoError(t, err)

		// Try to start again
		err = m.Start(ctx)
		require.Error(t, err, "Should return error on double start")
		require.Contains(t, err.Error(), "already started")

		// Cleanup
		stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()
		_ = m.Stop(stopCtx)
	})

	t.Run("stop idempotent", func(t *testing.T) {
		m := NewMemoryManager(2)
		ctx := context.Background()

		err := m.Start(ctx)
		require.NoError(t, err)

		stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Stop once
		err = m.Stop(stopCtx)
		require.NoError(t, err)

		// Stop again - should be no-op
		err = m.Stop(stopCtx)
		require.NoError(t, err, "Stop should be idempotent")
	})

	t.Run("stop without start is no-op", func(t *testing.T) {
		m := NewMemoryManager(2)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err := m.Stop(ctx)
		require.NoError(t, err, "Stop without start should be no-op")
	})
}

func TestMemoryManager_GracefulShutdown(t *testing.T) {
	t.Run("context cancellation triggers shutdown", func(t *testing.T) {
		m := NewMemoryManager(2)
		ctx, cancel := context.WithCancel(context.Background())

		err := m.Start(ctx)
		require.NoError(t, err)

		// Cancel context to trigger shutdown
		cancel()

		// Give workers time to detect cancellation
		time.Sleep(50 * time.Millisecond)

		// Stop should complete quickly since context already canceled
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
		defer stopCancel()

		err = m.Stop(stopCtx)
		require.NoError(t, err, "Should stop gracefully after context cancellation")
	})

	t.Run("stop timeout triggers error", func(t *testing.T) {
		m := NewMemoryManager(2)
		ctx := context.Background()

		err := m.Start(ctx)
		require.NoError(t, err)

		// Create already-canceled context for immediate timeout
		stopCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Manually prevent workers from stopping by not calling cancelFunc
		// This simulates workers being stuck and not responding to cancellation
		originalCancelFunc := m.cancelFunc
		m.mu.Lock()
		m.cancelFunc = func() {} // No-op cancel function
		m.mu.Unlock()

		err = m.Stop(stopCtx)
		require.Error(t, err, "Should return error when stop times out")
		require.ErrorIs(t, err, context.Canceled)

		// Cleanup: restore cancel function and actually stop workers
		m.mu.Lock()
		m.cancelFunc = originalCancelFunc
		m.mu.Unlock()
		if originalCancelFunc != nil {
			originalCancelFunc()
		}
		time.Sleep(50 * time.Millisecond)
	})
}

func TestMemoryManager_JobProcessing(t *testing.T) {
	t.Run("workers process jobs from queue", func(t *testing.T) {
		m := NewMemoryManager(2)
		ctx := context.Background()

		err := m.Start(ctx)
		require.NoError(t, err)

		// Give workers time to start
		time.Sleep(10 * time.Millisecond)

		// Submit jobs to the queue
		testJobs := []Job{
			{ID: "job-1", Type: "test-type-1", Payload: "data-1"},
			{ID: "job-2", Type: "test-type-2", Payload: "data-2"},
			{ID: "job-3", Type: "test-type-3", Payload: "data-3"},
		}

		for _, job := range testJobs {
			m.jobQueue <- job
		}

		// Give workers time to process jobs
		time.Sleep(50 * time.Millisecond)

		// Stop manager
		stopCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err = m.Stop(stopCtx)
		require.NoError(t, err, "Should stop after processing jobs")
	})
}

package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewMemoryManager(t *testing.T) {
	mgr := NewMemoryManager(4)

	require.NotNil(t, mgr)
	require.Equal(t, 4, mgr.concurrency)
	require.NotNil(t, mgr.queue)
}

func TestNewMemoryManager_DefaultConcurrency(t *testing.T) {
	mgr := NewMemoryManager(0)

	require.NotNil(t, mgr)
	require.Equal(t, 4, mgr.concurrency, "Should default to 4 when concurrency is 0")
}

func TestNewMemoryManager_NegativeConcurrency(t *testing.T) {
	mgr := NewMemoryManager(-1)

	require.NotNil(t, mgr)
	require.Equal(t, 4, mgr.concurrency, "Should default to 4 when concurrency is negative")
}

func TestMemoryManager_StartStop(t *testing.T) {
	mgr := NewMemoryManager(2)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start
	err := mgr.Start(ctx)
	require.NoError(t, err)
	require.Len(t, mgr.workers, 2)

	// Give workers time to start
	time.Sleep(50 * time.Millisecond)

	// Stop
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()

	err = mgr.Stop(stopCtx)
	require.NoError(t, err)
}

func TestMemoryManager_GracefulShutdown(t *testing.T) {
	mgr := NewMemoryManager(2)

	ctx, cancel := context.WithCancel(context.Background())

	err := mgr.Start(ctx)
	require.NoError(t, err)

	// Cancel context (simulating server shutdown)
	cancel()

	// Stop should complete quickly
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer stopCancel()

	err = mgr.Stop(stopCtx)
	require.NoError(t, err)
}

func TestMemoryManager_MultipleWorkers(t *testing.T) {
	mgr := NewMemoryManager(5)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := mgr.Start(ctx)
	require.NoError(t, err)
	require.Len(t, mgr.workers, 5, "Should create 5 workers")

	// Give workers time to start
	time.Sleep(50 * time.Millisecond)

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()

	err = mgr.Stop(stopCtx)
	require.NoError(t, err)
}

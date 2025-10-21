package deps

import (
	"context"
	"testing"

	"github.com/pentora-ai/pentora/pkg/engine"
	"github.com/pentora-ai/pentora/pkg/storage"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// MockStorage is a mock implementation of storage.Backend for testing.
type MockStorage struct{}

func (m *MockStorage) Initialize(ctx context.Context) error { return nil }
func (m *MockStorage) Close() error                         { return nil }
func (m *MockStorage) Scans() storage.ScanStore             { return nil }

func TestNew(t *testing.T) {
	logger := zerolog.Nop()
	storage := &MockStorage{}
	engineMgr := engine.NewTestAppManager()

	deps := New(storage, engineMgr, &logger)

	require.NotNil(t, deps)
	require.Equal(t, storage, deps.Storage)
	require.Equal(t, engineMgr, deps.Engine)
	require.Equal(t, &logger, deps.Logger)
	require.NotNil(t, deps.Ready)
	require.False(t, deps.IsReady(), "Should start as not ready")
}

func TestDeps_ReadyState(t *testing.T) {
	logger := zerolog.Nop()
	storage := &MockStorage{}
	engineMgr := engine.NewTestAppManager()

	deps := New(storage, engineMgr, &logger)

	// Initially not ready
	require.False(t, deps.IsReady())

	// Set ready
	deps.SetReady()
	require.True(t, deps.IsReady())

	// Set not ready
	deps.SetNotReady()
	require.False(t, deps.IsReady())
}

func TestDeps_ReadyThreadSafe(t *testing.T) {
	logger := zerolog.Nop()
	storage := &MockStorage{}
	engineMgr := engine.NewTestAppManager()

	deps := New(storage, engineMgr, &logger)

	// Test concurrent access to ready state
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			deps.SetReady()
			deps.SetNotReady()
			deps.IsReady()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// No panic = success
}

// pkg/core/manager_test.go

package app_test

import (
	"context"
	"testing"

	app "github.com/pentoraai/pentora/pkg/core"
)

func TestNewAppManager(t *testing.T) {
	manager := app.NewAppManager()
	if manager == nil {
		t.Errorf("Expected non-nil manager, got nil")
	}
}

func TestAppManagerInit(t *testing.T) {
	manager := app.NewAppManager()
	err := manager.Init()
	if err != nil {
		t.Errorf("Expected no error during initialization, got %v", err)
	}
	if manager.HookManager == nil {
		t.Errorf("Expected HookManager to be initialized, got nil")
	}
}

func TestAppManagerContext(t *testing.T) {
	manager := app.NewAppManager()
	if manager.Context() == nil {
		t.Errorf("Expected non-nil context, got nil")
	}
}

func TestAppManagerShutdown(t *testing.T) {
	manager := app.NewAppManager()
	manager.Init()

	// Ensure HookManager is initialized
	if manager.HookManager == nil {
		t.Fatalf("HookManager should be initialized before shutdown")
	}

	// Mock a hook to verify it gets triggered
	triggered := false
	manager.HookManager.Register("onShutdown", func(ctx context.Context) {
		triggered = true
	})

	manager.Shutdown()

	if !triggered && !manager.HookManager.IsTriggered("onShutdown") {
		t.Errorf("Expected onShutdown hook to be triggered, but it was not")
	}
}



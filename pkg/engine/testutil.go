package engine

import (
	"context"

	"github.com/vulntor/vulntor/pkg/config"
	"github.com/vulntor/vulntor/pkg/event"
	"github.com/vulntor/vulntor/pkg/hook"
)

// NewTestAppManager creates a minimal AppManager for tests without loading config files.
func NewTestAppManager() *AppManager {
	ctx, cancel := context.WithCancel(context.Background())
	cfg := config.NewManager()
	return &AppManager{
		ctx:           ctx,
		cancel:        cancel,
		ConfigManager: cfg,
		EventManager:  event.NewManager(),
		HookManager:   hook.NewManager(),
	}
}

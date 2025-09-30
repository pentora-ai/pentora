package engine

import (
	"context"

	"github.com/pentora-ai/pentora/pkg/config"
	"github.com/pentora-ai/pentora/pkg/event"
	"github.com/pentora-ai/pentora/pkg/hook"
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

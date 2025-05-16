// pkg/core/app.go

// AppManager is the core lifecycle orchestrator for Pentora.
// It is responsible for initializing and managing the application runtime environment.
// All subsystems (config, logging, hooks, registry, etc.) are injected and controlled from here.
package core

import (
	"context"
	"sync"

	"github.com/pentora-ai/pentora/pkg/event"
	"github.com/pentora-ai/pentora/pkg/hook"
	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/pentora-ai/pentora/pkg/scan"
	"github.com/pentora-ai/pentora/pkg/version"
)

// AppManager is the central controller for the application's lifecycle.
type AppManager struct {
	ctx    context.Context    // shared context for all subsystems
	cancel context.CancelFunc // cancellation for graceful shutdown

	//Config     *config.Config // configuration subsystem
	//LogManager *log.Manager   // logging subsystem
	HookManager  *hook.Manager            // hooks subsystem
	EventBus     event.EventBus           // internal pub/sub event bus
	Orchestrator scan.Orchestrator        // orchestrator for running plugins
	Plugin       plugin.RegistryInterface //
	Version      version.Struct           // version manager for checking updates

	once sync.Once // ensures single initialization
}

// NewAppManager creates a new AppManager instance with an isolated context.
func NewAppManager() *AppManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &AppManager{
		ctx:    ctx,
		cancel: cancel,
	}
}

// Init initializes all subsystems and prepares the application environment.
func (a *AppManager) Init() error {
	a.once.Do(func() {
		// Initialize logging first so other subsystems can log properly
		//a.LogManager = log.NewManager()
		//a.LogManager.Init()

		// Initialize configuration manager
		//a.Config = config.LoadFromEnv()

		// Initialize hook manager for internal lifecycle extensions
		a.HookManager = hook.NewManager()

		// Initialize internal event bus for pub/sub communication
		a.EventBus = event.New()

		//
		a.Plugin = plugin.GlobalRegistry

		// Load version information at startup
		vm := version.Get()
		a.Version = vm

		a.Orchestrator = scan.New(a.Plugin, a.HookManager)
	})

	return nil
}

// Context returns the shared application context.
func (a *AppManager) Context() context.Context {
	return a.ctx
}

// Shutdown gracefully shuts down all subsystems.
func (a *AppManager) Shutdown() {
	a.cancel()
	if a.HookManager != nil {
		a.HookManager.Trigger(a.ctx, "onShutdown")
	}
}

// AppManager manages the application lifecycle, providing access to the application's context and version information.
// It is constructed by the Factory and holds a cancellable context for controlling application execution.
// pkg/engine/manager.go
package engine

import (
	"context"

	"github.com/pentora-ai/pentora/pkg/config"
	"github.com/pentora-ai/pentora/pkg/event"
	"github.com/pentora-ai/pentora/pkg/hook"
	"github.com/pentora-ai/pentora/pkg/version"
)

// AppManager represents the application manager constructed by Factory.
type AppManager struct {
	// ctx is the context for managing request-scoped values, cancellation signals, and deadlines across API boundaries.
	ctx context.Context
	// cancel is the function to cancel the associated context, used to signal termination or cleanup.
	cancel context.CancelFunc

	ConfigManager *config.Manager // Configuration manager for loading and managing application settings.

	EventManager *event.Manager // Event manager for handling events and notifications within the application.

	HookManager *hook.Manager // Hook manager for managing lifecycle hooks and custom event triggers.

	// Version represents the version information of the engine, encapsulated in the version.Struct type.
	Version version.Struct
}

// Context returns the context associated with the AppManager instance.
// This context can be used for managing deadlines, cancellation signals,
// and other request-scoped values across API boundaries and between processes.
func (a *AppManager) Context() context.Context {
	return a.ctx
}

// Shutdown gracefully shuts down the AppManager by invoking its cancellation function.
// This will signal any running operations to terminate and release associated resources.
func (a *AppManager) Shutdown() {
	a.cancel()
}

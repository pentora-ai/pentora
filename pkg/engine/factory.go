// pkg/engine/factory.go
package engine

import (
	"context"
	"fmt"

	"github.com/pentora-ai/pentora/pkg/config"
	"github.com/pentora-ai/pentora/pkg/logging"
	"github.com/pentora-ai/pentora/pkg/version"
	"github.com/spf13/pflag"
)

// Factory is responsible for constructing an AppManager instance with all required components.
type AppManagerFactory interface {
	Create(flags *pflag.FlagSet, configFile string) (*AppManager, error)
}

// DefaultAppManagerFactory is a factory type responsible for creating instances of the default application manager.
// It provides methods to instantiate and configure application managers with default settings.
type DefaultAppManagerFactory struct{}

// Create initializes a new AppManager instance with a cancellable context and the current application version.
// It returns a pointer to the created AppManager.
func (f *DefaultAppManagerFactory) Create(flags *pflag.FlagSet, configFile string) (*AppManager, error) {
	context, cancel := context.WithCancel(context.Background())

	ConfigManager := config.NewManager()
	if err := ConfigManager.Load(flags, configFile); err != nil {
		return nil, err
	}

	if err := logging.ConfigureGlobalLogging(ConfigManager.Get().Log.Level); err != nil {
		return nil, fmt.Errorf("failed to configure global logging: %w", err)
	}

	return &AppManager{
		ctx:           context,
		cancel:        cancel,
		ConfigManager: ConfigManager,
		Version:       version.Get(),
	}, nil
}

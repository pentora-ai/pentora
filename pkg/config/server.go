package config

import (
	"time"

	"github.com/spf13/pflag"
)

// DefaultServerConfig returns the default server configuration.
// These are sensible defaults for local development and can be overridden
// via flags, environment variables, or config files.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Addr:         "127.0.0.1",
		Port:         8080,
		UIEnabled:    true,
		APIEnabled:   true,
		JobsEnabled:  true,
		Concurrency:  4,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
}

// BindServerFlags binds server-specific flags to the provided FlagSet.
// These flags will be used by the 'pentora server start' command.
//
// Flags are namespaced under 'server.' to avoid conflicts with global flags.
// Example: --server.addr, --server.port
//
// This function should be called when setting up the server command.
func BindServerFlags(flags *pflag.FlagSet) {
	defaults := DefaultServerConfig()

	flags.String("server.addr", defaults.Addr, "Server listen address (use 0.0.0.0 for all interfaces)")
	flags.Int("server.port", defaults.Port, "Server listen port")
	flags.Bool("server.ui_enabled", defaults.UIEnabled, "Enable UI static serving")
	flags.Bool("server.api_enabled", defaults.APIEnabled, "Enable REST API endpoints")
	flags.Bool("server.jobs_enabled", defaults.JobsEnabled, "Enable background job workers")
	flags.String("server.ui_assets_path", "", "UI assets directory (dev mode: serve from disk instead of embedded)")
	flags.Int("server.concurrency", defaults.Concurrency, "Number of concurrent background workers")
	flags.Duration("server.read_timeout", defaults.ReadTimeout, "HTTP read timeout")
	flags.Duration("server.write_timeout", defaults.WriteTimeout, "HTTP write timeout")
}

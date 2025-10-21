package config

import (
	"fmt"
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
		UI: UIConfig{
			DevMode:    false,
			AssetsPath: "",
		},
		Auth: AuthConfig{
			Mode:  "token",
			Token: "",
		},
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

	// UI flags
	flags.Bool("server.ui.dev_mode", defaults.UI.DevMode, "Enable dev mode (disables auth, localhost only)")
	flags.String("server.ui.assets_path", defaults.UI.AssetsPath, "Override embedded assets with disk path")

	// Auth flags
	flags.String("server.auth.mode", defaults.Auth.Mode, "Authentication mode: none|token|oidc")
	flags.String("server.auth.token", defaults.Auth.Token, "Static bearer token (required for token mode)")
}

// Validate validates the ServerConfig and returns an error if invalid.
func (c *ServerConfig) Validate() error {
	// Validate port range
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1-65535)", c.Port)
	}

	// Validate concurrency
	if c.Concurrency < 1 {
		return fmt.Errorf("invalid concurrency: %d (must be >= 1)", c.Concurrency)
	}

	// Validate timeouts
	if c.ReadTimeout < 0 {
		return fmt.Errorf("invalid read_timeout: %v (must be >= 0)", c.ReadTimeout)
	}
	if c.WriteTimeout < 0 {
		return fmt.Errorf("invalid write_timeout: %v (must be >= 0)", c.WriteTimeout)
	}

	// Validate auth config
	if err := c.Auth.Validate(); err != nil {
		return fmt.Errorf("auth config: %w", err)
	}

	// If dev mode is enabled, override some settings for safety
	if c.UI.DevMode {
		// Dev mode should only listen on localhost
		if c.Addr != "127.0.0.1" && c.Addr != "localhost" {
			return fmt.Errorf("dev_mode requires addr to be localhost or 127.0.0.1, got: %s", c.Addr)
		}
		// Dev mode disables auth
		c.Auth.Mode = "none"
	}

	return nil
}

// ListenAddr returns the full listen address (addr:port).
func (c *ServerConfig) ListenAddr() string {
	return fmt.Sprintf("%s:%d", c.Addr, c.Port)
}

// IsAuthEnabled returns true if authentication is enabled.
func (c *ServerConfig) IsAuthEnabled() bool {
	return c.Auth.Mode != "none"
}

// Validate validates the AuthConfig and returns an error if invalid.
func (a *AuthConfig) Validate() error {
	// Validate auth mode
	switch a.Mode {
	case "none", "token", "oidc":
		// Valid modes
	default:
		return fmt.Errorf("invalid auth mode: %s (must be none|token|oidc)", a.Mode)
	}

	// Token mode requires a non-empty token
	if a.Mode == "token" && a.Token == "" {
		return fmt.Errorf("token mode requires a non-empty auth.token")
	}

	// OIDC mode is not yet implemented
	if a.Mode == "oidc" {
		return fmt.Errorf("oidc mode is not yet implemented")
	}

	return nil
}

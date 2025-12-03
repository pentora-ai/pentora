package config

import (
	"fmt"
	"time"
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
			AssetsPath: "",
		},
		Auth: AuthConfig{
			Mode:  "token",
			Token: "",
		},
	}
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

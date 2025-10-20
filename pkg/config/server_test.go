package config

import (
	"testing"
	"time"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

func TestDefaultServerConfig(t *testing.T) {
	cfg := DefaultServerConfig()

	// Network settings
	require.Equal(t, "127.0.0.1", cfg.Addr)
	require.Equal(t, 8080, cfg.Port)

	// Component toggles
	require.True(t, cfg.UIEnabled)
	require.True(t, cfg.APIEnabled)
	require.True(t, cfg.JobsEnabled)

	// Performance
	require.Equal(t, 4, cfg.Concurrency)

	// Timeouts
	require.Equal(t, 30*time.Second, cfg.ReadTimeout)
	require.Equal(t, 30*time.Second, cfg.WriteTimeout)

	// Paths should be empty by default
	require.Empty(t, cfg.WorkspaceDir)
	require.Empty(t, cfg.UIAssetsPath)
}

func TestBindServerFlags(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	BindServerFlags(flags)

	// Parse test flags
	err := flags.Parse([]string{
		"--server.addr=0.0.0.0",
		"--server.port=9090",
		"--server.ui_enabled=false",
		"--server.concurrency=8",
	})
	require.NoError(t, err)

	// Verify flags were registered and parsed correctly
	addr, err := flags.GetString("server.addr")
	require.NoError(t, err)
	require.Equal(t, "0.0.0.0", addr)

	port, err := flags.GetInt("server.port")
	require.NoError(t, err)
	require.Equal(t, 9090, port)

	uiEnabled, err := flags.GetBool("server.ui_enabled")
	require.NoError(t, err)
	require.False(t, uiEnabled)

	concurrency, err := flags.GetInt("server.concurrency")
	require.NoError(t, err)
	require.Equal(t, 8, concurrency)
}

func TestBindServerFlags_Defaults(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	BindServerFlags(flags)

	// Don't parse any flags, just check defaults
	defaults := DefaultServerConfig()

	addr, err := flags.GetString("server.addr")
	require.NoError(t, err)
	require.Equal(t, defaults.Addr, addr)

	port, err := flags.GetInt("server.port")
	require.NoError(t, err)
	require.Equal(t, defaults.Port, port)

	uiEnabled, err := flags.GetBool("server.ui_enabled")
	require.NoError(t, err)
	require.Equal(t, defaults.UIEnabled, uiEnabled)

	apiEnabled, err := flags.GetBool("server.api_enabled")
	require.NoError(t, err)
	require.Equal(t, defaults.APIEnabled, apiEnabled)
}

func TestBindServerFlags_AllFlags(t *testing.T) {
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	BindServerFlags(flags)

	// Verify all expected flags are registered
	expectedFlags := []string{
		"server.addr",
		"server.port",
		"server.ui_enabled",
		"server.api_enabled",
		"server.jobs_enabled",
		"server.ui_assets_path",
		"server.concurrency",
		"server.read_timeout",
		"server.write_timeout",
	}

	for _, flagName := range expectedFlags {
		flag := flags.Lookup(flagName)
		require.NotNil(t, flag, "Flag %s should be registered", flagName)
	}
}

func TestServerConfig_Integration(t *testing.T) {
	// Test integration with config manager
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	BindServerFlags(flags)

	err := flags.Parse([]string{
		"--server.addr=0.0.0.0",
		"--server.port=8888",
		"--server.concurrency=10",
	})
	require.NoError(t, err)

	// Create config manager and load
	mgr := NewManager()
	err = mgr.Load(flags, "")
	require.NoError(t, err)

	// Get final config
	cfg := mgr.Get()

	// Verify server config was loaded correctly
	require.Equal(t, "0.0.0.0", cfg.Server.Addr)
	require.Equal(t, 8888, cfg.Server.Port)
	require.Equal(t, 10, cfg.Server.Concurrency)

	// Verify defaults for non-overridden values
	require.True(t, cfg.Server.UIEnabled)
	require.True(t, cfg.Server.APIEnabled)
}

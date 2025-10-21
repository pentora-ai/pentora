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

	// UI config
	require.False(t, cfg.UI.DevMode)
	require.Empty(t, cfg.UI.AssetsPath)

	// Auth config
	require.Equal(t, "token", cfg.Auth.Mode)
	require.Empty(t, cfg.Auth.Token)
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
		"server.ui.dev_mode",
		"server.ui.assets_path",
		"server.auth.mode",
		"server.auth.token",
	}

	for _, flagName := range expectedFlags {
		flag := flags.Lookup(flagName)
		require.NotNil(t, flag, "Flag %s should be registered", flagName)
	}
}

func TestServerConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     ServerConfig
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid default config with token",
			cfg: func() ServerConfig {
				c := DefaultServerConfig()
				c.Auth.Token = "test-token"
				return c
			}(),
			wantErr: false,
		},
		{
			name: "invalid port - too low",
			cfg: ServerConfig{
				Port: 0,
				Auth: AuthConfig{Mode: "none"},
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
		{
			name: "invalid port - too high",
			cfg: ServerConfig{
				Port: 65536,
				Auth: AuthConfig{Mode: "none"},
			},
			wantErr: true,
			errMsg:  "invalid port",
		},
		{
			name: "invalid concurrency",
			cfg: ServerConfig{
				Port:        8080,
				Concurrency: 0,
				Auth:        AuthConfig{Mode: "none"},
			},
			wantErr: true,
			errMsg:  "invalid concurrency",
		},
		{
			name: "invalid read timeout",
			cfg: ServerConfig{
				Port:        8080,
				Concurrency: 1,
				ReadTimeout: -1 * time.Second,
				Auth:        AuthConfig{Mode: "none"},
			},
			wantErr: true,
			errMsg:  "invalid read_timeout",
		},
		{
			name: "invalid write timeout",
			cfg: ServerConfig{
				Port:         8080,
				Concurrency:  1,
				WriteTimeout: -1 * time.Second,
				Auth:         AuthConfig{Mode: "none"},
			},
			wantErr: true,
			errMsg:  "invalid write_timeout",
		},
		{
			name: "token mode without token",
			cfg: ServerConfig{
				Port:        8080,
				Concurrency: 1,
				Auth:        AuthConfig{Mode: "token", Token: ""},
			},
			wantErr: true,
			errMsg:  "token mode requires",
		},
		{
			name: "dev mode with non-localhost addr",
			cfg: ServerConfig{
				Addr:        "0.0.0.0",
				Port:        8080,
				Concurrency: 1,
				UI:          UIConfig{DevMode: true},
				Auth:        AuthConfig{Mode: "none"},
			},
			wantErr: true,
			errMsg:  "dev_mode requires addr to be localhost",
		},
		{
			name: "dev mode with localhost",
			cfg: ServerConfig{
				Addr:        "127.0.0.1",
				Port:        8080,
				Concurrency: 1,
				UI:          UIConfig{DevMode: true},
				Auth:        AuthConfig{Mode: "none"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServerConfig_ListenAddr(t *testing.T) {
	tests := []struct {
		name string
		cfg  ServerConfig
		want string
	}{
		{
			name: "default config",
			cfg:  ServerConfig{Addr: "127.0.0.1", Port: 8080},
			want: "127.0.0.1:8080",
		},
		{
			name: "custom port",
			cfg:  ServerConfig{Addr: "0.0.0.0", Port: 9000},
			want: "0.0.0.0:9000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.ListenAddr()
			require.Equal(t, tt.want, got)
		})
	}
}

func TestServerConfig_IsAuthEnabled(t *testing.T) {
	tests := []struct {
		name string
		cfg  ServerConfig
		want bool
	}{
		{
			name: "auth mode none",
			cfg:  ServerConfig{Auth: AuthConfig{Mode: "none"}},
			want: false,
		},
		{
			name: "auth mode token",
			cfg:  ServerConfig{Auth: AuthConfig{Mode: "token"}},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cfg.IsAuthEnabled()
			require.Equal(t, tt.want, got)
		})
	}
}

func TestAuthConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     AuthConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid none mode",
			cfg:     AuthConfig{Mode: "none"},
			wantErr: false,
		},
		{
			name:    "valid token mode",
			cfg:     AuthConfig{Mode: "token", Token: "test-token"},
			wantErr: false,
		},
		{
			name:    "invalid mode",
			cfg:     AuthConfig{Mode: "invalid"},
			wantErr: true,
			errMsg:  "invalid auth mode",
		},
		{
			name:    "token mode without token",
			cfg:     AuthConfig{Mode: "token", Token: ""},
			wantErr: true,
			errMsg:  "token mode requires a non-empty auth.token",
		},
		{
			name:    "oidc mode not implemented",
			cfg:     AuthConfig{Mode: "oidc"},
			wantErr: true,
			errMsg:  "oidc mode is not yet implemented",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
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

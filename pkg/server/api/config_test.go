package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// Verify default timeout values
	require.Equal(t, 30*time.Second, cfg.HandlerTimeout, "HandlerTimeout should be 30s")
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errType error
	}{
		{
			name: "valid default config",
			config: Config{
				HandlerTimeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "valid custom timeout",
			config: Config{
				HandlerTimeout: 60 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "zero timeout is valid (disables timeout)",
			config: Config{
				HandlerTimeout: 0,
			},
			wantErr: false,
		},
		{
			name: "negative timeout is invalid",
			config: Config{
				HandlerTimeout: -1 * time.Second,
			},
			wantErr: true,
			errType: ErrInvalidTimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					require.ErrorIs(t, err, tt.errType)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

package bind

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestBindFingerprintOptions(t *testing.T) {
	tests := []struct {
		name    string
		flags   map[string]interface{}
		want    FingerprintOptions
		wantErr bool
	}{
		{
			name: "all flags set",
			flags: map[string]interface{}{
				"file":      "/path/to/catalog.yaml",
				"url":       "https://example.com/catalog",
				"cache-dir": "/custom/cache",
			},
			want: FingerprintOptions{
				FilePath: "/path/to/catalog.yaml",
				URL:      "https://example.com/catalog",
				CacheDir: "/custom/cache",
			},
			wantErr: false,
		},
		{
			name: "only file set",
			flags: map[string]interface{}{
				"file":      "/local/catalog.yaml",
				"url":       "",
				"cache-dir": "",
			},
			want: FingerprintOptions{
				FilePath: "/local/catalog.yaml",
				URL:      "",
				CacheDir: "",
			},
			wantErr: false,
		},
		{
			name: "only url set",
			flags: map[string]interface{}{
				"file":      "",
				"url":       "https://probes.pentora.ai/catalog.yaml",
				"cache-dir": "",
			},
			want: FingerprintOptions{
				FilePath: "",
				URL:      "https://probes.pentora.ai/catalog.yaml",
				CacheDir: "",
			},
			wantErr: false,
		},
		{
			name: "defaults (all empty)",
			flags: map[string]interface{}{
				"file":      "",
				"url":       "",
				"cache-dir": "",
			},
			want: FingerprintOptions{
				FilePath: "",
				URL:      "",
				CacheDir: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := setupFingerprintCommand(tt.flags)
			got, err := BindFingerprintOptions(cmd)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// setupFingerprintCommand creates a mock command with fingerprint flags
func setupFingerprintCommand(flags map[string]interface{}) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().String("file", "", "File path")
	cmd.Flags().String("url", "", "URL")
	cmd.Flags().String("cache-dir", "", "Cache dir")

	// Set flag values
	if filePath, ok := flags["file"].(string); ok {
		_ = cmd.Flags().Set("file", filePath)
	}
	if url, ok := flags["url"].(string); ok {
		_ = cmd.Flags().Set("url", url)
	}
	if cacheDir, ok := flags["cache-dir"].(string); ok {
		_ = cmd.Flags().Set("cache-dir", cacheDir)
	}

	return cmd
}

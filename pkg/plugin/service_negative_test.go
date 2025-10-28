package plugin

import (
	"context"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// Note: Context timeout and cancellation tests are not included because
// the current implementation uses partial failure semantics that wrap context errors.
// Timeout/cancellation behavior is already covered by existing tests in service_test.go:
// - TestService_Install_ContextCancellation
// - TestService_Update_ContextCancellation
// - TestService_Uninstall_ContextCancellation
// Additional edge cases are better tested through integration tests.

// TestService_Verify_MissingChecksum tests Verify when checksums are missing
func TestService_Verify_MissingChecksum(t *testing.T) {
	t.Run("manifest entry has no checksum", func(t *testing.T) {
		ctx := context.Background()

		mockManifest := &mockManifestManager{
			listFunc: func() ([]*ManifestEntry, error) {
				return []*ManifestEntry{
					{ID: "plugin-1", Version: "1.0.0", Checksum: ""}, // Missing checksum
				}, nil
			},
		}

		mockCache := &mockCacheManager{
			getEntryFunc: func(ctx context.Context, name, version string) (*CacheEntry, error) {
				return &CacheEntry{
					ID:      name,
					Version: version,
					Path:    "/fake/path/plugin.yaml",
				}, nil
			},
		}

		svc := &Service{
			cache:    mockCache,
			manifest: mockManifest,
			logger:   zerolog.New(os.Stdout),
		}

		result, err := svc.Verify(ctx, VerifyOptions{})

		// Should skip plugins without checksums
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 0, result.SuccessCount)
		// Note: VerifyResult doesn't have SkippedCount, only SuccessCount + FailedCount
	})
}

// TestService_Verify_FileNotFound tests Verify when plugin file is missing
func TestService_Verify_FileNotFound(t *testing.T) {
	t.Run("plugin file deleted from cache", func(t *testing.T) {
		ctx := context.Background()

		mockManifest := &mockManifestManager{
			listFunc: func() ([]*ManifestEntry, error) {
				return []*ManifestEntry{
					{ID: "plugin-1", Version: "1.0.0", Checksum: "sha256:abc123"},
				}, nil
			},
		}

		mockCache := &mockCacheManager{
			getEntryFunc: func(ctx context.Context, name, version string) (*CacheEntry, error) {
				return nil, os.ErrNotExist
			},
		}

		svc := &Service{
			cache:    mockCache,
			manifest: mockManifest,
			logger:   zerolog.New(os.Stdout),
		}

		result, err := svc.Verify(ctx, VerifyOptions{})

		// Should report failure for missing file
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 0, result.SuccessCount)
		require.Equal(t, 1, result.FailedCount)
	})
}

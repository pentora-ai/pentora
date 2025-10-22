package storage

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRetentionConfig_IsEnabled(t *testing.T) {
	tests := []struct {
		name     string
		config   RetentionConfig
		expected bool
	}{
		{
			name:     "no policies",
			config:   RetentionConfig{},
			expected: false,
		},
		{
			name:     "only max_scans",
			config:   RetentionConfig{MaxScans: 10},
			expected: true,
		},
		{
			name:     "only max_age_days",
			config:   RetentionConfig{MaxAgeDays: 30},
			expected: true,
		},
		{
			name:     "both policies",
			config:   RetentionConfig{MaxScans: 10, MaxAgeDays: 30},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.IsEnabled()
			require.Equal(t, tt.expected, got)
		})
	}
}

func TestRetentionConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  RetentionConfig
		wantErr bool
	}{
		{
			name:    "valid: no limits",
			config:  RetentionConfig{},
			wantErr: false,
		},
		{
			name:    "valid: max_scans only",
			config:  RetentionConfig{MaxScans: 100},
			wantErr: false,
		},
		{
			name:    "valid: max_age_days only",
			config:  RetentionConfig{MaxAgeDays: 30},
			wantErr: false,
		},
		{
			name:    "valid: both limits",
			config:  RetentionConfig{MaxScans: 100, MaxAgeDays: 30},
			wantErr: false,
		},
		{
			name:    "invalid: negative max_scans",
			config:  RetentionConfig{MaxScans: -1},
			wantErr: true,
		},
		{
			name:    "invalid: negative max_age_days",
			config:  RetentionConfig{MaxAgeDays: -1},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestLocalBackend_GarbageCollect_NoPolicy(t *testing.T) {
	backend := setupTestBackend(t)
	ctx := context.Background()

	// Create some scans
	createTestScan(t, backend, ctx, "default", "scan-1", time.Now().Add(-10*24*time.Hour))
	createTestScan(t, backend, ctx, "default", "scan-2", time.Now().Add(-5*24*time.Hour))

	// Run GC with no retention policy
	result, err := backend.GarbageCollect(ctx, GCOptions{})
	require.NoError(t, err)
	require.Equal(t, 0, result.ScansDeleted)
	require.Empty(t, result.DeletedScanIDs)
}

func TestLocalBackend_GarbageCollect_MaxScans(t *testing.T) {
	backend := setupTestBackend(t)
	ctx := context.Background()

	// Create 5 scans with different ages
	createTestScan(t, backend, ctx, "default", "scan-1", time.Now().Add(-50*24*time.Hour))
	createTestScan(t, backend, ctx, "default", "scan-2", time.Now().Add(-40*24*time.Hour))
	createTestScan(t, backend, ctx, "default", "scan-3", time.Now().Add(-30*24*time.Hour))
	createTestScan(t, backend, ctx, "default", "scan-4", time.Now().Add(-20*24*time.Hour))
	createTestScan(t, backend, ctx, "default", "scan-5", time.Now().Add(-10*24*time.Hour))

	// Run GC with MaxScans=3 (should delete 2 oldest)
	result, err := backend.GarbageCollect(ctx, GCOptions{
		Retention: &RetentionConfig{MaxScans: 3},
	})
	require.NoError(t, err)
	require.Equal(t, 2, result.ScansDeleted)
	require.Contains(t, result.DeletedScanIDs, "scan-1")
	require.Contains(t, result.DeletedScanIDs, "scan-2")

	// Verify remaining scans
	scans, err := backend.Scans().List(ctx, "default", ScanFilter{})
	require.NoError(t, err)
	require.Len(t, scans, 3)
}

func TestLocalBackend_GarbageCollect_MaxAgeDays(t *testing.T) {
	backend := setupTestBackend(t)
	ctx := context.Background()

	// Create scans with different ages
	createTestScan(t, backend, ctx, "default", "scan-old-1", time.Now().Add(-50*24*time.Hour))
	createTestScan(t, backend, ctx, "default", "scan-old-2", time.Now().Add(-40*24*time.Hour))
	createTestScan(t, backend, ctx, "default", "scan-recent-1", time.Now().Add(-10*24*time.Hour))
	createTestScan(t, backend, ctx, "default", "scan-recent-2", time.Now().Add(-5*24*time.Hour))

	// Run GC with MaxAgeDays=30 (should delete scans older than 30 days)
	result, err := backend.GarbageCollect(ctx, GCOptions{
		Retention: &RetentionConfig{MaxAgeDays: 30},
	})
	require.NoError(t, err)
	require.Equal(t, 2, result.ScansDeleted)
	require.Contains(t, result.DeletedScanIDs, "scan-old-1")
	require.Contains(t, result.DeletedScanIDs, "scan-old-2")

	// Verify remaining scans
	scans, err := backend.Scans().List(ctx, "default", ScanFilter{})
	require.NoError(t, err)
	require.Len(t, scans, 2)
}

func TestLocalBackend_GarbageCollect_BothPolicies(t *testing.T) {
	backend := setupTestBackend(t)
	ctx := context.Background()

	// Create scans with different ages
	createTestScan(t, backend, ctx, "default", "scan-1", time.Now().Add(-50*24*time.Hour))
	createTestScan(t, backend, ctx, "default", "scan-2", time.Now().Add(-40*24*time.Hour))
	createTestScan(t, backend, ctx, "default", "scan-3", time.Now().Add(-20*24*time.Hour))
	createTestScan(t, backend, ctx, "default", "scan-4", time.Now().Add(-10*24*time.Hour))
	createTestScan(t, backend, ctx, "default", "scan-5", time.Now().Add(-5*24*time.Hour))

	// Run GC with MaxScans=3 AND MaxAgeDays=30
	// Should delete: scan-1, scan-2 (older than 30 days)
	// Then check if remaining > 3: scan-3, scan-4, scan-5 = 3, so no more deletions
	result, err := backend.GarbageCollect(ctx, GCOptions{
		Retention: &RetentionConfig{
			MaxScans:   3,
			MaxAgeDays: 30,
		},
	})
	require.NoError(t, err)
	require.Equal(t, 2, result.ScansDeleted)
	require.Contains(t, result.DeletedScanIDs, "scan-1")
	require.Contains(t, result.DeletedScanIDs, "scan-2")

	// Verify remaining scans
	scans, err := backend.Scans().List(ctx, "default", ScanFilter{})
	require.NoError(t, err)
	require.Len(t, scans, 3)
}

func TestLocalBackend_GarbageCollect_DryRun(t *testing.T) {
	backend := setupTestBackend(t)
	ctx := context.Background()

	// Create scans
	createTestScan(t, backend, ctx, "default", "scan-1", time.Now().Add(-50*24*time.Hour))
	createTestScan(t, backend, ctx, "default", "scan-2", time.Now().Add(-10*24*time.Hour))

	// Run GC in dry-run mode
	result, err := backend.GarbageCollect(ctx, GCOptions{
		DryRun:    true,
		Retention: &RetentionConfig{MaxAgeDays: 30},
	})
	require.NoError(t, err)
	require.Equal(t, 1, result.ScansDeleted)
	require.Contains(t, result.DeletedScanIDs, "scan-1")

	// Verify NO scans were actually deleted
	scans, err := backend.Scans().List(ctx, "default", ScanFilter{})
	require.NoError(t, err)
	require.Len(t, scans, 2, "dry-run should not delete scans")
}

// Helper functions

func createTestScan(t *testing.T, backend Backend, ctx context.Context, orgID, scanID string, startedAt time.Time) {
	t.Helper()

	metadata := &ScanMetadata{
		ID:              scanID,
		OrgID:           orgID,
		UserID:          "test-user",
		Target:          "192.168.1.1",
		Status:          "completed",
		StartedAt:       startedAt,
		CompletedAt:     startedAt.Add(1 * time.Minute),
		HostCount:       1,
		VulnCount:       VulnCounts{},
		StorageLocation: "",
	}

	err := backend.Scans().Create(ctx, orgID, metadata)
	require.NoError(t, err)
}

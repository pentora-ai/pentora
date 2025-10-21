package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewLocalBackend(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				WorkspaceRoot: t.TempDir(),
			},
			wantErr: false,
		},
		{
			name: "invalid config - empty workspace",
			cfg: &Config{
				WorkspaceRoot: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := NewLocalBackend(context.Background(), tt.cfg)
			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, backend)
			} else {
				require.NoError(t, err)
				require.NotNil(t, backend)
				require.NotNil(t, backend.Scans())
			}
		})
	}
}

func TestLocalBackend_Initialize(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	backend, err := NewLocalBackend(ctx, &Config{
		WorkspaceRoot: tmpDir,
	})
	require.NoError(t, err)

	err = backend.Initialize(ctx)
	require.NoError(t, err)

	// Verify directory structure
	expectedDirs := []string{
		"scans",
		"queue",
		"cache",
		"logs",
		"reports",
		"audit",
	}

	for _, dir := range expectedDirs {
		path := filepath.Join(tmpDir, dir)
		info, err := os.Stat(path)
		require.NoError(t, err, "directory %s should exist", dir)
		require.True(t, info.IsDir(), "%s should be a directory", dir)
	}
}

func TestLocalBackend_Close(t *testing.T) {
	ctx := context.Background()
	backend, err := NewLocalBackend(ctx, &Config{
		WorkspaceRoot: t.TempDir(),
	})
	require.NoError(t, err)

	err = backend.Close()
	require.NoError(t, err)

	// Calling Close again should not error
	err = backend.Close()
	require.NoError(t, err)
}

func TestLocalScanStore_Create(t *testing.T) {
	ctx := context.Background()
	backend := setupTestBackend(t)

	scanStore := backend.Scans()

	tests := []struct {
		name    string
		scan    *ScanMetadata
		wantErr bool
		errType error
	}{
		{
			name: "valid scan",
			scan: &ScanMetadata{
				ID:     "scan-1",
				Target: "192.168.1.0/24",
				Status: string(StatusPending),
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			scan: &ScanMetadata{
				Target: "192.168.1.0/24",
				Status: string(StatusPending),
			},
			wantErr: true,
			errType: &InvalidInputError{},
		},
		{
			name: "missing target",
			scan: &ScanMetadata{
				ID:     "scan-2",
				Status: string(StatusPending),
			},
			wantErr: true,
			errType: &InvalidInputError{},
		},
		{
			name: "duplicate scan",
			scan: &ScanMetadata{
				ID:     "scan-1", // Already created
				Target: "192.168.1.0/24",
				Status: string(StatusPending),
			},
			wantErr: true,
			errType: &AlreadyExistsError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := scanStore.Create(ctx, "default", tt.scan)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					require.ErrorAs(t, err, &tt.errType)
				}
			} else {
				require.NoError(t, err)

				// Verify scan was created
				retrieved, err := scanStore.Get(ctx, "default", tt.scan.ID)
				require.NoError(t, err)
				require.Equal(t, tt.scan.ID, retrieved.ID)
				require.Equal(t, tt.scan.Target, retrieved.Target)
				require.Equal(t, tt.scan.Status, retrieved.Status)
				require.False(t, retrieved.CreatedAt.IsZero())
				require.False(t, retrieved.UpdatedAt.IsZero())
			}
		})
	}
}

func TestLocalScanStore_Get(t *testing.T) {
	ctx := context.Background()
	backend := setupTestBackend(t)
	scanStore := backend.Scans()

	// Create a scan
	scan := &ScanMetadata{
		ID:     "scan-1",
		Target: "192.168.1.0/24",
		Status: string(StatusPending),
	}
	err := scanStore.Create(ctx, "default", scan)
	require.NoError(t, err)

	tests := []struct {
		name    string
		orgID   string
		scanID  string
		wantErr bool
		errType error
	}{
		{
			name:    "existing scan",
			orgID:   "default",
			scanID:  "scan-1",
			wantErr: false,
		},
		{
			name:    "non-existent scan",
			orgID:   "default",
			scanID:  "scan-999",
			wantErr: true,
			errType: &NotFoundError{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := scanStore.Get(ctx, tt.orgID, tt.scanID)
			if tt.wantErr {
				require.Error(t, err)
				if tt.errType != nil {
					require.ErrorAs(t, err, &tt.errType)
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, retrieved)
				require.Equal(t, tt.scanID, retrieved.ID)
			}
		})
	}
}

func TestLocalScanStore_Update(t *testing.T) {
	ctx := context.Background()
	backend := setupTestBackend(t)
	scanStore := backend.Scans()

	// Create a scan
	scan := &ScanMetadata{
		ID:     "scan-1",
		Target: "192.168.1.0/24",
		Status: string(StatusPending),
	}
	err := scanStore.Create(ctx, "default", scan)
	require.NoError(t, err)

	// Update scan
	completedAt := time.Now()
	duration := 120
	status := string(StatusCompleted)
	hostCount := 10
	serviceCount := 25

	updates := ScanUpdates{
		Status:       &status,
		CompletedAt:  &completedAt,
		Duration:     &duration,
		HostCount:    &hostCount,
		ServiceCount: &serviceCount,
	}

	err = scanStore.Update(ctx, "default", "scan-1", updates)
	require.NoError(t, err)

	// Verify updates
	retrieved, err := scanStore.Get(ctx, "default", "scan-1")
	require.NoError(t, err)
	require.Equal(t, string(StatusCompleted), retrieved.Status)
	require.Equal(t, duration, retrieved.Duration)
	require.Equal(t, hostCount, retrieved.HostCount)
	require.Equal(t, serviceCount, retrieved.ServiceCount)
	require.WithinDuration(t, completedAt, retrieved.CompletedAt, time.Second)
}

func TestLocalScanStore_Delete(t *testing.T) {
	ctx := context.Background()
	backend := setupTestBackend(t)
	scanStore := backend.Scans()

	// Create a scan
	scan := &ScanMetadata{
		ID:     "scan-1",
		Target: "192.168.1.0/24",
		Status: string(StatusPending),
	}
	err := scanStore.Create(ctx, "default", scan)
	require.NoError(t, err)

	// Delete scan
	err = scanStore.Delete(ctx, "default", "scan-1")
	require.NoError(t, err)

	// Verify scan is deleted
	_, err = scanStore.Get(ctx, "default", "scan-1")
	require.Error(t, err)
	require.True(t, IsNotFound(err))

	// Deleting again should return not found
	err = scanStore.Delete(ctx, "default", "scan-1")
	require.Error(t, err)
	require.True(t, IsNotFound(err))
}

func TestLocalScanStore_List(t *testing.T) {
	ctx := context.Background()
	backend := setupTestBackend(t)
	scanStore := backend.Scans()

	// Create multiple scans
	scans := []*ScanMetadata{
		{
			ID:     "scan-1",
			Target: "192.168.1.0/24",
			Status: string(StatusPending),
		},
		{
			ID:     "scan-2",
			Target: "192.168.2.0/24",
			Status: string(StatusRunning),
		},
		{
			ID:     "scan-3",
			Target: "192.168.1.100",
			Status: string(StatusCompleted),
		},
	}

	for _, scan := range scans {
		err := scanStore.Create(ctx, "default", scan)
		require.NoError(t, err)
	}

	tests := []struct {
		name      string
		filter    ScanFilter
		wantCount int
	}{
		{
			name:      "list all",
			filter:    ScanFilter{},
			wantCount: 3,
		},
		{
			name: "filter by status",
			filter: ScanFilter{
				Status: string(StatusPending),
			},
			wantCount: 1,
		},
		{
			name: "filter by target substring",
			filter: ScanFilter{
				Target: "192.168.1",
			},
			wantCount: 2,
		},
		{
			name: "limit results",
			filter: ScanFilter{
				Limit: 2,
			},
			wantCount: 2,
		},
		{
			name: "offset results",
			filter: ScanFilter{
				Offset: 1,
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := scanStore.List(ctx, "default", tt.filter)
			require.NoError(t, err)
			require.Len(t, results, tt.wantCount)
		})
	}
}

func TestLocalScanStore_ListEmptyOrg(t *testing.T) {
	ctx := context.Background()
	backend := setupTestBackend(t)
	scanStore := backend.Scans()

	// List scans for non-existent org
	scans, err := scanStore.List(ctx, "non-existent-org", ScanFilter{})
	require.NoError(t, err)
	require.Empty(t, scans)
}

func TestLocalScanStore_WriteData(t *testing.T) {
	ctx := context.Background()
	backend := setupTestBackend(t)
	scanStore := backend.Scans()

	// Create a scan
	scan := &ScanMetadata{
		ID:     "scan-1",
		Target: "192.168.1.0/24",
		Status: string(StatusPending),
	}
	err := scanStore.Create(ctx, "default", scan)
	require.NoError(t, err)

	// Write data
	data := strings.NewReader(`{"ip":"192.168.1.1","status":"up"}
{"ip":"192.168.1.2","status":"up"}
`)
	err = scanStore.WriteData(ctx, "default", "scan-1", DataTypeHosts, data)
	require.NoError(t, err)

	// Verify data was written
	reader, err := scanStore.ReadData(ctx, "default", "scan-1", DataTypeHosts)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Contains(t, string(content), "192.168.1.1")
	require.Contains(t, string(content), "192.168.1.2")
}

func TestLocalScanStore_AppendData(t *testing.T) {
	ctx := context.Background()
	backend := setupTestBackend(t)
	scanStore := backend.Scans()

	// Create a scan
	scan := &ScanMetadata{
		ID:     "scan-1",
		Target: "192.168.1.0/24",
		Status: string(StatusPending),
	}
	err := scanStore.Create(ctx, "default", scan)
	require.NoError(t, err)

	// Append data multiple times
	err = scanStore.AppendData(ctx, "default", "scan-1", DataTypeHosts, []byte(`{"ip":"192.168.1.1"}`+"\n"))
	require.NoError(t, err)

	err = scanStore.AppendData(ctx, "default", "scan-1", DataTypeHosts, []byte(`{"ip":"192.168.1.2"}`+"\n"))
	require.NoError(t, err)

	// Read and verify
	reader, err := scanStore.ReadData(ctx, "default", "scan-1", DataTypeHosts)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()

	content, err := io.ReadAll(reader)
	require.NoError(t, err)

	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.Len(t, lines, 2)
	require.Contains(t, lines[0], "192.168.1.1")
	require.Contains(t, lines[1], "192.168.1.2")
}

func TestLocalScanStore_ReadData_NotFound(t *testing.T) {
	ctx := context.Background()
	backend := setupTestBackend(t)
	scanStore := backend.Scans()

	// Create a scan but don't write data
	scan := &ScanMetadata{
		ID:     "scan-1",
		Target: "192.168.1.0/24",
		Status: string(StatusPending),
	}
	err := scanStore.Create(ctx, "default", scan)
	require.NoError(t, err)

	// Try to read non-existent data file
	_, err = scanStore.ReadData(ctx, "default", "scan-1", DataTypeHosts)
	require.Error(t, err)
	require.True(t, IsNotFound(err))
}

func TestLocalScanStore_InvalidDataType(t *testing.T) {
	ctx := context.Background()
	backend := setupTestBackend(t)
	scanStore := backend.Scans()

	// Create a scan
	scan := &ScanMetadata{
		ID:     "scan-1",
		Target: "192.168.1.0/24",
		Status: string(StatusPending),
	}
	err := scanStore.Create(ctx, "default", scan)
	require.NoError(t, err)

	// Try to write with invalid data type
	err = scanStore.WriteData(ctx, "default", "scan-1", DataType("invalid.txt"), strings.NewReader("data"))
	require.Error(t, err)
	require.True(t, IsInvalidInput(err))
}

func TestLocalScanStore_GetAnalytics(t *testing.T) {
	ctx := context.Background()
	backend := setupTestBackend(t)
	scanStore := backend.Scans()

	// GetAnalytics should return ErrNotSupported for OSS
	_, err := scanStore.GetAnalytics(ctx, "default", TimePeriod{})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotSupported)
}

// Helper function to set up a test backend
func setupTestBackend(t *testing.T) *LocalBackend {
	t.Helper()

	ctx := context.Background()
	tmpDir := t.TempDir()

	backend, err := NewLocalBackend(ctx, &Config{
		WorkspaceRoot: tmpDir,
	})
	require.NoError(t, err)

	err = backend.Initialize(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = backend.Close()
	})

	return backend
}

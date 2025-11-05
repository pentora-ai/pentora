package storage

import (
	"context"
	"encoding/json"
	"fmt"
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

// --- extra tests for full coverage ---

func TestLocalScanStore_NormalizeLimit(t *testing.T) {
	s := &LocalScanStore{}
	require.Equal(t, 50, s.normalizeLimit(0))
	require.Equal(t, 50, s.normalizeLimit(-10))
	require.Equal(t, 100, s.normalizeLimit(200))
	require.Equal(t, 25, s.normalizeLimit(25))
}

func TestLocalScanStore_MatchesFilter(t *testing.T) {
	s := &LocalScanStore{}
	meta := &ScanMetadata{Status: string(StatusCompleted), Target: "host1"}

	require.True(t, s.matchesFilter(meta, ScanFilter{}))
	require.False(t, s.matchesFilter(meta, ScanFilter{Status: string(StatusPending)}))
	require.False(t, s.matchesFilter(meta, ScanFilter{Target: "xyz"}))
	require.True(t, s.matchesFilter(meta, ScanFilter{Target: "host"}))
}

func TestLocalScanStore_SortAndFindCursor(t *testing.T) {
	s := &LocalScanStore{}
	now := time.Now()
	scans := []*ScanMetadata{
		{ID: "1", StartedAt: now.Add(-3 * time.Minute)},
		{ID: "2", StartedAt: now.Add(-1 * time.Minute)},
		{ID: "3", StartedAt: now.Add(-2 * time.Minute)},
	}

	// sort by time descending
	s.sortScansByTime(scans)
	require.Equal(t, "2", scans[0].ID)

	// find cursor positions
	require.Equal(t, 0, s.findCursorPosition(scans, nil))
	require.Equal(t, 1, s.findCursorPosition(scans, &Cursor{LastScanID: "2"}))
	require.Equal(t, 0, s.findCursorPosition(scans, &Cursor{LastScanID: "x"}))
}

func TestLocalScanStore_PaginateScans(t *testing.T) {
	s := &LocalScanStore{}
	now := time.Now()
	scans := []*ScanMetadata{
		{ID: "a", StartedAt: now.Add(-3 * time.Minute)},
		{ID: "b", StartedAt: now.Add(-2 * time.Minute)},
		{ID: "c", StartedAt: now.Add(-1 * time.Minute)},
	}

	// first page (no cursor)
	page, next := s.paginateScans(scans, nil, 2)
	require.Len(t, page, 2)
	require.NotEmpty(t, next)

	// continue with cursor
	cur, err := DecodeCursor(next)
	require.NoError(t, err)
	page2, next2 := s.paginateScans(scans, cur, 2)
	require.Len(t, page2, 1)
	require.Empty(t, next2)
}

func TestLocalScanStore_ListPaginated_AllBranches(t *testing.T) {
	ctx := context.Background()
	backend := setupTestBackend(t)
	store := backend.Scans().(*LocalScanStore)

	// create sample scans
	now := time.Now()
	for i := 0; i < 3; i++ {
		scan := &ScanMetadata{
			ID:        fmt.Sprintf("scan-%d", i),
			Target:    "target",
			Status:    string(StatusCompleted),
			StartedAt: now.Add(-time.Duration(i) * time.Hour),
		}
		require.NoError(t, store.Create(ctx, "org", scan))
	}

	// valid pagination
	cur := &Cursor{LastScanID: "scan-0", LastTime: now.UnixNano()}
	cursorStr := EncodeCursor(cur)
	page, next, total, err := store.ListPaginated(ctx, "org", ScanFilter{}, cursorStr, 1)
	require.NoError(t, err)
	require.NotEmpty(t, page)
	require.NotZero(t, total)
	require.NotEmpty(t, next)

	// invalid cursor encoding
	pageBad, nextBad, totalBad, err2 := store.ListPaginated(ctx, "org", ScanFilter{}, "%%%bad", 2)
	require.Error(t, err2)
	require.True(t, IsInvalidInput(err2))
	require.Nil(t, pageBad)
	require.Empty(t, nextBad)
	require.Zero(t, totalBad)

	// missing org
	page, next, total, err = store.ListPaginated(ctx, "no-org", ScanFilter{}, "", 2)
	require.NoError(t, err)
	require.Empty(t, page)
	require.Zero(t, total)
	require.Empty(t, next)

	// limit normalization
	page, _, _, err = store.ListPaginated(ctx, "org", ScanFilter{}, "", 0)
	require.NoError(t, err)
	require.NotEmpty(t, page)
	page, _, _, err = store.ListPaginated(ctx, "org", ScanFilter{}, "", 999)
	require.NoError(t, err)
	require.NotEmpty(t, page)
}

func TestLocalScanStore_LoadFilteredScans_Cases(t *testing.T) {
	ctx := context.Background()
	backend := setupTestBackend(t)
	store := backend.Scans().(*LocalScanStore)

	// no org dir
	scans, err := store.loadFilteredScans(ctx, "none", ScanFilter{})
	require.NoError(t, err)
	require.Empty(t, scans)

	// valid dir
	org := "orgx"
	require.NoError(t, os.MkdirAll(filepath.Join(store.root, org, "scan1"), 0o755))
	meta := &ScanMetadata{
		ID:        "scan1",
		Target:    "target",
		Status:    string(StatusCompleted),
		StartedAt: time.Now(),
	}
	data, _ := json.Marshal(meta)
	require.NoError(t, os.WriteFile(filepath.Join(store.root, org, "scan1", "metadata.json"), data, 0o644))

	scans, err = store.loadFilteredScans(ctx, org, ScanFilter{})
	require.NoError(t, err)
	require.Len(t, scans, 1)
}

func TestLocalScanStore_List_BasicFilteringAndPagination(t *testing.T) {
	ctx := context.Background()
	backend := setupTestBackend(t)
	scanStore := backend.Scans()

	// Create scans with different targets and statuses
	scans := []*ScanMetadata{
		{ID: "scan-1", Target: "10.0.0.1", Status: string(StatusPending)},
		{ID: "scan-2", Target: "10.0.0.2", Status: string(StatusRunning)},
		{ID: "scan-3", Target: "10.0.0.3", Status: string(StatusCompleted)},
		{ID: "scan-4", Target: "10.0.0.1", Status: string(StatusCompleted)},
	}

	for _, scan := range scans {
		require.NoError(t, scanStore.Create(ctx, "org1", scan))
	}

	t.Run("list all scans", func(t *testing.T) {
		results, err := scanStore.List(ctx, "org1", ScanFilter{})
		require.NoError(t, err)
		require.Len(t, results, 4)
	})

	t.Run("filter by status", func(t *testing.T) {
		results, err := scanStore.List(ctx, "org1", ScanFilter{Status: string(StatusCompleted)})
		require.NoError(t, err)
		require.Len(t, results, 2)
		for _, s := range results {
			require.Equal(t, string(StatusCompleted), s.Status)
		}
	})

	t.Run("filter by target substring", func(t *testing.T) {
		results, err := scanStore.List(ctx, "org1", ScanFilter{Target: "10.0.0.1"})
		require.NoError(t, err)
		require.Len(t, results, 2)
		for _, s := range results {
			require.Contains(t, s.Target, "10.0.0.1")
		}
	})

	t.Run("limit results", func(t *testing.T) {
		results, err := scanStore.List(ctx, "org1", ScanFilter{Limit: 2})
		require.NoError(t, err)
		require.Len(t, results, 2)
	})

	t.Run("offset results", func(t *testing.T) {
		results, err := scanStore.List(ctx, "org1", ScanFilter{Offset: 3})
		require.NoError(t, err)
		require.Len(t, results, 1)
	})

	t.Run("offset exceeds results", func(t *testing.T) {
		results, err := scanStore.List(ctx, "org1", ScanFilter{Offset: 10})
		require.NoError(t, err)
		require.Empty(t, results)
	})

	t.Run("limit less than available", func(t *testing.T) {
		results, err := scanStore.List(ctx, "org1", ScanFilter{Limit: 1})
		require.NoError(t, err)
		require.Len(t, results, 1)
	})

	t.Run("limit greater than available", func(t *testing.T) {
		results, err := scanStore.List(ctx, "org1", ScanFilter{Limit: 10})
		require.NoError(t, err)
		require.Len(t, results, 4)
	})

	t.Run("non-existent org returns empty", func(t *testing.T) {
		results, err := scanStore.List(ctx, "no-org", ScanFilter{})
		require.NoError(t, err)
		require.Empty(t, results)
	})

	t.Run("scan with invalid metadata is skipped", func(t *testing.T) {
		// Create a directory with no metadata.json
		scanDir := filepath.Join(backend.Scans().(*LocalScanStore).root, "org1", "badscan")
		require.NoError(t, os.MkdirAll(scanDir, 0o755))
		results, err := scanStore.List(ctx, "org1", ScanFilter{})
		require.NoError(t, err)
		// Should still only return the 4 valid scans
		require.Len(t, results, 4)
	})
}

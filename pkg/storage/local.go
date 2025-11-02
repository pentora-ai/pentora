package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

func init() {
	// Register LocalBackend as the default factory for OSS edition
	DefaultFactory = func(ctx context.Context, cfg *Config) (Backend, error) {
		return NewLocalBackend(ctx, cfg)
	}
}

// LocalBackend implements Backend using file-based storage.
//
// Storage layout:
//
//	{workspace}/
//	  scans/
//	    {org-id}/
//	      {scan-id}/
//	        metadata.json
//	        hosts.jsonl
//	        services.jsonl
//	        vulnerabilities.jsonl
//	        banners.txt
//
// Thread-safety: All operations are protected by file locks for concurrent access.
type LocalBackend struct {
	cfg       *Config
	scanStore *LocalScanStore
	mu        sync.RWMutex
	closed    bool
}

// NewLocalBackend creates a new file-based backend.
func NewLocalBackend(ctx context.Context, cfg *Config) (*LocalBackend, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	backend := &LocalBackend{
		cfg: cfg,
	}

	// Create scan store
	backend.scanStore = &LocalScanStore{
		root: filepath.Join(cfg.WorkspaceRoot, "scans"),
	}

	return backend, nil
}

// Initialize prepares the backend for use.
func (b *LocalBackend) Initialize(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return ErrClosed
	}

	// Create workspace directory structure
	dirs := []string{
		filepath.Join(b.cfg.WorkspaceRoot, "scans"),
		filepath.Join(b.cfg.WorkspaceRoot, "queue"),
		filepath.Join(b.cfg.WorkspaceRoot, "cache"),
		filepath.Join(b.cfg.WorkspaceRoot, "logs"),
		filepath.Join(b.cfg.WorkspaceRoot, "reports"),
		filepath.Join(b.cfg.WorkspaceRoot, "audit"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// Close releases resources held by the backend.
func (b *LocalBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true
	return nil
}

// Scans returns the scan storage interface.
func (b *LocalBackend) Scans() ScanStore {
	return b.scanStore
}

// LocalScanStore implements ScanStore using file-based storage.
type LocalScanStore struct {
	root string // Root directory for scans (workspace/scans)
}

// List returns a list of scans matching the given filter.
func (s *LocalScanStore) List(ctx context.Context, orgID string, filter ScanFilter) ([]*ScanMetadata, error) {
	orgDir := filepath.Join(s.root, orgID)

	// Check if org directory exists
	if _, err := os.Stat(orgDir); os.IsNotExist(err) {
		return []*ScanMetadata{}, nil
	}

	// Read all scan directories
	entries, err := os.ReadDir(orgDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read org directory: %w", err)
	}

	var scans []*ScanMetadata
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		scanID := entry.Name()
		metadata, err := s.Get(ctx, orgID, scanID)
		if err != nil {
			// Skip scans with invalid metadata
			continue
		}

		// Apply filters
		if filter.Status != "" && string(metadata.Status) != filter.Status {
			continue
		}
		if filter.Target != "" && !strings.Contains(metadata.Target, filter.Target) {
			continue
		}

		scans = append(scans, metadata)
	}

	// Apply sorting
	// TODO: Implement sorting based on filter.SortBy and filter.SortOrder

	// Apply pagination
	if filter.Offset > 0 {
		if filter.Offset >= len(scans) {
			return []*ScanMetadata{}, nil
		}
		scans = scans[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(scans) {
		scans = scans[:filter.Limit]
	}

	return scans, nil
}

// ListPaginated returns a paginated list of scans matching the given filter.
func (s *LocalScanStore) ListPaginated(ctx context.Context, orgID string, filter ScanFilter, cursor string, limit int) ([]*ScanMetadata, string, int, error) {
	// Validate limit
	limit = s.normalizeLimit(limit)

	// Decode cursor
	cursorData, err := DecodeCursor(cursor)
	if err != nil {
		return nil, "", 0, NewInvalidInputError("cursor", err.Error())
	}

	// Load and filter scans
	allScans, err := s.loadFilteredScans(ctx, orgID, filter)
	if err != nil {
		return nil, "", 0, err
	}

	// Sort by start time (newest first)
	s.sortScansByTime(allScans)

	// Paginate results
	page, nextCursor := s.paginateScans(allScans, cursorData, limit)

	return page, nextCursor, len(allScans), nil
}

// normalizeLimit validates and normalizes the limit parameter
func (s *LocalScanStore) normalizeLimit(limit int) int {
	if limit <= 0 {
		return 50 // Default
	}
	if limit > 100 {
		return 100 // Max
	}
	return limit
}

// loadFilteredScans loads all scans for an org and applies filters
func (s *LocalScanStore) loadFilteredScans(ctx context.Context, orgID string, filter ScanFilter) ([]*ScanMetadata, error) {
	orgDir := filepath.Join(s.root, orgID)

	// Check if org directory exists
	if _, err := os.Stat(orgDir); os.IsNotExist(err) {
		return []*ScanMetadata{}, nil
	}

	// Read all scan directories
	entries, err := os.ReadDir(orgDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read org directory: %w", err)
	}

	// Collect matching scans
	var scans []*ScanMetadata
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metadata, err := s.Get(ctx, orgID, entry.Name())
		if err != nil {
			continue // Skip invalid metadata
		}

		if s.matchesFilter(metadata, filter) {
			scans = append(scans, metadata)
		}
	}

	return scans, nil
}

// matchesFilter checks if a scan matches the given filter
func (s *LocalScanStore) matchesFilter(metadata *ScanMetadata, filter ScanFilter) bool {
	if filter.Status != "" && string(metadata.Status) != filter.Status {
		return false
	}
	if filter.Target != "" && !strings.Contains(metadata.Target, filter.Target) {
		return false
	}
	return true
}

// sortScansByTime sorts scans by start time (newest first)
func (s *LocalScanStore) sortScansByTime(scans []*ScanMetadata) {
	sort.Slice(scans, func(i, j int) bool {
		return scans[i].StartedAt.After(scans[j].StartedAt)
	})
}

// paginateScans applies cursor-based pagination to scan list
func (s *LocalScanStore) paginateScans(scans []*ScanMetadata, cursorData *Cursor, limit int) ([]*ScanMetadata, string) {
	// Find start index from cursor
	startIdx := s.findCursorPosition(scans, cursorData)

	// Calculate page boundaries
	endIdx := startIdx + limit
	if endIdx > len(scans) {
		endIdx = len(scans)
	}

	page := scans[startIdx:endIdx]

	// Generate next cursor
	var nextCursor string
	if endIdx < len(scans) && len(page) > 0 {
		lastScan := page[len(page)-1]
		nextCursor = EncodeCursor(&Cursor{
			LastScanID: lastScan.ID,
			LastTime:   lastScan.StartedAt.UnixNano(),
		})
	}

	return page, nextCursor
}

// findCursorPosition finds the starting index for pagination based on cursor
func (s *LocalScanStore) findCursorPosition(scans []*ScanMetadata, cursorData *Cursor) int {
	if cursorData == nil {
		return 0
	}

	for i, scan := range scans {
		if scan.ID == cursorData.LastScanID {
			return i + 1 // Start from next scan
		}
	}

	return 0 // Cursor not found, start from beginning
}

// Get retrieves metadata for a specific scan.
func (s *LocalScanStore) Get(ctx context.Context, orgID, scanID string) (*ScanMetadata, error) {
	metadataPath := s.metadataPath(orgID, scanID)

	// Check if metadata file exists
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return nil, NewNotFoundError("scan", scanID)
	}

	// Read metadata file with file lock
	lock := flock.New(metadataPath + ".lock")
	if err := lock.RLock(); err != nil {
		return nil, fmt.Errorf("failed to acquire read lock: %w", err)
	}
	defer func() { _ = lock.Unlock() }()

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata ScanMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &metadata, nil
}

// Create creates a new scan with the given metadata.
func (s *LocalScanStore) Create(ctx context.Context, orgID string, scan *ScanMetadata) error {
	if scan.ID == "" {
		return NewInvalidInputError("scan ID is required", "ID")
	}
	if scan.Target == "" {
		return NewInvalidInputError("scan target is required", "Target")
	}

	scanDir := s.scanDir(orgID, scan.ID)
	metadataPath := s.metadataPath(orgID, scan.ID)

	// Check if scan already exists
	if _, err := os.Stat(metadataPath); err == nil {
		return NewAlreadyExistsError("scan", scan.ID)
	}

	// Create scan directory
	if err := os.MkdirAll(scanDir, 0o755); err != nil {
		return fmt.Errorf("failed to create scan directory: %w", err)
	}

	// Set timestamps
	now := time.Now()
	if scan.CreatedAt.IsZero() {
		scan.CreatedAt = now
	}
	if scan.UpdatedAt.IsZero() {
		scan.UpdatedAt = now
	}
	if scan.OrgID == "" {
		scan.OrgID = orgID
	}

	// Write metadata with file lock
	lock := flock.New(metadataPath + ".lock")
	if err := lock.Lock(); err != nil {
		return fmt.Errorf("failed to acquire write lock: %w", err)
	}
	defer func() { _ = lock.Unlock() }()

	data, err := json.MarshalIndent(scan, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// Update updates metadata for an existing scan.
func (s *LocalScanStore) Update(ctx context.Context, orgID, scanID string, updates ScanUpdates) error {
	metadataPath := s.metadataPath(orgID, scanID)

	// Check if metadata file exists
	if _, err := os.Stat(metadataPath); os.IsNotExist(err) {
		return NewNotFoundError("scan", scanID)
	}

	// Lock metadata file for update
	lock := flock.New(metadataPath + ".lock")
	if err := lock.Lock(); err != nil {
		return fmt.Errorf("failed to acquire write lock: %w", err)
	}
	defer func() { _ = lock.Unlock() }()

	// Read current metadata (without lock since we already have it)
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %w", err)
	}

	var metadata ScanMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return fmt.Errorf("failed to parse metadata: %w", err)
	}

	// Apply updates (only non-nil fields)
	if updates.Status != nil {
		metadata.Status = *updates.Status
	}
	if updates.CompletedAt != nil {
		metadata.CompletedAt = *updates.CompletedAt
	}
	if updates.Duration != nil {
		metadata.Duration = *updates.Duration
	}
	if updates.HostCount != nil {
		metadata.HostCount = *updates.HostCount
	}
	if updates.ServiceCount != nil {
		metadata.ServiceCount = *updates.ServiceCount
	}
	if updates.VulnCount != nil {
		metadata.VulnCount = *updates.VulnCount
	}
	if updates.ErrorMessage != nil {
		metadata.ErrorMessage = *updates.ErrorMessage
	}

	// Update timestamp
	metadata.UpdatedAt = time.Now()

	// Write updated metadata
	data, err = json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// Delete removes a scan and all its associated data.
func (s *LocalScanStore) Delete(ctx context.Context, orgID, scanID string) error {
	scanDir := s.scanDir(orgID, scanID)

	// Check if scan exists
	if _, err := os.Stat(scanDir); os.IsNotExist(err) {
		return NewNotFoundError("scan", scanID)
	}

	// Remove entire scan directory
	if err := os.RemoveAll(scanDir); err != nil {
		return fmt.Errorf("failed to delete scan directory: %w", err)
	}

	// Remove lock file if it exists
	lockPath := s.metadataPath(orgID, scanID) + ".lock"
	_ = os.Remove(lockPath) // Ignore error

	return nil
}

// ReadData opens a data file for reading.
func (s *LocalScanStore) ReadData(ctx context.Context, orgID, scanID string, dataType DataType) (io.ReadCloser, error) {
	if !dataType.IsValid() {
		return nil, NewInvalidInputError(fmt.Sprintf("invalid data type: %s", dataType), "dataType")
	}

	dataPath := s.dataPath(orgID, scanID, dataType)

	// Check if file exists
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		return nil, NewNotFoundError("data file", string(dataType))
	}

	// Open file for reading
	file, err := os.Open(dataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open data file: %w", err)
	}

	return file, nil
}

// WriteData writes data to a file, replacing any existing content.
func (s *LocalScanStore) WriteData(ctx context.Context, orgID, scanID string, dataType DataType, data io.Reader) error {
	if !dataType.IsValid() {
		return NewInvalidInputError(fmt.Sprintf("invalid data type: %s", dataType), "dataType")
	}

	dataPath := s.dataPath(orgID, scanID, dataType)

	// Ensure scan directory exists
	scanDir := s.scanDir(orgID, scanID)
	if err := os.MkdirAll(scanDir, 0o755); err != nil {
		return fmt.Errorf("failed to create scan directory: %w", err)
	}

	// Create or truncate file
	file, err := os.Create(dataPath)
	if err != nil {
		return fmt.Errorf("failed to create data file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Copy data
	if _, err := io.Copy(file, data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}

// AppendData appends data to an existing file.
func (s *LocalScanStore) AppendData(ctx context.Context, orgID, scanID string, dataType DataType, data []byte) error {
	if !dataType.IsValid() {
		return NewInvalidInputError(fmt.Sprintf("invalid data type: %s", dataType), "dataType")
	}

	dataPath := s.dataPath(orgID, scanID, dataType)

	// Ensure scan directory exists
	scanDir := s.scanDir(orgID, scanID)
	if err := os.MkdirAll(scanDir, 0o755); err != nil {
		return fmt.Errorf("failed to create scan directory: %w", err)
	}

	// Use file lock for concurrent append safety
	lock := flock.New(dataPath + ".lock")
	if err := lock.Lock(); err != nil {
		return fmt.Errorf("failed to acquire write lock: %w", err)
	}
	defer func() { _ = lock.Unlock() }()

	// Open file for append
	file, err := os.OpenFile(dataPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open data file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Write data
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("failed to append data: %w", err)
	}

	return nil
}

// GetAnalytics returns ErrNotSupported for OSS edition.
func (s *LocalScanStore) GetAnalytics(ctx context.Context, orgID string, period TimePeriod) (*Analytics, error) {
	return nil, ErrNotSupported
}

// Helper methods

func (s *LocalScanStore) scanDir(orgID, scanID string) string {
	return filepath.Join(s.root, orgID, scanID)
}

func (s *LocalScanStore) metadataPath(orgID, scanID string) string {
	return filepath.Join(s.scanDir(orgID, scanID), "metadata.json")
}

func (s *LocalScanStore) dataPath(orgID, scanID string, dataType DataType) string {
	return filepath.Join(s.scanDir(orgID, scanID), string(dataType))
}

package storage

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// GCOptions defines options for garbage collection.
type GCOptions struct {
	// DryRun performs a dry run without actually deleting scans.
	// When true, returns the list of scans that would be deleted.
	DryRun bool

	// OrgID specifies which organization to clean up.
	// If empty, cleans up all organizations.
	OrgID string

	// Retention overrides the storage backend's configured retention policy.
	// If nil, uses the backend's default retention config.
	Retention *RetentionConfig
}

// GCResult contains the results of a garbage collection operation.
type GCResult struct {
	// ScansDeleted is the number of scans deleted.
	ScansDeleted int

	// DeletedScanIDs is the list of scan IDs that were deleted.
	DeletedScanIDs []string

	// BytesFreed is the approximate number of bytes freed (if available).
	BytesFreed int64

	// Errors contains any errors encountered during deletion.
	// GC continues even if individual deletions fail.
	Errors []error
}

// GarbageCollect performs garbage collection on scans based on retention policies.
//
// This function deletes scans that violate the configured retention policies:
//   - Scans older than MaxAgeDays
//   - Scans exceeding MaxScans count (oldest deleted first)
//
// The function operates per-organization. If opts.OrgID is empty, it processes
// all organizations.
//
// Returns:
//   - GCResult with deletion statistics
//   - error if GC operation fails (individual deletion errors are in GCResult.Errors)
func (b *LocalBackend) GarbageCollect(ctx context.Context, opts GCOptions) (*GCResult, error) {
	// Determine which retention policy to use
	retention := b.cfg.Retention
	if opts.Retention != nil {
		retention = *opts.Retention
	}

	// If no retention policy is enabled, nothing to do
	if !retention.IsEnabled() {
		return &GCResult{}, nil
	}

	result := &GCResult{
		DeletedScanIDs: make([]string, 0),
		Errors:         make([]error, 0),
	}

	// Determine which orgs to process
	orgs := []string{opts.OrgID}
	if opts.OrgID == "" {
		// Process all orgs (for now, just "default" since we don't have multi-tenancy yet)
		orgs = []string{"default"}
	}

	for _, orgID := range orgs {
		if err := b.gcOrganization(ctx, orgID, retention, opts.DryRun, result); err != nil {
			return result, fmt.Errorf("gc org %s: %w", orgID, err)
		}
	}

	return result, nil
}

// gcOrganization performs GC for a single organization.
func (b *LocalBackend) gcOrganization(ctx context.Context, orgID string, retention RetentionConfig, dryRun bool, result *GCResult) error {
	// List all scans for this org
	scans, err := b.Scans().List(ctx, orgID, ScanFilter{
		Limit: 10000, // Large limit to get all scans
	})
	if err != nil {
		return fmt.Errorf("list scans: %w", err)
	}

	if len(scans) == 0 {
		return nil
	}

	// Sort scans by start time (oldest first)
	sort.Slice(scans, func(i, j int) bool {
		return scans[i].StartedAt.Before(scans[j].StartedAt)
	})

	// Calculate cutoff time for age-based retention
	var ageCutoff time.Time
	if retention.MaxAgeDays > 0 {
		ageCutoff = time.Now().AddDate(0, 0, -retention.MaxAgeDays)
	}

	// Determine which scans to delete
	toDelete := make([]string, 0)

	// Phase 1: Delete scans older than MaxAgeDays
	if retention.MaxAgeDays > 0 {
		for _, scan := range scans {
			if scan.StartedAt.Before(ageCutoff) {
				toDelete = append(toDelete, scan.ID)
			}
		}
	}

	// Phase 2: If we still exceed MaxScans, delete oldest scans
	if retention.MaxScans > 0 {
		// Filter out already-marked scans
		remaining := make([]*ScanMetadata, 0)
		for _, scan := range scans {
			// Check if already marked for deletion
			markedForDeletion := false
			for _, id := range toDelete {
				if scan.ID == id {
					markedForDeletion = true
					break
				}
			}
			if !markedForDeletion {
				remaining = append(remaining, scan)
			}
		}

		// If remaining count exceeds MaxScans, delete oldest
		if len(remaining) > retention.MaxScans {
			excessCount := len(remaining) - retention.MaxScans
			for i := 0; i < excessCount; i++ {
				toDelete = append(toDelete, remaining[i].ID)
			}
		}
	}

	// Perform deletions
	for _, scanID := range toDelete {
		if dryRun {
			// Dry run: just record what would be deleted
			result.DeletedScanIDs = append(result.DeletedScanIDs, scanID)
			result.ScansDeleted++
		} else {
			// Actually delete the scan
			if err := b.Scans().Delete(ctx, orgID, scanID); err != nil {
				// Record error but continue with other deletions
				result.Errors = append(result.Errors, fmt.Errorf("delete scan %s: %w", scanID, err))
			} else {
				result.DeletedScanIDs = append(result.DeletedScanIDs, scanID)
				result.ScansDeleted++
			}
		}
	}

	return nil
}

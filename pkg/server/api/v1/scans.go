package v1

import (
	"context"
	"errors"
	"net/http"

	"github.com/pentora-ai/pentora/pkg/server/api"
	"github.com/pentora-ai/pentora/pkg/storage"
)

// ListScansHandler handles GET /api/v1/scans
//
// Returns a JSON array of scan metadata (id, status, start time, target count).
// This is a lightweight endpoint for listing scans without full details.
//
// Response format:
//
//	[
//	  {"id": "scan-1", "status": "completed", "start_time": "2024-01-01T00:00:00Z", "targets": 10},
//	  {"id": "scan-2", "status": "running", "start_time": "2024-01-02T00:00:00Z", "targets": 5}
//	]
func ListScansHandler(deps *api.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var scans []api.ScanMetadata
		var err error

		// Try new storage backend first, fall back to workspace
		if deps.Storage != nil {
			scans, err = listScansFromStorage(r.Context(), deps.Storage)
		} else if deps.Workspace != nil {
			scans, err = deps.Workspace.ListScans()
		} else {
			err = errors.New("no storage backend configured")
			api.WriteError(w, r, err)
			return
		}

		if err != nil {
			api.WriteError(w, r, err)
			return
		}

		api.WriteJSON(w, http.StatusOK, scans)
	}
}

// GetScanHandler handles GET /api/v1/scans/{id}
//
// Returns full scan details including results for a specific scan ID.
//
// Path parameter:
//   - id: Scan identifier
//
// Response format:
//
//	{
//	  "id": "scan-1",
//	  "status": "completed",
//	  "start_time": "2024-01-01T00:00:00Z",
//	  "end_time": "2024-01-01T00:05:00Z",
//	  "results": {
//	    "hosts_found": 10,
//	    "ports_open": 25,
//	    "vulnerabilities": []
//	  }
//	}
//
// Returns 404 if scan not found.
func GetScanHandler(deps *api.Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")

		var scan *api.ScanDetail
		var err error

		// Try new storage backend first, fall back to workspace
		if deps.Storage != nil {
			scan, err = getScanFromStorage(r.Context(), deps.Storage, id)
		} else if deps.Workspace != nil {
			scan, err = deps.Workspace.GetScan(id)
		} else {
			err = errors.New("no storage backend configured")
			api.WriteError(w, r, err)
			return
		}

		if err != nil {
			api.WriteError(w, r, err)
			return
		}

		api.WriteJSON(w, http.StatusOK, scan)
	}
}

// listScansFromStorage converts storage scan metadata to API format
func listScansFromStorage(ctx context.Context, backend storage.Backend) ([]api.ScanMetadata, error) {
	// Get all scans from storage (orgID="default" for OSS)
	storageScans, err := backend.Scans().List(ctx, "default", storage.ScanFilter{})
	if err != nil {
		return nil, err
	}

	// Convert to API format
	apiScans := make([]api.ScanMetadata, 0, len(storageScans))
	for _, s := range storageScans {
		apiScans = append(apiScans, api.ScanMetadata{
			ID:        s.ID,
			StartTime: s.StartedAt.Format("2006-01-02T15:04:05Z"),
			Status:    s.Status,
			Targets:   1, // TODO: Calculate from target string (e.g., CIDR range)
		})
	}

	return apiScans, nil
}

// getScanFromStorage retrieves scan details from storage and converts to API format
func getScanFromStorage(ctx context.Context, backend storage.Backend, scanID string) (*api.ScanDetail, error) {
	// Get scan metadata
	metadata, err := backend.Scans().Get(ctx, "default", scanID)
	if err != nil {
		return nil, err
	}

	// Build results map
	results := map[string]interface{}{
		"hosts_found":      metadata.HostCount,
		"services_found":   metadata.ServiceCount,
		"vulnerabilities":  metadata.VulnCount.Total(),
		"vuln_critical":    metadata.VulnCount.Critical,
		"vuln_high":        metadata.VulnCount.High,
		"vuln_medium":      metadata.VulnCount.Medium,
		"vuln_low":         metadata.VulnCount.Low,
		"vuln_info":        metadata.VulnCount.Info,
		"duration_seconds": metadata.Duration,
		"storage_location": metadata.StorageLocation,
	}

	// Add error message if scan failed
	if metadata.ErrorMessage != "" {
		results["error"] = metadata.ErrorMessage
	}

	// Convert to API format
	detail := &api.ScanDetail{
		ID:        metadata.ID,
		StartTime: metadata.StartedAt.Format("2006-01-02T15:04:05Z"),
		Status:    metadata.Status,
		Results:   results,
	}

	// Add end time if scan completed
	if !metadata.CompletedAt.IsZero() {
		detail.EndTime = metadata.CompletedAt.Format("2006-01-02T15:04:05Z")
	}

	return detail, nil
}

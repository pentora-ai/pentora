package v1

import (
	"encoding/json"
	"net/http"

	"github.com/pentora-ai/pentora/pkg/server/api"
	"github.com/rs/zerolog/log"
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
		scans, err := deps.Workspace.ListScans()
		if err != nil {
			log.Error().
				Str("component", "api").
				Err(err).
				Msg("Failed to list scans")
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(scans); err != nil {
			log.Error().
				Str("component", "api").
				Err(err).
				Msg("Failed to encode response")
		}
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

		scan, err := deps.Workspace.GetScan(id)
		if err != nil {
			log.Error().
				Str("component", "api").
				Str("scan_id", id).
				Err(err).
				Msg("Failed to get scan")
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(scan); err != nil {
			log.Error().
				Str("component", "api").
				Err(err).
				Msg("Failed to encode response")
		}
	}
}

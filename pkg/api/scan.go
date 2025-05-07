package api

import (
	"encoding/json"
	"net/http"

	"github.com/pentoraai/pentora/pkg/scanner"
)

// ScanRequest represents the input JSON structure for a scan job
type ScanRequest struct {
	Targets []string `json:"targets"`
	Ports   []int    `json:"ports"`
}

// ScanHandler handles POST /api/scan requests and triggers a scan
func ScanHandler(w http.ResponseWriter, r *http.Request) {
	var req ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	job := scanner.ScanJob{
		Targets: req.Targets,
		Ports:   req.Ports,
	}

	results, err := scanner.Run(job)
	if err != nil {
		http.Error(w, "Scan error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(results); err != nil {
		http.Error(w, "Failed to encode response: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

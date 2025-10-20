package api

import (
	"sync/atomic"
)

// Deps holds dependencies for API handlers.
// This pattern enables dependency injection and easier testing.
type Deps struct {
	// Workspace provides access to scan data
	Workspace WorkspaceInterface

	// Ready flag for readiness check
	Ready *atomic.Bool
}

// WorkspaceInterface is the subset of workspace methods needed by the API.
// Defined here to avoid circular dependencies and ease mocking.
type WorkspaceInterface interface {
	ListScans() ([]ScanMetadata, error)
	GetScan(id string) (*ScanDetail, error)
}

// ScanMetadata represents scan list item
type ScanMetadata struct {
	ID        string `json:"id"`
	StartTime string `json:"start_time"`
	Status    string `json:"status"`
	Targets   int    `json:"targets"`
}

// ScanDetail represents full scan details
type ScanDetail struct {
	ID        string                 `json:"id"`
	StartTime string                 `json:"start_time"`
	EndTime   string                 `json:"end_time,omitempty"`
	Status    string                 `json:"status"`
	Results   map[string]interface{} `json:"results"`
}

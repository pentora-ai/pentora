package api

import (
	"net/http"
)

// NewRouter returns an http.Handler that routes all API endpoints
func NewRouter() http.Handler {
	mux := http.NewServeMux()

	// Register API routes
	mux.HandleFunc("/api/scan", ScanHandler)

	return mux
}

package httpx

import (
	"net/http"

	"github.com/pentora-ai/pentora/pkg/config"
)

// NewRouter creates and configures the main HTTP router.
// It mounts health endpoints, API, and UI based on the configuration.
//
// The router uses Go 1.22+ enhanced pattern matching for cleaner routes.
// Routes are mounted conditionally based on cfg.APIEnabled and cfg.UIEnabled.
//
// Health endpoints are always enabled for liveness/readiness checks.
func NewRouter(cfg config.ServerConfig) *http.ServeMux {
	mux := http.NewServeMux()

	// Health endpoints (always enabled)
	mux.HandleFunc("GET /healthz", HealthzHandler)

	// Note: /readyz will be wired in Phase 3 with Ready atomic.Bool

	return mux
}

// HealthzHandler responds with 200 OK if the server process is alive.
// This endpoint is used by load balancers and orchestrators for liveness checks.
//
// It does not check dependencies (database, jobs, etc.) - just process health.
// For comprehensive readiness checks, use /readyz instead.
func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

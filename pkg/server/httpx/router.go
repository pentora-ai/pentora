package httpx

import (
	"net/http"

	"github.com/pentora-ai/pentora/pkg/config"
	"github.com/pentora-ai/pentora/pkg/server/api"
	v1 "github.com/pentora-ai/pentora/pkg/server/api/v1"
)

// NewRouter creates and configures the main HTTP router.
// It mounts health endpoints, API, and UI based on the configuration.
//
// The router uses Go 1.22+ enhanced pattern matching for cleaner routes.
// Routes are mounted conditionally based on cfg.APIEnabled and cfg.UIEnabled.
//
// Health endpoints are always enabled for liveness/readiness checks.
func NewRouter(cfg config.ServerConfig, deps *api.Deps) *http.ServeMux {
	mux := http.NewServeMux()

	// Health endpoints (always enabled)
	mux.HandleFunc("GET /healthz", HealthzHandler)
	mux.HandleFunc("GET /readyz", v1.ReadyzHandler(deps.Ready))

	// API endpoints (conditional)
	if cfg.APIEnabled {
		mux.HandleFunc("GET /api/v1/scans", v1.ListScansHandler(deps))
		mux.HandleFunc("GET /api/v1/scans/{id}", v1.GetScanHandler(deps))
	}

	// TODO: UI static serving will be added in Phase 4

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

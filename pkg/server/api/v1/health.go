package v1

import (
	"net/http"
	"sync/atomic"
)

// ReadyzHandler returns 200 when server is ready, 503 otherwise.
//
// This is used by orchestrators (Kubernetes, Docker, etc.) for readiness checks.
// Unlike /healthz (liveness), this checks if the server is fully initialized
// and ready to accept traffic.
//
// The ready flag is set by the app runtime after:
// - HTTP server starts
// - Background jobs start
// - All components initialize successfully
func ReadyzHandler(ready *atomic.Bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ready.Load() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Ready"))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("Not Ready"))
		}
	}
}

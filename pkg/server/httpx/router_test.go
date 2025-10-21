package httpx

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/pentora-ai/pentora/pkg/config"
	"github.com/pentora-ai/pentora/pkg/server/api"
	"github.com/stretchr/testify/require"
)

func TestNewRouter(t *testing.T) {
	cfg := config.DefaultServerConfig()
	deps := &api.Deps{
		Ready: &atomic.Bool{},
	}
	router := NewRouter(cfg, deps)

	require.NotNil(t, router)
}

func TestNewRouter_HealthzMounted(t *testing.T) {
	cfg := config.DefaultServerConfig()
	deps := &api.Deps{
		Ready: &atomic.Bool{},
	}
	router := NewRouter(cfg, deps)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "OK", w.Body.String())
}

func TestHealthzHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	HealthzHandler(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "OK", w.Body.String())
}

func TestHealthzHandler_AlwaysReturnsOK(t *testing.T) {
	// Test multiple calls to ensure idempotency
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()

		HealthzHandler(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "OK", w.Body.String())
	}
}

func TestHealthzHandler_IgnoresRequestBody(t *testing.T) {
	// Health check should work regardless of request body
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	HealthzHandler(w, req)

	require.Equal(t, http.StatusOK, w.Code)
}

package v1

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadyzHandler_NotReady(t *testing.T) {
	ready := &atomic.Bool{}
	ready.Store(false)

	handler := ReadyzHandler(ready)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusServiceUnavailable, w.Code)
	require.Equal(t, "Not Ready", w.Body.String())
}

func TestReadyzHandler_Ready(t *testing.T) {
	ready := &atomic.Bool{}
	ready.Store(true)

	handler := ReadyzHandler(ready)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "Ready", w.Body.String())
}

func TestReadyzHandler_TransitionToReady(t *testing.T) {
	ready := &atomic.Bool{}
	ready.Store(false)

	handler := ReadyzHandler(ready)

	// First call - not ready
	req1 := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)
	require.Equal(t, http.StatusServiceUnavailable, w1.Code)

	// Transition to ready
	ready.Store(true)

	// Second call - now ready
	req2 := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)
	require.Equal(t, "Ready", w2.Body.String())
}

func TestReadyzHandler_MultipleReads(t *testing.T) {
	ready := &atomic.Bool{}
	ready.Store(true)

	handler := ReadyzHandler(ready)

	// Multiple concurrent calls should all succeed
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "Ready", w.Body.String())
	}
}

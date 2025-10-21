package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pentora-ai/pentora/pkg/storage"
	"github.com/stretchr/testify/require"
)

func TestWriteError_NotFound(t *testing.T) {
	notFoundErr := &storage.NotFoundError{
		ResourceType: "scan",
		ResourceID:   "scan-123",
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scans/scan-123", nil)
	w := httptest.NewRecorder()

	WriteError(w, req, notFoundErr)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, "Not Found", response.Error)
	require.Contains(t, response.Message, "scan-123")
}

func TestWriteError_InternalServerError(t *testing.T) {
	genericErr := errors.New("database connection failed")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scans", nil)
	w := httptest.NewRecorder()

	WriteError(w, req, genericErr)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, "Internal Server Error", response.Error)
	require.Equal(t, "database connection failed", response.Message)
}

func TestWriteJSONError(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSONError(w, http.StatusBadRequest, "Invalid Input", "Target parameter is required")

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, "Invalid Input", response.Error)
	require.Equal(t, "Target parameter is required", response.Message)
}

func TestWriteJSON_Success(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]interface{}{
		"id":     "scan-1",
		"status": "completed",
	}

	WriteJSON(w, http.StatusOK, data)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, "scan-1", response["id"])
	require.Equal(t, "completed", response["status"])
}

func TestWriteJSON_Array(t *testing.T) {
	w := httptest.NewRecorder()

	data := []ScanMetadata{
		{ID: "scan-1", Status: "completed", StartTime: "2024-01-01T00:00:00Z", Targets: 10},
		{ID: "scan-2", Status: "running", StartTime: "2024-01-02T00:00:00Z", Targets: 5},
	}

	WriteJSON(w, http.StatusOK, data)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response []ScanMetadata
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	require.Len(t, response, 2)
	require.Equal(t, "scan-1", response[0].ID)
	require.Equal(t, "scan-2", response[1].ID)
}

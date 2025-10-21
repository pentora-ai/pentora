package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pentora-ai/pentora/pkg/server/api"
	"github.com/pentora-ai/pentora/pkg/storage"
	"github.com/stretchr/testify/require"
)

// Mock workspace for testing
type mockWorkspace struct {
	scans      []api.ScanMetadata
	scanDetail map[string]*api.ScanDetail
	listError  error
	getError   error
}

func (m *mockWorkspace) ListScans() ([]api.ScanMetadata, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	return m.scans, nil
}

func (m *mockWorkspace) GetScan(id string) (*api.ScanDetail, error) {
	if m.getError != nil {
		return nil, m.getError
	}
	if detail, ok := m.scanDetail[id]; ok {
		return detail, nil
	}
	// Return storage.NotFoundError so handler correctly returns 404
	return nil, &storage.NotFoundError{
		ResourceType: "scan",
		ResourceID:   id,
	}
}

func TestListScansHandler_Success(t *testing.T) {
	mockWs := &mockWorkspace{
		scans: []api.ScanMetadata{
			{ID: "scan-1", Status: "completed", StartTime: "2024-01-01T00:00:00Z", Targets: 10},
			{ID: "scan-2", Status: "running", StartTime: "2024-01-02T00:00:00Z", Targets: 5},
		},
	}

	deps := &api.Deps{
		Workspace: mockWs,
	}

	handler := ListScansHandler(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scans", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var scans []api.ScanMetadata
	err := json.NewDecoder(w.Body).Decode(&scans)
	require.NoError(t, err)
	require.Len(t, scans, 2)
	require.Equal(t, "scan-1", scans[0].ID)
	require.Equal(t, "completed", scans[0].Status)
	require.Equal(t, 10, scans[0].Targets)
}

func TestListScansHandler_EmptyList(t *testing.T) {
	mockWs := &mockWorkspace{
		scans: []api.ScanMetadata{},
	}

	deps := &api.Deps{
		Workspace: mockWs,
	}

	handler := ListScansHandler(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scans", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var scans []api.ScanMetadata
	err := json.NewDecoder(w.Body).Decode(&scans)
	require.NoError(t, err)
	require.Len(t, scans, 0)
}

func TestListScansHandler_WorkspaceError(t *testing.T) {
	mockWs := &mockWorkspace{
		listError: fmt.Errorf("workspace error"),
	}

	deps := &api.Deps{
		Workspace: mockWs,
	}

	handler := ListScansHandler(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scans", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "Internal Server Error")
}

func TestGetScanHandler_Success(t *testing.T) {
	mockWs := &mockWorkspace{
		scanDetail: map[string]*api.ScanDetail{
			"scan-1": {
				ID:        "scan-1",
				Status:    "completed",
				StartTime: "2024-01-01T00:00:00Z",
				EndTime:   "2024-01-01T00:05:00Z",
				Results: map[string]interface{}{
					"hosts_found": 10,
				},
			},
		},
	}

	deps := &api.Deps{
		Workspace: mockWs,
	}

	handler := GetScanHandler(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scans/scan-1", nil)
	req.SetPathValue("id", "scan-1")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var scan api.ScanDetail
	err := json.NewDecoder(w.Body).Decode(&scan)
	require.NoError(t, err)
	require.Equal(t, "scan-1", scan.ID)
	require.Equal(t, "completed", scan.Status)
	require.Equal(t, "2024-01-01T00:05:00Z", scan.EndTime)
	require.Equal(t, float64(10), scan.Results["hosts_found"])
}

func TestGetScanHandler_NotFound(t *testing.T) {
	mockWs := &mockWorkspace{
		scanDetail: map[string]*api.ScanDetail{},
	}

	deps := &api.Deps{
		Workspace: mockWs,
	}

	handler := GetScanHandler(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scans/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "Not Found")
}

func TestGetScanHandler_WorkspaceError(t *testing.T) {
	mockWs := &mockWorkspace{
		getError: fmt.Errorf("workspace error"),
	}

	deps := &api.Deps{
		Workspace: mockWs,
	}

	handler := GetScanHandler(deps)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/scans/scan-1", nil)
	req.SetPathValue("id", "scan-1")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Generic workspace errors should return 500, not 404
	// Only storage.NotFoundError returns 404
	require.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestGetScanHandler_DifferentIDs(t *testing.T) {
	mockWs := &mockWorkspace{
		scanDetail: map[string]*api.ScanDetail{
			"scan-1": {ID: "scan-1", Status: "completed"},
			"scan-2": {ID: "scan-2", Status: "running"},
		},
	}

	deps := &api.Deps{
		Workspace: mockWs,
	}

	handler := GetScanHandler(deps)

	// Test scan-1
	req1 := httptest.NewRequest(http.MethodGet, "/api/v1/scans/scan-1", nil)
	req1.SetPathValue("id", "scan-1")
	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	require.Equal(t, http.StatusOK, w1.Code)
	var scan1 api.ScanDetail
	err := json.NewDecoder(w1.Body).Decode(&scan1)
	require.NoError(t, err)
	require.Equal(t, "scan-1", scan1.ID)

	// Test scan-2
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/scans/scan-2", nil)
	req2.SetPathValue("id", "scan-2")
	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	require.Equal(t, http.StatusOK, w2.Code)
	var scan2 api.ScanDetail
	err = json.NewDecoder(w2.Body).Decode(&scan2)
	require.NoError(t, err)
	require.Equal(t, "scan-2", scan2.ID)
}

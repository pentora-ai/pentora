package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pentora-ai/pentora/pkg/plugin"
)

// mockPluginService implements PluginService for testing
type mockPluginService struct {
	installResult   *plugin.InstallResult
	installError    error
	listResult      []*plugin.PluginInfo
	listError       error
	getInfoResult   *plugin.PluginInfo
	getInfoError    error
	uninstallResult *plugin.UninstallResult
	uninstallError  error
	updateResult    *plugin.UpdateResult
	updateError     error
}

func (m *mockPluginService) Install(ctx context.Context, target string, opts plugin.InstallOptions) (*plugin.InstallResult, error) {
	if m.installError != nil {
		return nil, m.installError
	}
	return m.installResult, nil
}

func (m *mockPluginService) List(ctx context.Context) ([]*plugin.PluginInfo, error) {
	if m.listError != nil {
		return nil, m.listError
	}
	return m.listResult, nil
}

func (m *mockPluginService) GetInfo(ctx context.Context, id string) (*plugin.PluginInfo, error) {
	if m.getInfoError != nil {
		return nil, m.getInfoError
	}
	return m.getInfoResult, nil
}

func (m *mockPluginService) Uninstall(ctx context.Context, target string, opts plugin.UninstallOptions) (*plugin.UninstallResult, error) {
	if m.uninstallError != nil {
		return nil, m.uninstallError
	}
	return m.uninstallResult, nil
}

func (m *mockPluginService) Update(ctx context.Context, opts plugin.UpdateOptions) (*plugin.UpdateResult, error) {
	if m.updateError != nil {
		return nil, m.updateError
	}
	return m.updateResult, nil
}

// TestInstallPluginHandler_Success tests successful plugin installation
func TestInstallPluginHandler_Success(t *testing.T) {
	mockSvc := &mockPluginService{
		installResult: &plugin.InstallResult{
			InstalledCount: 1,
			SkippedCount:   0,
			FailedCount:    0,
			Plugins: []*plugin.PluginInfo{
				{
					ID:       "ssh-weak-cipher",
					Name:     "SSH Weak Cipher Detection",
					Version:  "1.0.0",
					Type:     "evaluation",
					Author:   "pentora-security",
					Severity: "high",
					Tags:     []string{"ssh", "crypto"},
				},
			},
			Errors: []plugin.PluginError{},
		},
	}

	handler := InstallPluginHandler(mockSvc)

	reqBody := InstallPluginRequest{
		Target: "ssh-weak-cipher",
		Force:  false,
		Source: "official",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/install", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp InstallPluginResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	require.Equal(t, 1, resp.InstalledCount)
	require.Equal(t, 0, resp.SkippedCount)
	require.Equal(t, 0, resp.FailedCount)
	require.Len(t, resp.Plugins, 1)
	require.Equal(t, "ssh-weak-cipher", resp.Plugins[0].ID)
	require.Equal(t, "SSH Weak Cipher Detection", resp.Plugins[0].Name)
	require.Equal(t, "1.0.0", resp.Plugins[0].Version)
	require.Empty(t, resp.Errors)
}

// TestInstallPluginHandler_InvalidJSON tests handler with malformed JSON
func TestInstallPluginHandler_InvalidJSON(t *testing.T) {
	mockSvc := &mockPluginService{}
	handler := InstallPluginHandler(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/install", bytes.NewReader([]byte(`{"target": invalid}`)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "INVALID_REQUEST_BODY")
}

// TestInstallPluginHandler_EmptyTarget tests handler with empty target
func TestInstallPluginHandler_EmptyTarget(t *testing.T) {
	mockSvc := &mockPluginService{}
	handler := InstallPluginHandler(mockSvc)

	reqBody := InstallPluginRequest{
		Target: "",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/install", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "TARGET_REQUIRED")
}

// TestInstallPluginHandler_PluginNotFound tests handler when plugin not found
func TestInstallPluginHandler_PluginNotFound(t *testing.T) {
	mockSvc := &mockPluginService{
		installError: plugin.ErrPluginNotFound,
	}
	handler := InstallPluginHandler(mockSvc)

	reqBody := InstallPluginRequest{
		Target: "nonexistent-plugin",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/install", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "PLUGIN_NOT_FOUND")
}

// TestInstallPluginHandler_ServiceError tests handler with service error
func TestInstallPluginHandler_ServiceError(t *testing.T) {
	mockSvc := &mockPluginService{
		installError: errors.New("cache failure"),
	}
	handler := InstallPluginHandler(mockSvc)

	reqBody := InstallPluginRequest{
		Target: "ssh-weak-cipher",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/install", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "INTERNAL_ERROR")
}

// TestListPluginsHandler_EmptyList tests handler with no plugins installed
func TestListPluginsHandler_EmptyList(t *testing.T) {
	mockSvc := &mockPluginService{
		listResult: []*plugin.PluginInfo{},
	}
	handler := ListPluginsHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp PluginListResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	require.Equal(t, 0, resp.Count)
	require.Empty(t, resp.Plugins)
}

// TestListPluginsHandler_MultiplePlugins tests handler with multiple plugins
func TestListPluginsHandler_MultiplePlugins(t *testing.T) {
	mockSvc := &mockPluginService{
		listResult: []*plugin.PluginInfo{
			{
				ID:       "ssh-weak-cipher",
				Name:     "SSH Weak Cipher Detection",
				Version:  "1.0.0",
				Type:     "evaluation",
				Author:   "pentora-security",
				Severity: "high",
				Tags:     []string{"ssh", "crypto"},
			},
			{
				ID:       "http-default-pages",
				Name:     "HTTP Default Pages",
				Version:  "1.0.0",
				Type:     "evaluation",
				Author:   "pentora-security",
				Severity: "medium",
				Tags:     []string{"http", "web"},
			},
		},
	}
	handler := ListPluginsHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp PluginListResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	require.Equal(t, 2, resp.Count)
	require.Len(t, resp.Plugins, 2)
	require.Equal(t, "ssh-weak-cipher", resp.Plugins[0].ID)
	require.Equal(t, "http-default-pages", resp.Plugins[1].ID)
}

// TestListPluginsHandler_ServiceError tests handler with service error
func TestListPluginsHandler_ServiceError(t *testing.T) {
	mockSvc := &mockPluginService{
		listError: errors.New("database error"),
	}
	handler := ListPluginsHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "INTERNAL_ERROR")
}

// TestGetPluginHandler_Success tests successful plugin retrieval
func TestGetPluginHandler_Success(t *testing.T) {
	mockSvc := &mockPluginService{
		getInfoResult: &plugin.PluginInfo{
			ID:       "ssh-weak-cipher",
			Name:     "SSH Weak Cipher Detection",
			Version:  "1.0.0",
			Type:     "evaluation",
			Author:   "pentora-security",
			Severity: "high",
			Tags:     []string{"ssh", "crypto"},
		},
	}
	handler := GetPluginHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins/ssh-weak-cipher", nil)
	req.SetPathValue("id", "ssh-weak-cipher")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp PluginInfoDTO
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	require.Equal(t, "ssh-weak-cipher", resp.ID)
	require.Equal(t, "SSH Weak Cipher Detection", resp.Name)
	require.Equal(t, "1.0.0", resp.Version)
	require.Equal(t, "high", resp.Severity)
}

// TestGetPluginHandler_NotFound tests handler when plugin not found
func TestGetPluginHandler_NotFound(t *testing.T) {
	mockSvc := &mockPluginService{
		getInfoError: plugin.ErrPluginNotFound,
	}
	handler := GetPluginHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "PLUGIN_NOT_FOUND")
}

// TestGetPluginHandler_ServiceError tests handler with service error
func TestGetPluginHandler_ServiceError(t *testing.T) {
	mockSvc := &mockPluginService{
		getInfoError: errors.New("registry read error"),
	}
	handler := GetPluginHandler(mockSvc)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/plugins/ssh-weak-cipher", nil)
	req.SetPathValue("id", "ssh-weak-cipher")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "INTERNAL_ERROR")
}

// TestUninstallPluginHandler_Success tests successful plugin uninstallation
func TestUninstallPluginHandler_Success(t *testing.T) {
	mockSvc := &mockPluginService{
		uninstallResult: &plugin.UninstallResult{
			RemovedCount: 1,
			FailedCount:  0,
		},
	}
	handler := UninstallPluginHandler(mockSvc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/plugins/ssh-weak-cipher", nil)
	req.SetPathValue("id", "ssh-weak-cipher")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp map[string]interface{}
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	require.Equal(t, "plugin uninstalled successfully", resp["message"])
	require.Equal(t, float64(1), resp["removed_count"])
}

// TestUninstallPluginHandler_NotFound tests handler when plugin not found
func TestUninstallPluginHandler_NotFound(t *testing.T) {
	mockSvc := &mockPluginService{
		uninstallError: plugin.ErrPluginNotFound,
	}
	handler := UninstallPluginHandler(mockSvc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/plugins/nonexistent", nil)
	req.SetPathValue("id", "nonexistent")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Contains(t, w.Body.String(), "PLUGIN_NOT_FOUND")
}

// TestUninstallPluginHandler_ServiceError tests handler with service error
func TestUninstallPluginHandler_ServiceError(t *testing.T) {
	mockSvc := &mockPluginService{
		uninstallError: errors.New("filesystem error"),
	}
	handler := UninstallPluginHandler(mockSvc)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/plugins/ssh-weak-cipher", nil)
	req.SetPathValue("id", "ssh-weak-cipher")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "INTERNAL_ERROR")
}

// TestUpdatePluginsHandler_Success tests successful plugin update
func TestUpdatePluginsHandler_Success(t *testing.T) {
	mockSvc := &mockPluginService{
		updateResult: &plugin.UpdateResult{
			UpdatedCount: 2,
			SkippedCount: 1,
			FailedCount:  0,
			Plugins: []*plugin.PluginInfo{
				{
					ID:      "ssh-weak-cipher",
					Name:    "SSH Weak Cipher Detection",
					Version: "1.0.1",
					Author:  "pentora-security",
				},
				{
					ID:      "http-default-pages",
					Name:    "HTTP Default Pages",
					Version: "1.0.1",
					Author:  "pentora-security",
				},
			},
			Errors: []plugin.PluginError{},
		},
	}
	handler := UpdatePluginsHandler(mockSvc)

	reqBody := UpdatePluginsRequest{
		Category: "",
		Source:   "official",
		Force:    false,
		DryRun:   false,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/update", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp UpdatePluginsResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	require.Equal(t, 2, resp.UpdatedCount)
	require.Equal(t, 1, resp.SkippedCount)
	require.Equal(t, 0, resp.FailedCount)
	require.Len(t, resp.Plugins, 2)
	require.Empty(t, resp.Errors)
}

// TestUpdatePluginsHandler_WithCategoryFilter tests update with category filter
func TestUpdatePluginsHandler_WithCategoryFilter(t *testing.T) {
	mockSvc := &mockPluginService{
		updateResult: &plugin.UpdateResult{
			UpdatedCount: 1,
			SkippedCount: 0,
			FailedCount:  0,
			Plugins: []*plugin.PluginInfo{
				{
					ID:      "ssh-weak-cipher",
					Name:    "SSH Weak Cipher Detection",
					Version: "1.0.1",
				},
			},
			Errors: []plugin.PluginError{},
		},
	}
	handler := UpdatePluginsHandler(mockSvc)

	reqBody := UpdatePluginsRequest{
		Category: "ssh",
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/update", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp UpdatePluginsResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	require.Equal(t, 1, resp.UpdatedCount)
}

// TestUpdatePluginsHandler_EmptyBody tests update with empty request body
func TestUpdatePluginsHandler_EmptyBody(t *testing.T) {
	mockSvc := &mockPluginService{
		updateResult: &plugin.UpdateResult{
			UpdatedCount: 0,
			SkippedCount: 0,
			FailedCount:  0,
			Plugins:      []*plugin.PluginInfo{},
			Errors:       []plugin.PluginError{},
		},
	}
	handler := UpdatePluginsHandler(mockSvc)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/update", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)

	var resp UpdatePluginsResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	// Empty body is valid - updates all plugins
}

// TestUpdatePluginsHandler_ServiceError tests handler with service error
func TestUpdatePluginsHandler_ServiceError(t *testing.T) {
	mockSvc := &mockPluginService{
		updateError: errors.New("network error"),
	}
	handler := UpdatePluginsHandler(mockSvc)

	reqBody := UpdatePluginsRequest{}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/plugins/update", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	require.Equal(t, http.StatusInternalServerError, w.Code)
	require.Contains(t, w.Body.String(), "INTERNAL_ERROR")
}

package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/pentora-ai/pentora/pkg/server/api"
)

// formatSourceList formats a string slice as a comma-separated list.
// Helper function for generating user-friendly error messages.
func formatSourceList(items []string) string {
	return strings.Join(items, ", ")
}

// PluginService defines the interface for plugin operations.
// This allows for easy mocking in tests.
// This interface matches the pkg/plugin.Service methods.
type PluginService interface {
	Install(ctx context.Context, target string, opts plugin.InstallOptions) (*plugin.InstallResult, error)
	Update(ctx context.Context, opts plugin.UpdateOptions) (*plugin.UpdateResult, error)
	Uninstall(ctx context.Context, target string, opts plugin.UninstallOptions) (*plugin.UninstallResult, error)
	List(ctx context.Context) ([]*plugin.PluginInfo, error)
	GetInfo(ctx context.Context, id string) (*plugin.PluginInfo, error)
}

// InstallPluginRequest represents the request body for plugin installation
type InstallPluginRequest struct {
	// Target is the plugin ID or category to install
	Target string `json:"target"`

	// Force reinstall even if already installed
	Force bool `json:"force,omitempty"`

	// Source to download from (optional, defaults to all sources)
	Source string `json:"source,omitempty"`
}

// InstallPluginResponse represents the response for plugin installation
type InstallPluginResponse struct {
	// InstalledCount is the number of plugins successfully installed
	InstalledCount int `json:"installed_count"`

	// SkippedCount is the number of plugins skipped (already installed)
	SkippedCount int `json:"skipped_count"`

	// FailedCount is the number of plugins that failed to install
	FailedCount int `json:"failed_count"`

	// Plugins is the list of successfully installed plugins
	Plugins []PluginInfoDTO `json:"plugins"`

	// Errors contains detailed error information for failed plugins
	// Each error includes plugin ID, error message, error code, and actionable suggestion
	Errors []PluginErrorDTO `json:"errors,omitempty"`

	// PartialFailure indicates if some plugins succeeded while others failed
	PartialFailure bool `json:"partial_failure"`
}

// UpdatePluginsRequest represents the request body for plugin updates
type UpdatePluginsRequest struct {
	// Category filter (optional)
	Category string `json:"category,omitempty"`

	// Source filter (optional)
	Source string `json:"source,omitempty"`

	// Force re-download even if cached
	Force bool `json:"force,omitempty"`

	// DryRun simulates the update without actually downloading
	DryRun bool `json:"dry_run,omitempty"`
}

// UpdatePluginsResponse represents the response for plugin updates
type UpdatePluginsResponse struct {
	// UpdatedCount is the number of plugins downloaded
	UpdatedCount int `json:"updated_count"`

	// SkippedCount is the number of plugins skipped (already cached)
	SkippedCount int `json:"skipped_count"`

	// FailedCount is the number of plugins that failed to download
	FailedCount int `json:"failed_count"`

	// Plugins is the list of updated plugins
	Plugins []PluginInfoDTO `json:"plugins"`

	// Errors contains detailed error information for failed plugins
	// Each error includes plugin ID, error message, error code, and actionable suggestion
	Errors []PluginErrorDTO `json:"errors,omitempty"`

	// PartialFailure indicates if some plugins succeeded while others failed
	PartialFailure bool `json:"partial_failure"`
}

// PluginListResponse represents the response for listing plugins
type PluginListResponse struct {
	// Plugins is the list of installed plugins
	Plugins []PluginInfoDTO `json:"plugins"`

	// Count is the total number of plugins
	Count int `json:"count"`
}

// PluginErrorDTO represents a plugin error in API responses (ADR-0003)
type PluginErrorDTO struct {
	// PluginID is the unique identifier of the plugin that failed
	PluginID string `json:"plugin_id"`

	// Error is the human-readable error message
	Error string `json:"error"`

	// Code is the machine-readable error code (e.g., CHECKSUM_MISMATCH, SOURCE_NOT_AVAILABLE)
	Code string `json:"code"`

	// Suggestion is an actionable suggestion for resolving the error
	Suggestion string `json:"suggestion,omitempty"`
}

// PluginInfoDTO represents plugin information in API responses
type PluginInfoDTO struct {
	// ID is the unique plugin identifier
	ID string `json:"id"`

	// Name is the human-readable plugin name
	Name string `json:"name"`

	// Version is the plugin version
	Version string `json:"version"`

	// Type is the plugin type
	Type string `json:"type,omitempty"`

	// Author is the plugin author
	Author string `json:"author"`

	// Severity is the severity level (critical, high, medium, low)
	Severity string `json:"severity,omitempty"`

	// Tags are the plugin tags
	Tags []string `json:"tags,omitempty"`
}

// InstallPluginHandler handles POST /api/v1/plugins/install
//
// Installs one or more plugins by ID or category.
//
// Request body:
//
//	{
//	  "target": "ssh-weak-cipher",  // Plugin ID or category name
//	  "force": false,                 // Optional: force reinstall
//	  "source": "official"            // Optional: source filter
//	}
//
// Response format:
//
//	{
//	  "installed_count": 1,
//	  "skipped_count": 0,
//	  "failed_count": 0,
//	  "plugins": [{"id": "ssh-weak-cipher", "name": "...", ...}],
//	  "errors": []
//	}
//
// Returns 400 for invalid requests, 500 for server errors, 504 for timeout.
func InstallPluginHandler(pluginService PluginService, config api.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Setup structured logger with operation context
		logger := log.With().
			Str("component", "api.plugins").
			Str("op", "install").
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Logger()

		start := time.Now()
		var statusCode int
		defer func() {
			logger.Info().
				Int("status", statusCode).
				Dur("duration_ms", time.Since(start)).
				Msg("request completed")
		}()

		// Apply handler-level timeout (only if request context doesn't have deadline)
		ctx := r.Context()
		if _, hasDeadline := ctx.Deadline(); !hasDeadline && config.HandlerTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, config.HandlerTimeout)
			defer cancel()
		}

		// Defense-in-depth: Limit request body size (2MB)
		r.Body = http.MaxBytesReader(w, r.Body, plugin.MaxRequestBodySize)

		var req InstallPluginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			statusCode = http.StatusBadRequest
			logger.Error().
				Err(err).
				Str("error_code", "INVALID_REQUEST_BODY").
				Msg("failed to decode request")
			api.WriteJSONError(w, statusCode, "Bad Request", "INVALID_REQUEST_BODY", "invalid request body: "+err.Error())
			return
		}

		// Log request snapshot
		logger.Info().
			Str("target", req.Target).
			Str("source", req.Source).
			Bool("force", req.Force).
			Msg("install started")

		// Validate request - required fields
		if req.Target == "" {
			statusCode = http.StatusBadRequest
			logger.Error().
				Str("error_code", "TARGET_REQUIRED").
				Msg("validation failed: target required")
			api.WriteJSONError(w, statusCode, "Bad Request", "TARGET_REQUIRED", "target is required")
			return
		}

		// Validate source whitelist (API layer - defense-in-depth)
		if req.Source != "" && !plugin.IsValidSource(req.Source) {
			validSources := plugin.ValidSources
			statusCode = http.StatusBadRequest
			logger.Error().
				Str("error_code", "INVALID_SOURCE").
				Str("source", req.Source).
				Msg("validation failed: invalid source")
			api.WriteJSONError(w, statusCode, "Bad Request", "INVALID_SOURCE",
				"invalid source '"+req.Source+"' (valid: "+formatSourceList(validSources)+")")
			return
		}

		// Build install options
		opts := plugin.InstallOptions{
			Force:  req.Force,
			Source: req.Source,
		}

		// Call service with timeout context
		result, err := pluginService.Install(ctx, req.Target, opts)
		if err != nil {
			// Check if timeout occurred
			if ctx.Err() == context.DeadlineExceeded {
				statusCode = http.StatusGatewayTimeout
				logger.Error().
					Err(err).
					Str("error_code", "TIMEOUT").
					Msg("install failed: timeout")
				api.WriteJSONError(w, statusCode, "Gateway Timeout", "TIMEOUT",
					"operation timed out after "+config.HandlerTimeout.String())
				return
			}
			statusCode = plugin.HTTPStatus(err)
			logger.Error().
				Err(err).
				Str("error_code", plugin.ErrorCode(err)).
				Msg("install failed")
			api.WriteError(w, r, err)
			return
		}

		// Build response
		resp := InstallPluginResponse{
			InstalledCount: result.InstalledCount,
			SkippedCount:   result.SkippedCount,
			FailedCount:    result.FailedCount,
			Plugins:        make([]PluginInfoDTO, 0, len(result.Plugins)),
			Errors:         make([]PluginErrorDTO, 0, len(result.Errors)),
			PartialFailure: result.FailedCount > 0 && result.InstalledCount > 0,
		}

		// Convert plugins to DTO
		for _, p := range result.Plugins {
			resp.Plugins = append(resp.Plugins, PluginInfoDTO{
				ID:       p.ID,
				Name:     p.Name,
				Version:  p.Version,
				Type:     p.Type,
				Author:   p.Author,
				Severity: p.Severity,
				Tags:     p.Tags,
			})
		}

		// Convert errors to DTO
		for _, err := range result.Errors {
			resp.Errors = append(resp.Errors, PluginErrorDTO{
				PluginID:   err.PluginID,
				Error:      err.Error,
				Code:       err.Code,
				Suggestion: err.Suggestion,
			})
		}

		// Log success with result metrics
		statusCode = http.StatusOK
		logger.Info().
			Int("installed_count", result.InstalledCount).
			Int("skipped_count", result.SkippedCount).
			Int("failed_count", result.FailedCount).
			Bool("partial_failure", resp.PartialFailure).
			Msg("install succeeded")

		api.WriteJSON(w, statusCode, resp)
	}
}

// ListPluginsHandler handles GET /api/v1/plugins
//
// Returns a list of all installed plugins with their metadata.
//
// Response format:
//
//	{
//	  "plugins": [
//	    {
//	      "id": "ssh-weak-cipher",
//	      "name": "SSH Weak Cipher Detection",
//	      "version": "1.0.0",
//	      "author": "pentora-security",
//	      "severity": "high",
//	      "tags": ["ssh", "crypto"]
//	    }
//	  ],
//	  "count": 1
//	}
func ListPluginsHandler(pluginService PluginService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		plugins, err := pluginService.List(r.Context())
		if err != nil {
			api.WriteError(w, r, err)
			return
		}

		// Convert to DTO
		dtos := make([]PluginInfoDTO, 0, len(plugins))
		for _, p := range plugins {
			dtos = append(dtos, PluginInfoDTO{
				ID:       p.ID,
				Name:     p.Name,
				Version:  p.Version,
				Type:     p.Type,
				Author:   p.Author,
				Severity: p.Severity,
				Tags:     p.Tags,
			})
		}

		resp := PluginListResponse{
			Plugins: dtos,
			Count:   len(dtos),
		}

		api.WriteJSON(w, http.StatusOK, resp)
	}
}

// GetPluginHandler handles GET /api/v1/plugins/{id}
//
// Returns detailed information about a specific plugin.
//
// Path parameter:
//   - id: Plugin identifier
//
// Response format:
//
//	{
//	  "id": "ssh-weak-cipher",
//	  "name": "SSH Weak Cipher Detection",
//	  "version": "1.0.0",
//	  "author": "pentora-security",
//	  "severity": "high",
//	  "tags": ["ssh", "crypto"]
//	}
//
// Returns 404 if plugin not found.
func GetPluginHandler(pluginService PluginService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			api.WriteJSONError(w, http.StatusBadRequest, "Bad Request", "PLUGIN_ID_REQUIRED", "plugin id is required")
			return
		}

		info, err := pluginService.GetInfo(r.Context(), id)
		if err != nil {
			// Use WriteError which will automatically map plugin errors to correct HTTP status
			api.WriteError(w, r, err)
			return
		}

		// Convert to DTO
		dto := PluginInfoDTO{
			ID:       info.ID,
			Name:     info.Name,
			Version:  info.Version,
			Type:     info.Type,
			Author:   info.Author,
			Severity: info.Severity,
			Tags:     info.Tags,
		}

		api.WriteJSON(w, http.StatusOK, dto)
	}
}

// UninstallPluginHandler handles DELETE /api/v1/plugins/{id}
//
// Uninstalls a plugin by ID.
//
// Path parameter:
//   - id: Plugin identifier
//
// Response format:
//
//	{
//	  "message": "plugin uninstalled successfully",
//	  "removed_count": 1
//	}
//
// Returns 404 if plugin not found, 500 for server errors, 504 for timeout.
func UninstallPluginHandler(pluginService PluginService, config api.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Apply handler-level timeout (only if request context doesn't have deadline)
		ctx := r.Context()
		if _, hasDeadline := ctx.Deadline(); !hasDeadline && config.HandlerTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, config.HandlerTimeout)
			defer cancel()
		}

		id := r.PathValue("id")
		if id == "" {
			api.WriteJSONError(w, http.StatusBadRequest, "Bad Request", "PLUGIN_ID_REQUIRED", "plugin id is required")
			return
		}

		// Build uninstall options (single plugin)
		opts := plugin.UninstallOptions{
			All: false,
		}

		result, err := pluginService.Uninstall(ctx, id, opts)
		if err != nil {
			// Check if timeout occurred
			if ctx.Err() == context.DeadlineExceeded {
				api.WriteJSONError(w, http.StatusGatewayTimeout, "Gateway Timeout", "TIMEOUT",
					"operation timed out after "+config.HandlerTimeout.String())
				return
			}
			// Use WriteError which will automatically map plugin errors to correct HTTP status
			api.WriteError(w, r, err)
			return
		}

		api.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"message":       "plugin uninstalled successfully",
			"removed_count": result.RemovedCount,
		})
	}
}

// UpdatePluginsHandler handles POST /api/v1/plugins/update
//
// Updates plugins from remote sources, optionally filtered by category or source.
//
// Request body:
//
//	{
//	  "category": "ssh",      // Optional: filter by category
//	  "source": "official",    // Optional: filter by source
//	  "force": false,          // Optional: force re-download
//	  "dry_run": false         // Optional: simulate without downloading
//	}
//
// Response format:
//
//	{
//	  "updated_count": 5,
//	  "skipped_count": 3,
//	  "failed_count": 0,
//	  "plugins": [{"id": "ssh-weak-cipher", "name": "...", ...}],
//	  "errors": []
//	}
func UpdatePluginsHandler(pluginService PluginService, config api.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Apply handler-level timeout (only if request context doesn't have deadline)
		ctx := r.Context()
		if _, hasDeadline := ctx.Deadline(); !hasDeadline && config.HandlerTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, config.HandlerTimeout)
			defer cancel()
		}

		// Defense-in-depth: Limit request body size (2MB)
		r.Body = http.MaxBytesReader(w, r.Body, plugin.MaxRequestBodySize)

		var req UpdatePluginsRequest
		// Empty body is OK for update (updates all plugins)
		_ = json.NewDecoder(r.Body).Decode(&req)

		// Validate category whitelist (API layer - defense-in-depth)
		if req.Category != "" && !plugin.IsValidCategory(req.Category) {
			validCategories := plugin.GetValidCategories()
			api.WriteJSONError(w, http.StatusBadRequest, "Bad Request", "INVALID_CATEGORY",
				"invalid category '"+req.Category+"' (valid: "+formatSourceList(validCategories)+")")
			return
		}

		// Validate source whitelist (API layer - defense-in-depth)
		if req.Source != "" && !plugin.IsValidSource(req.Source) {
			validSources := plugin.ValidSources
			api.WriteJSONError(w, http.StatusBadRequest, "Bad Request", "INVALID_SOURCE",
				"invalid source '"+req.Source+"' (valid: "+formatSourceList(validSources)+")")
			return
		}

		// Build update options
		opts := plugin.UpdateOptions{
			Source: req.Source,
			Force:  req.Force,
			DryRun: req.DryRun,
		}

		// Convert category string to Category type
		if req.Category != "" {
			opts.Category = plugin.Category(req.Category)
		}

		// Call service with timeout context
		result, err := pluginService.Update(ctx, opts)
		if err != nil {
			// Check if timeout occurred
			if ctx.Err() == context.DeadlineExceeded {
				api.WriteJSONError(w, http.StatusGatewayTimeout, "Gateway Timeout", "TIMEOUT",
					"operation timed out after "+config.HandlerTimeout.String())
				return
			}
			api.WriteError(w, r, err)
			return
		}

		// Build response
		resp := UpdatePluginsResponse{
			UpdatedCount:   result.UpdatedCount,
			SkippedCount:   result.SkippedCount,
			FailedCount:    result.FailedCount,
			Plugins:        make([]PluginInfoDTO, 0, len(result.Plugins)),
			Errors:         make([]PluginErrorDTO, 0, len(result.Errors)),
			PartialFailure: result.FailedCount > 0 && result.UpdatedCount > 0,
		}

		// Convert plugins to DTO
		for _, p := range result.Plugins {
			resp.Plugins = append(resp.Plugins, PluginInfoDTO{
				ID:       p.ID,
				Name:     p.Name,
				Version:  p.Version,
				Type:     p.Type,
				Author:   p.Author,
				Severity: p.Severity,
				Tags:     p.Tags,
			})
		}

		// Convert errors to DTO
		for _, err := range result.Errors {
			resp.Errors = append(resp.Errors, PluginErrorDTO{
				PluginID:   err.PluginID,
				Error:      err.Error,
				Code:       err.Code,
				Suggestion: err.Suggestion,
			})
		}

		api.WriteJSON(w, http.StatusOK, resp)
	}
}

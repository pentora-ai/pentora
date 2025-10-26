package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/pentora-ai/pentora/pkg/plugin"
	"github.com/pentora-ai/pentora/pkg/storage"
	"github.com/rs/zerolog/log"
)

// ErrorResponse represents a standard JSON error response.
// Used consistently across all API endpoints for error responses.
//
// Example:
//
//	{
//	  "error": "Not Found",
//	  "code": "PLUGIN_NOT_FOUND",
//	  "message": "Plugin 'ssh-weak-cipher' not found"
//	}
type ErrorResponse struct {
	Error   string `json:"error"`             // Short error type (e.g., "Not Found", "Internal Server Error")
	Code    string `json:"code,omitempty"`    // Machine-readable error code (e.g., "PLUGIN_NOT_FOUND", "INVALID_INPUT")
	Message string `json:"message,omitempty"` // Detailed error message (optional)
}

// WriteError writes a standard JSON error response to the client.
// It automatically determines the HTTP status code based on error type:
//   - Plugin errors (ErrPluginNotFound, ErrInvalidOption, etc.) → Mapped via plugin.HTTPStatus()
//   - storage.NotFoundError → 404 Not Found
//   - All other errors → 500 Internal Server Error
//
// It also logs the error with structured logging for observability.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	// Determine status code, error type, and error code based on error type
	var statusCode int
	var errorType string
	var errorCode string
	var message string

	// First, try to map plugin errors using plugin.HTTPStatus
	// This handles: ErrPluginNotFound, ErrInvalidInput, ErrUnavailable, ErrConflict, ErrPartialFailure, etc.
	if isPluginError(err) {
		statusCode = plugin.HTTPStatus(err)
		errorCode = plugin.ErrorCode(err)
		message = err.Error()
		errorType = httpStatusText(statusCode)
	} else {
		// Check for storage errors
		var notFoundErr *storage.NotFoundError
		if errors.As(err, &notFoundErr) {
			statusCode = http.StatusNotFound
			errorType = "Not Found"
			errorCode = "RESOURCE_NOT_FOUND"
			message = notFoundErr.Error()
		} else {
			// Generic error - return 500
			statusCode = http.StatusInternalServerError
			errorType = "Internal Server Error"
			errorCode = "INTERNAL_ERROR"
			message = err.Error()
		}
	}

	// Log the error with context
	logEvent := log.Error().
		Str("component", "api").
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Int("status", statusCode).
		Str("error_code", errorCode).
		Err(err)

	if statusCode == http.StatusNotFound {
		logEvent.Msg("Resource not found")
	} else if statusCode >= 500 {
		logEvent.Msg("Internal server error")
	} else if statusCode >= 400 {
		logEvent.Msg("Client error")
	} else {
		logEvent.Msg("Request failed")
	}

	// Write error response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error:   errorType,
		Code:    errorCode,
		Message: message,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().
			Str("component", "api").
			Err(err).
			Msg("Failed to encode error response")
	}
}

// isPluginError checks if the error is a plugin service error
func isPluginError(err error) bool {
	return errors.Is(err, plugin.ErrPluginNotFound) ||
		errors.Is(err, plugin.ErrPluginNotInstalled) ||
		errors.Is(err, plugin.ErrNoPluginsFound) ||
		errors.Is(err, plugin.ErrInvalidInput) ||
		errors.Is(err, plugin.ErrInvalidCategory) ||
		errors.Is(err, plugin.ErrInvalidPluginID) ||
		errors.Is(err, plugin.ErrSourceNotAvailable) ||
		errors.Is(err, plugin.ErrUnavailable) ||
		errors.Is(err, plugin.ErrPluginAlreadyInstalled) ||
		errors.Is(err, plugin.ErrConflict) ||
		errors.Is(err, plugin.ErrPartialFailure) ||
		errors.Is(err, plugin.ErrChecksumMismatch)
}

// httpStatusText returns human-readable text for HTTP status codes
func httpStatusText(statusCode int) string {
	switch statusCode {
	case http.StatusOK:
		return "OK"
	case http.StatusBadRequest:
		return "Bad Request"
	case http.StatusNotFound:
		return "Not Found"
	case http.StatusConflict:
		return "Conflict"
	case http.StatusInternalServerError:
		return "Internal Server Error"
	case http.StatusServiceUnavailable:
		return "Service Unavailable"
	default:
		return http.StatusText(statusCode)
	}
}

// WriteJSONError writes a custom JSON error response with a specific status code.
// Use this when you need fine-grained control over the error response.
//
// Example:
//
//	WriteJSONError(w, http.StatusBadRequest, "Invalid Input", "INVALID_TARGET", "Target parameter is required")
func WriteJSONError(w http.ResponseWriter, statusCode int, errorType, errorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error:   errorType,
		Code:    errorCode,
		Message: message,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().
			Str("component", "api").
			Err(err).
			Msg("Failed to encode error response")
	}
}

// WriteJSON writes a JSON response to the client.
// Use this for successful API responses.
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Error().
			Str("component", "api").
			Err(err).
			Msg("Failed to encode JSON response")
	}
}

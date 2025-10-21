package api

import (
	"encoding/json"
	"errors"
	"net/http"

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
//	  "message": "Scan with ID 'scan-123' not found"
//	}
type ErrorResponse struct {
	Error   string `json:"error"`             // Short error type (e.g., "Not Found", "Internal Server Error")
	Message string `json:"message,omitempty"` // Detailed error message (optional)
}

// WriteError writes a standard JSON error response to the client.
// It automatically determines the HTTP status code based on error type:
//   - storage.NotFoundError → 404 Not Found
//   - All other errors → 500 Internal Server Error
//
// It also logs the error with structured logging for observability.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	// Determine status code and error message based on error type
	var statusCode int
	var errorType string
	var message string

	// Check for specific error types
	var notFoundErr *storage.NotFoundError
	if errors.As(err, &notFoundErr) {
		statusCode = http.StatusNotFound
		errorType = "Not Found"
		message = notFoundErr.Error()
	} else {
		// Generic error - return 500
		statusCode = http.StatusInternalServerError
		errorType = "Internal Server Error"
		message = err.Error()
	}

	// Log the error with context
	logEvent := log.Error().
		Str("component", "api").
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Int("status", statusCode).
		Err(err)

	if statusCode == http.StatusNotFound {
		logEvent.Msg("Resource not found")
	} else {
		logEvent.Msg("Request failed")
	}

	// Write error response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error:   errorType,
		Message: message,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().
			Str("component", "api").
			Err(err).
			Msg("Failed to encode error response")
	}
}

// WriteJSONError writes a custom JSON error response with a specific status code.
// Use this when you need fine-grained control over the error response.
//
// Example:
//
//	WriteJSONError(w, http.StatusBadRequest, "Invalid Input", "Target parameter is required")
func WriteJSONError(w http.ResponseWriter, statusCode int, errorType, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := ErrorResponse{
		Error:   errorType,
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

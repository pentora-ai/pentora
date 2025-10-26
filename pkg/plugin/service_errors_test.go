package plugin

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExitCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "nil error returns 0",
			err:      nil,
			expected: 0,
		},
		{
			name:     "ErrPluginNotFound returns 4",
			err:      ErrPluginNotFound,
			expected: 4,
		},
		{
			name:     "ErrInvalidOption returns 2",
			err:      ErrInvalidOption,
			expected: 2,
		},
		{
			name:     "ErrUnavailable returns 7",
			err:      ErrUnavailable,
			expected: 7,
		},
		{
			name:     "ErrPartialFailure returns 8",
			err:      ErrPartialFailure,
			expected: 8,
		},
		{
			name:     "ErrConflict returns 1",
			err:      ErrConflict,
			expected: 1,
		},
		{
			name:     "wrapped ErrPluginNotFound returns 4",
			err:      fmt.Errorf("failed to install: %w", ErrPluginNotFound),
			expected: 4,
		},
		{
			name:     "wrapped ErrInvalidOption returns 2",
			err:      fmt.Errorf("validation failed: %w", ErrInvalidOption),
			expected: 2,
		},
		{
			name:     "unknown error returns 1",
			err:      errors.New("unknown error"),
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := ExitCode(tt.err)
			require.Equal(t, tt.expected, code)
		})
	}
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{
			name:     "nil error returns 200",
			err:      nil,
			expected: 200,
		},
		{
			name:     "ErrPluginNotFound returns 404",
			err:      ErrPluginNotFound,
			expected: 404,
		},
		{
			name:     "ErrInvalidOption returns 400",
			err:      ErrInvalidOption,
			expected: 400,
		},
		{
			name:     "ErrUnavailable returns 503",
			err:      ErrUnavailable,
			expected: 503,
		},
		{
			name:     "ErrConflict returns 409",
			err:      ErrConflict,
			expected: 409,
		},
		{
			name:     "ErrPartialFailure returns 200",
			err:      ErrPartialFailure,
			expected: 200,
		},
		{
			name:     "wrapped ErrPluginNotFound returns 404",
			err:      fmt.Errorf("plugin not found: %w", ErrPluginNotFound),
			expected: 404,
		},
		{
			name:     "wrapped ErrInvalidOption returns 400",
			err:      fmt.Errorf("invalid target: %w", ErrInvalidOption),
			expected: 400,
		},
		{
			name:     "unknown error returns 500",
			err:      errors.New("database error"),
			expected: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status := HTTPStatus(tt.err)
			require.Equal(t, tt.expected, status)
		})
	}
}

func TestErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error returns empty string",
			err:      nil,
			expected: "",
		},
		{
			name:     "ErrPluginNotFound returns PLUGIN_NOT_FOUND",
			err:      ErrPluginNotFound,
			expected: "PLUGIN_NOT_FOUND",
		},
		{
			name:     "ErrInvalidOption returns INVALID_INPUT (alias)",
			err:      ErrInvalidOption,
			expected: "INVALID_INPUT",
		},
		{
			name:     "ErrUnavailable returns SERVICE_UNAVAILABLE",
			err:      ErrUnavailable,
			expected: "SERVICE_UNAVAILABLE",
		},
		{
			name:     "ErrConflict returns VERSION_CONFLICT",
			err:      ErrConflict,
			expected: "VERSION_CONFLICT",
		},
		{
			name:     "ErrPartialFailure returns PARTIAL_FAILURE",
			err:      ErrPartialFailure,
			expected: "PARTIAL_FAILURE",
		},
		{
			name:     "wrapped ErrPluginNotFound returns PLUGIN_NOT_FOUND",
			err:      fmt.Errorf("failed: %w", ErrPluginNotFound),
			expected: "PLUGIN_NOT_FOUND",
		},
		{
			name:     "unknown error returns INTERNAL_ERROR",
			err:      errors.New("unknown"),
			expected: "INTERNAL_ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := ErrorCode(tt.err)
			require.Equal(t, tt.expected, code)
		})
	}
}

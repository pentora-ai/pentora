package ui

import (
	"testing"

	"github.com/pentora-ai/pentora/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestNewHandler_ProductionMode(t *testing.T) {
	cfg := config.ServerConfig{
		UI: config.UIConfig{
			AssetsPath: "", // Production mode (empty path)
		},
	}

	handler := NewHandler(cfg)
	require.NotNil(t, handler)
}

func TestNewHandler_DevMode(t *testing.T) {
	cfg := config.ServerConfig{
		UI: config.UIConfig{
			AssetsPath: "./testdata", // Dev mode (disk path)
		},
	}

	handler := NewHandler(cfg)
	require.NotNil(t, handler)
}

func TestShouldServeFile(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		// Should serve directly
		{"root", "/", true},
		{"index.html", "/index.html", true},
		{"js file", "/assets/main.js", true},
		{"css file", "/assets/style.css", true},
		{"png image", "/assets/logo.png", true},
		{"static asset", "/static/image.jpg", true},
		{"public asset", "/public/data.json", true},
		{"favicon", "/favicon.ico", true},
		{"manifest", "/manifest.json", true},
		{"robots", "/robots.txt", true},

		// Should fall back to index.html (SPA routes)
		{"dashboard route", "/dashboard", false},
		{"settings route", "/settings", false},
		{"nested route", "/settings/profile", false},
		{"user route", "/users/123", false},
		{"deep nested", "/app/settings/security", false},
		{"html route", "/about.html", false}, // Falls back for SPA
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldServeFile(tt.path)
			require.Equal(t, tt.expected, result, "shouldServeFile(%s) = %v, want %v", tt.path, result, tt.expected)
		})
	}
}

func TestFileExists(t *testing.T) {
	// Test with static.go (this file exists)
	require.True(t, FileExists("static.go"))

	// Test with non-existent file
	require.False(t, FileExists("nonexistent-file-12345.txt"))

	// Test with directory (should return false)
	require.False(t, FileExists("."))
}

// Note: Full integration test requires built UI assets
// To run manually:
//  1. cd ui/web && npm run build
//  2. go test ./pkg/server/ui -v
//
// The test below is skipped in CI until UI is built
func TestNewHandler_Integration(t *testing.T) {
	t.Skip("Requires built UI assets - run manually with 'make build-ui && go test'")

	// This would test actual file serving with built assets
	// Left as placeholder for manual testing
}

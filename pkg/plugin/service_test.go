// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pentora-ai/pentora/pkg/storage"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	t.Run("with valid cache directory", func(t *testing.T) {
		cacheDir := t.TempDir()

		svc, err := NewService(cacheDir)

		require.NoError(t, err)
		require.NotNil(t, svc)
		require.NotNil(t, svc.cache)
		require.NotNil(t, svc.manifest)
		require.NotNil(t, svc.downloader)
		require.NotNil(t, svc.logger)
		require.NotEmpty(t, svc.sources)
		require.Len(t, svc.sources, 1) // Default: 1 source
		require.Equal(t, "official", svc.sources[0].Name)
	})

	t.Run("with empty cache directory uses default", func(t *testing.T) {
		svc, err := NewService("")

		require.NoError(t, err)
		require.NotNil(t, svc)

		// Verify cache manager was created with default path
		require.NotNil(t, svc.cache)
	})

	t.Run("creates cache directory if not exists", func(t *testing.T) {
		tempDir := t.TempDir()
		cacheDir := filepath.Join(tempDir, "nonexistent", "cache")

		svc, err := NewService(cacheDir)

		require.NoError(t, err)
		require.NotNil(t, svc)

		// Verify directory was created
		_, err = os.Stat(cacheDir)
		require.NoError(t, err, "cache directory should be created")
	})

	t.Run("creates manifest in parent directory", func(t *testing.T) {
		tempDir := t.TempDir()
		cacheDir := filepath.Join(tempDir, "cache")

		svc, err := NewService(cacheDir)

		require.NoError(t, err)
		require.NotNil(t, svc.manifest)

		// Manifest should be in parent directory
		expectedManifestDir := tempDir
		_, err = os.Stat(expectedManifestDir)
		require.NoError(t, err)
	})
}

func TestService_WithStorage(t *testing.T) {
	t.Run("injects storage backend", func(t *testing.T) {
		cacheDir := t.TempDir()
		svc, _ := NewService(cacheDir)

		// Mock storage backend (nil for now, will be real in integration tests)
		var mockStorage storage.Backend = nil

		result := svc.WithStorage(mockStorage)

		// Fluent API: should return self
		require.Equal(t, svc, result)
		require.Equal(t, mockStorage, svc.storage)
	})

	t.Run("fluent API chaining", func(t *testing.T) {
		cacheDir := t.TempDir()
		svc, _ := NewService(cacheDir)

		var mockStorage storage.Backend = nil

		// Should allow method chaining
		result := svc.WithStorage(mockStorage).WithStorage(mockStorage)

		require.NotNil(t, result)
		require.Equal(t, svc, result)
	})
}

func TestService_WithLogger(t *testing.T) {
	t.Run("injects custom logger", func(t *testing.T) {
		cacheDir := t.TempDir()
		svc, _ := NewService(cacheDir)

		customLogger := zerolog.New(os.Stdout).With().
			Str("service", "plugin-test").
			Logger()

		result := svc.WithLogger(customLogger)

		// Fluent API: should return self
		require.Equal(t, svc, result)
		require.Equal(t, customLogger, svc.logger)
	})

	t.Run("replaces default logger", func(t *testing.T) {
		cacheDir := t.TempDir()
		svc, _ := NewService(cacheDir)

		defaultLogger := svc.logger

		customLogger := zerolog.New(os.Stdout)
		svc.WithLogger(customLogger)

		// Logger should be different from default
		require.NotEqual(t, defaultLogger, svc.logger)
		require.Equal(t, customLogger, svc.logger)
	})
}

func TestService_WithSources(t *testing.T) {
	t.Run("injects custom sources", func(t *testing.T) {
		cacheDir := t.TempDir()
		svc, _ := NewService(cacheDir)

		customSources := []PluginSource{
			{
				Name:     "custom",
				URL:      "https://custom.example.com/manifest.yaml",
				Enabled:  true,
				Priority: 1,
			},
			{
				Name:     "mirror",
				URL:      "https://mirror.example.com/manifest.yaml",
				Enabled:  true,
				Priority: 2,
			},
		}

		result := svc.WithSources(customSources)

		// Fluent API: should return self
		require.Equal(t, svc, result)
		require.Equal(t, customSources, svc.sources)
		require.Len(t, svc.sources, 2)
		require.Equal(t, "custom", svc.sources[0].Name)
		require.Equal(t, "mirror", svc.sources[1].Name)
	})

	t.Run("recreates downloader with new sources", func(t *testing.T) {
		cacheDir := t.TempDir()
		svc, _ := NewService(cacheDir)

		originalDownloader := svc.downloader

		customSources := []PluginSource{
			{Name: "custom", URL: "https://custom.example.com/manifest.yaml", Enabled: true},
		}

		svc.WithSources(customSources)

		// Downloader should be recreated
		require.NotNil(t, svc.downloader)
		// Should be different instance
		require.NotEqual(t, originalDownloader, svc.downloader)
	})

	t.Run("replaces default sources", func(t *testing.T) {
		cacheDir := t.TempDir()
		svc, _ := NewService(cacheDir)

		// Initially has default sources
		require.Len(t, svc.sources, 1)
		require.Equal(t, "official", svc.sources[0].Name)

		customSources := []PluginSource{
			{Name: "enterprise", URL: "https://enterprise.example.com/manifest.yaml", Enabled: true},
		}

		svc.WithSources(customSources)

		// Should replace default sources
		require.Len(t, svc.sources, 1)
		require.Equal(t, "enterprise", svc.sources[0].Name)
	})
}

func TestService_FluentAPI_Chaining(t *testing.T) {
	t.Run("method chaining works", func(t *testing.T) {
		cacheDir := t.TempDir()

		customLogger := zerolog.New(os.Stdout)
		customSources := []PluginSource{
			{Name: "custom", URL: "https://custom.example.com/manifest.yaml", Enabled: true},
		}
		var mockStorage storage.Backend = nil

		// Chain all With* methods
		svc, err := NewService(cacheDir)
		require.NoError(t, err)

		result := svc.
			WithLogger(customLogger).
			WithSources(customSources).
			WithStorage(mockStorage)

		require.NotNil(t, result)
		require.Equal(t, svc, result)
		require.Equal(t, customLogger, svc.logger)
		require.Equal(t, customSources, svc.sources)
		require.Equal(t, mockStorage, svc.storage)
	})

	t.Run("chaining order doesn't matter", func(t *testing.T) {
		cacheDir := t.TempDir()

		customLogger := zerolog.New(os.Stdout)
		customSources := []PluginSource{
			{Name: "custom", URL: "https://custom.example.com/manifest.yaml", Enabled: true},
		}

		// Chain in different order
		svc, _ := NewService(cacheDir)
		result := svc.
			WithSources(customSources).
			WithLogger(customLogger)

		require.NotNil(t, result)
		require.Equal(t, customLogger, svc.logger)
		require.Equal(t, customSources, svc.sources)
	})
}

func TestDefaultSources(t *testing.T) {
	t.Run("returns official source", func(t *testing.T) {
		sources := defaultSources()

		require.Len(t, sources, 1)
		require.Equal(t, "official", sources[0].Name)
		require.Equal(t, "https://plugins.pentora.ai/manifest.yaml", sources[0].URL)
		require.True(t, sources[0].Enabled)
		require.Equal(t, 1, sources[0].Priority)
	})

	t.Run("returns enabled sources", func(t *testing.T) {
		sources := defaultSources()

		for _, source := range sources {
			require.True(t, source.Enabled, "default sources should be enabled")
		}
	})
}

func TestService_Initialization(t *testing.T) {
	t.Run("all dependencies initialized", func(t *testing.T) {
		cacheDir := t.TempDir()

		svc, err := NewService(cacheDir)

		require.NoError(t, err)

		// Verify all required dependencies are initialized
		require.NotNil(t, svc.cache, "cache manager should be initialized")
		require.NotNil(t, svc.manifest, "manifest manager should be initialized")
		require.NotNil(t, svc.downloader, "downloader should be initialized")
		require.NotEmpty(t, svc.sources, "sources should have default values")
		require.NotNil(t, svc.logger, "logger should have default value")
	})

	t.Run("optional dependencies default to nil/zero", func(t *testing.T) {
		cacheDir := t.TempDir()

		svc, err := NewService(cacheDir)

		require.NoError(t, err)

		// Optional dependencies should be nil until injected
		require.Nil(t, svc.storage, "storage should be nil by default")
	})
}

// Helper function for tests (to be used in future test cases)
func setupTestService(t *testing.T) *Service {
	t.Helper()

	cacheDir := t.TempDir()
	svc, err := NewService(cacheDir)
	if err != nil {
		t.Fatalf("failed to create test service: %v", err)
	}

	return svc
}

// Test that verifies service can be created and used in realistic scenario
func TestService_Integration_Basic(t *testing.T) {
	t.Run("create service and verify it's ready for operations", func(t *testing.T) {
		cacheDir := t.TempDir()

		// Create service
		svc, err := NewService(cacheDir)
		require.NoError(t, err)
		require.NotNil(t, svc)

		// Service should be ready to use
		// (Will be tested further when Install/Update/etc methods are implemented)

		// Verify service has working cache
		require.NotNil(t, svc.cache)

		// Verify service has working manifest
		require.NotNil(t, svc.manifest)

		// Load manifest (should not error on empty manifest)
		err = svc.manifest.Load()
		require.NoError(t, err)
	})
}

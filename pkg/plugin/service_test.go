// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package plugin

import (
	"context"
	"fmt"
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

	// Note: WithSources no longer recreates the downloader since we use interfaces.
	// If you need to change sources, create a new Service instance instead.

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

// setupTestService is a helper function for tests (will be used in future test cases)
// nolint:unused // Will be used when Install(), Update(), etc. methods are implemented
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

		// Verify service has working dependencies
		require.NotNil(t, svc.cache)
		require.NotNil(t, svc.manifest)
		require.NotNil(t, svc.downloader)
	})
}

// ============================================================================
// Install() Method Tests
// ============================================================================

// Test helper to create Service with mocks
func newTestService(cache CacheInterface, manifest ManifestInterface, downloader DownloaderInterface, sources []PluginSource) *Service {
	return &Service{
		cache:      cache,
		manifest:   manifest,
		downloader: downloader,
		sources:    sources,
		logger:     zerolog.New(os.Stdout),
	}
}

// Mock implementations

// mockDownloader for testing Install() method
type mockDownloader struct {
	fetchManifestFunc func(ctx context.Context, src PluginSource) (*PluginManifest, error)
	downloadFunc      func(ctx context.Context, id, version string) (*CacheEntry, error)
}

func (m *mockDownloader) FetchManifest(ctx context.Context, src PluginSource) (*PluginManifest, error) {
	if m.fetchManifestFunc != nil {
		return m.fetchManifestFunc(ctx, src)
	}
	return &PluginManifest{Plugins: []PluginManifestEntry{}}, nil
}

func (m *mockDownloader) Download(ctx context.Context, id, version string) (*CacheEntry, error) {
	if m.downloadFunc != nil {
		return m.downloadFunc(ctx, id, version)
	}
	return &CacheEntry{}, nil
}

// mockCacheManager for testing Install() method
type mockCacheManager struct {
	getEntryFunc func(name, version string) (*CacheEntry, error)
}

func (m *mockCacheManager) GetEntry(name, version string) (*CacheEntry, error) {
	if m.getEntryFunc != nil {
		return m.getEntryFunc(name, version)
	}
	return nil, ErrPluginNotInstalled
}

// mockManifestManager for testing Install() method
type mockManifestManager struct {
	addFunc    func(entry *ManifestEntry) error
	saveFunc   func() error
	listFunc   func() ([]*ManifestEntry, error)
	removeFunc func(id string) error
}

func (m *mockManifestManager) Add(entry *ManifestEntry) error {
	if m.addFunc != nil {
		return m.addFunc(entry)
	}
	return nil
}

func (m *mockManifestManager) Save() error {
	if m.saveFunc != nil {
		return m.saveFunc()
	}
	return nil
}

func (m *mockManifestManager) List() ([]*ManifestEntry, error) {
	if m.listFunc != nil {
		return m.listFunc()
	}
	return []*ManifestEntry{}, nil
}

func (m *mockManifestManager) Remove(id string) error {
	if m.removeFunc != nil {
		return m.removeFunc(id)
	}
	return nil
}

func TestService_Install_ByPluginID(t *testing.T) {
	t.Run("install plugin by ID successfully", func(t *testing.T) {
		ctx := context.Background()

		// Mock downloader that returns a test plugin
		dl := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{
					Plugins: []PluginManifestEntry{
						{
							ID:         "test-plugin",
							Name:       "Test Plugin",
							Version:    "1.0.0",
							Author:     "Test Author",
							Categories: []Category{CategorySSH},
							URL:        "https://example.com/plugin.tar.gz",
							Checksum:   "sha256:abcd1234",
							Size:       1024,
						},
					},
				}, nil
			},
			downloadFunc: func(ctx context.Context, id, version string) (*CacheEntry, error) {
				require.Equal(t, "test-plugin", id)
				require.Equal(t, "1.0.0", version)
				return &CacheEntry{Name: "Test Plugin", Version: "1.0.0"}, nil
			},
		}

		// Mock cache that returns "not found" (plugin not cached)
		cache := &mockCacheManager{
			getEntryFunc: func(name, version string) (*CacheEntry, error) {
				return nil, ErrPluginNotInstalled
			},
		}

		// Mock manifest
		manifest := &mockManifestManager{}

		// Create service with mocks
		svc := newTestService(cache, manifest, dl, []PluginSource{
			{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
		})

		// Install plugin by ID
		result, err := svc.Install(ctx, "test-plugin", InstallOptions{})

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 1, result.InstalledCount)
		require.Equal(t, 0, result.SkippedCount)
		require.Equal(t, 0, result.FailedCount)
		require.Len(t, result.Plugins, 1)
		require.Equal(t, "test-plugin", result.Plugins[0].ID)
		require.Equal(t, "Test Plugin", result.Plugins[0].Name)
		require.Equal(t, "1.0.0", result.Plugins[0].Version)
	})

	t.Run("plugin not found error", func(t *testing.T) {
		ctx := context.Background()

		// Mock downloader that returns empty manifest
		dl := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{Plugins: []PluginManifestEntry{}}, nil
			},
		}

		svc := newTestService(&mockCacheManager{}, &mockManifestManager{}, dl, []PluginSource{
			{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
		})

		// Try to install non-existent plugin
		result, err := svc.Install(ctx, "non-existent-plugin", InstallOptions{})

		// Verify error
		require.Error(t, err)
		require.Nil(t, result)
		require.ErrorIs(t, err, ErrNoPluginsFound, "should return ErrNoPluginsFound when manifest is empty")
	})
}

func TestService_Install_ByCategory(t *testing.T) {
	t.Run("install all plugins in category", func(t *testing.T) {
		ctx := context.Background()

		// Mock downloader with multiple SSH plugins
		mockDownloader := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{
					Plugins: []PluginManifestEntry{
						{
							ID:         "ssh-plugin-1",
							Name:       "SSH Plugin 1",
							Version:    "1.0.0",
							Categories: []Category{CategorySSH},
							URL:        "https://example.com/ssh1.tar.gz",
							Checksum:   "sha256:abcd1234",
						},
						{
							ID:         "ssh-plugin-2",
							Name:       "SSH Plugin 2",
							Version:    "2.0.0",
							Categories: []Category{CategorySSH},
							URL:        "https://example.com/ssh2.tar.gz",
							Checksum:   "sha256:efgh5678",
						},
						{
							ID:         "http-plugin",
							Name:       "HTTP Plugin",
							Version:    "1.0.0",
							Categories: []Category{CategoryHTTP},
							URL:        "https://example.com/http.tar.gz",
							Checksum:   "sha256:ijkl9012",
						},
					},
				}, nil
			},
			downloadFunc: func(ctx context.Context, id, version string) (*CacheEntry, error) {
				return &CacheEntry{}, nil
			},
		}

		mockCache := &mockCacheManager{
			getEntryFunc: func(name, version string) (*CacheEntry, error) {
				return nil, ErrPluginNotInstalled
			},
		}

		svc := &Service{
			cache:      mockCache,
			manifest:   &mockManifestManager{},
			downloader: mockDownloader,
			sources: []PluginSource{
				{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
			},
			logger: zerolog.New(os.Stdout),
		}

		// Install all SSH plugins
		result, err := svc.Install(ctx, "ssh", InstallOptions{})

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 2, result.InstalledCount, "should install 2 SSH plugins")
		require.Equal(t, 0, result.SkippedCount)
		require.Equal(t, 0, result.FailedCount)
		require.Len(t, result.Plugins, 2)
	})

	t.Run("no plugins found in category", func(t *testing.T) {
		ctx := context.Background()

		mockDownloader := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{
					Plugins: []PluginManifestEntry{
						{
							ID:         "http-plugin",
							Name:       "HTTP Plugin",
							Version:    "1.0.0",
							Categories: []Category{CategoryHTTP},
						},
					},
				}, nil
			},
		}

		svc := &Service{
			cache:      &mockCacheManager{},
			manifest:   &mockManifestManager{},
			downloader: mockDownloader,
			sources: []PluginSource{
				{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
			},
			logger: zerolog.New(os.Stdout),
		}

		// Try to install TLS plugins (none exist in manifest)
		result, err := svc.Install(ctx, "tls", InstallOptions{})

		// Verify error
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "no plugins match criteria")
	})
}

func TestService_Install_WithForce(t *testing.T) {
	t.Run("force reinstall cached plugin", func(t *testing.T) {
		ctx := context.Background()

		mockDownloader := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{
					Plugins: []PluginManifestEntry{
						{
							ID:         "cached-plugin",
							Name:       "Cached Plugin",
							Version:    "1.0.0",
							Categories: []Category{CategorySSH},
							URL:        "https://example.com/cached.tar.gz",
							Checksum:   "sha256:abcd1234",
						},
					},
				}, nil
			},
			downloadFunc: func(ctx context.Context, id, version string) (*CacheEntry, error) {
				return &CacheEntry{}, nil
			},
		}

		// Mock cache that returns plugin as already cached
		mockCache := &mockCacheManager{
			getEntryFunc: func(name, version string) (*CacheEntry, error) {
				return &CacheEntry{Name: name, Version: version}, nil
			},
		}

		svc := &Service{
			cache:      mockCache,
			manifest:   &mockManifestManager{},
			downloader: mockDownloader,
			sources: []PluginSource{
				{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
			},
			logger: zerolog.New(os.Stdout),
		}

		// Install with force=true
		result, err := svc.Install(ctx, "cached-plugin", InstallOptions{Force: true})

		// Verify plugin was reinstalled
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 1, result.InstalledCount, "should reinstall with force")
		require.Equal(t, 0, result.SkippedCount)
	})

	t.Run("skip already cached plugin without force", func(t *testing.T) {
		ctx := context.Background()

		mockDownloader := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{
					Plugins: []PluginManifestEntry{
						{
							ID:         "cached-plugin",
							Name:       "Cached Plugin",
							Version:    "1.0.0",
							Categories: []Category{CategorySSH},
						},
					},
				}, nil
			},
		}

		// Mock cache that returns plugin as already cached
		mockCache := &mockCacheManager{
			getEntryFunc: func(name, version string) (*CacheEntry, error) {
				return &CacheEntry{Name: name, Version: version}, nil
			},
		}

		svc := &Service{
			cache:      mockCache,
			manifest:   &mockManifestManager{},
			downloader: mockDownloader,
			sources: []PluginSource{
				{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
			},
			logger: zerolog.New(os.Stdout),
		}

		// Install without force (default)
		result, err := svc.Install(ctx, "cached-plugin", InstallOptions{Force: false})

		// Verify plugin was skipped
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 0, result.InstalledCount)
		require.Equal(t, 1, result.SkippedCount, "should skip already cached plugin")
	})
}

func TestService_Install_WithDryRun(t *testing.T) {
	t.Run("dry run does not download", func(t *testing.T) {
		ctx := context.Background()

		downloadCalled := false

		mockDownloader := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{
					Plugins: []PluginManifestEntry{
						{
							ID:         "test-plugin",
							Name:       "Test Plugin",
							Version:    "1.0.0",
							Categories: []Category{CategorySSH},
						},
					},
				}, nil
			},
			downloadFunc: func(ctx context.Context, id, version string) (*CacheEntry, error) {
				downloadCalled = true
				return &CacheEntry{}, nil
			},
		}

		mockCache := &mockCacheManager{
			getEntryFunc: func(name, version string) (*CacheEntry, error) {
				return nil, ErrPluginNotInstalled
			},
		}

		svc := &Service{
			cache:      mockCache,
			manifest:   &mockManifestManager{},
			downloader: mockDownloader,
			sources: []PluginSource{
				{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
			},
			logger: zerolog.New(os.Stdout),
		}

		// Install with DryRun=true
		result, err := svc.Install(ctx, "test-plugin", InstallOptions{DryRun: true})

		// Verify no download occurred
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 1, result.InstalledCount, "should count as installed in dry run")
		require.False(t, downloadCalled, "download should not be called in dry run")
	})
}

func TestService_Install_WithSourceFilter(t *testing.T) {
	t.Run("install from specific source", func(t *testing.T) {
		ctx := context.Background()

		officialSourceCalled := false
		communitySourceCalled := false

		mockDownloader := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				if src.Name == "official" {
					officialSourceCalled = true
					return &PluginManifest{
						Plugins: []PluginManifestEntry{
							{
								ID:         "official-plugin",
								Name:       "Official Plugin",
								Version:    "1.0.0",
								Categories: []Category{CategorySSH},
							},
						},
					}, nil
				}
				if src.Name == "community" {
					communitySourceCalled = true
					return &PluginManifest{
						Plugins: []PluginManifestEntry{
							{
								ID:         "community-plugin",
								Name:       "Community Plugin",
								Version:    "1.0.0",
								Categories: []Category{CategorySSH},
							},
						},
					}, nil
				}
				return &PluginManifest{}, nil
			},
			downloadFunc: func(ctx context.Context, id, version string) (*CacheEntry, error) {
				return &CacheEntry{}, nil
			},
		}

		mockCache := &mockCacheManager{
			getEntryFunc: func(name, version string) (*CacheEntry, error) {
				return nil, ErrPluginNotInstalled
			},
		}

		svc := &Service{
			cache:      mockCache,
			manifest:   &mockManifestManager{},
			downloader: mockDownloader,
			sources: []PluginSource{
				{Name: "official", URL: "https://official.com/manifest.yaml", Enabled: true},
				{Name: "community", URL: "https://community.com/manifest.yaml", Enabled: true},
			},
			logger: zerolog.New(os.Stdout),
		}

		// Install from official source only
		result, err := svc.Install(ctx, "official-plugin", InstallOptions{Source: "official"})

		// Verify only official source was called
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 1, result.InstalledCount)
		require.True(t, officialSourceCalled, "official source should be called")
		require.False(t, communitySourceCalled, "community source should NOT be called")
	})

	t.Run("source not found error", func(t *testing.T) {
		ctx := context.Background()

		svc := &Service{
			cache:      &mockCacheManager{},
			manifest:   &mockManifestManager{},
			downloader: &mockDownloader{},
			sources: []PluginSource{
				{Name: "official", URL: "https://official.com/manifest.yaml", Enabled: true},
			},
			logger: zerolog.New(os.Stdout),
		}

		// Try to install from non-existent source
		result, err := svc.Install(ctx, "test-plugin", InstallOptions{Source: "non-existent"})

		// Verify error
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "source 'non-existent' not found")
	})
}

func TestService_Install_PartialFailures(t *testing.T) {
	t.Run("some plugins succeed, some fail", func(t *testing.T) {
		ctx := context.Background()

		mockDownloader := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{
					Plugins: []PluginManifestEntry{
						{
							ID:         "success-plugin",
							Name:       "Success Plugin",
							Version:    "1.0.0",
							Categories: []Category{CategorySSH},
						},
						{
							ID:         "fail-plugin",
							Name:       "Fail Plugin",
							Version:    "1.0.0",
							Categories: []Category{CategorySSH},
						},
					},
				}, nil
			},
			downloadFunc: func(ctx context.Context, id, version string) (*CacheEntry, error) {
				if id == "fail-plugin" {
					return nil, fmt.Errorf("download failed")
				}
				return &CacheEntry{}, nil
			},
		}

		mockCache := &mockCacheManager{
			getEntryFunc: func(name, version string) (*CacheEntry, error) {
				return nil, ErrPluginNotInstalled
			},
		}

		svc := &Service{
			cache:      mockCache,
			manifest:   &mockManifestManager{},
			downloader: mockDownloader,
			sources: []PluginSource{
				{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
			},
			logger: zerolog.New(os.Stdout),
		}

		// Install category with partial failure
		result, err := svc.Install(ctx, "ssh", InstallOptions{})

		// Verify partial success
		require.NoError(t, err, "should not return error on partial failure")
		require.NotNil(t, result)
		require.Equal(t, 1, result.InstalledCount, "one plugin should succeed")
		require.Equal(t, 1, result.FailedCount, "one plugin should fail")
		require.Len(t, result.Errors, 1, "should collect errors")
		require.Contains(t, result.Errors[0].Error(), "download failed")
	})
}

func TestService_Install_ContextCancellation(t *testing.T) {
	t.Run("context cancelled during installation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		downloadCount := 0

		mockDownloader := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{
					Plugins: []PluginManifestEntry{
						{ID: "plugin-1", Name: "Plugin 1", Version: "1.0.0", Categories: []Category{CategorySSH}},
						{ID: "plugin-2", Name: "Plugin 2", Version: "1.0.0", Categories: []Category{CategorySSH}},
						{ID: "plugin-3", Name: "Plugin 3", Version: "1.0.0", Categories: []Category{CategorySSH}},
					},
				}, nil
			},
			downloadFunc: func(ctx context.Context, id, version string) (*CacheEntry, error) {
				downloadCount++
				if downloadCount == 2 {
					cancel() // Cancel after second download
				}
				return &CacheEntry{}, nil
			},
		}

		mockCache := &mockCacheManager{
			getEntryFunc: func(name, version string) (*CacheEntry, error) {
				return nil, ErrPluginNotInstalled
			},
		}

		svc := &Service{
			cache:      mockCache,
			manifest:   &mockManifestManager{},
			downloader: mockDownloader,
			sources: []PluginSource{
				{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
			},
			logger: zerolog.New(os.Stdout),
		}

		// Install category - should be cancelled mid-way
		result, err := svc.Install(ctx, "ssh", InstallOptions{})

		// Verify context cancellation
		require.Error(t, err)
		require.Equal(t, context.Canceled, err)
		require.NotNil(t, result)
		require.Equal(t, 2, result.InstalledCount, "should install 2 before cancellation")
	})
}

func TestService_Install_EmptyManifest(t *testing.T) {
	t.Run("no plugins in manifest", func(t *testing.T) {
		ctx := context.Background()

		mockDownloader := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{Plugins: []PluginManifestEntry{}}, nil
			},
		}

		svc := newTestService(&mockCacheManager{}, &mockManifestManager{}, mockDownloader, []PluginSource{
			{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
		})

		// Try to install from empty manifest
		result, err := svc.Install(ctx, "test-plugin", InstallOptions{})

		// Verify error
		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "no plugins found in any source")
	})
}

// ============================================================================
// Update() Method Tests
// ============================================================================

func TestService_Update_AllPlugins(t *testing.T) {
	t.Run("update all plugins successfully", func(t *testing.T) {
		ctx := context.Background()

		// Mock downloader with multiple plugins
		dl := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{
					Plugins: []PluginManifestEntry{
						{
							ID:         "plugin-1",
							Name:       "Plugin 1",
							Version:    "1.0.0",
							Categories: []Category{CategorySSH},
							URL:        "https://example.com/plugin1.tar.gz",
							Checksum:   "sha256:abc123",
						},
						{
							ID:         "plugin-2",
							Name:       "Plugin 2",
							Version:    "2.0.0",
							Categories: []Category{CategoryHTTP},
							URL:        "https://example.com/plugin2.tar.gz",
							Checksum:   "sha256:def456",
						},
					},
				}, nil
			},
			downloadFunc: func(ctx context.Context, id, version string) (*CacheEntry, error) {
				return &CacheEntry{Name: id, Version: version}, nil
			},
		}

		// Mock cache - plugins not cached
		cache := &mockCacheManager{
			getEntryFunc: func(name, version string) (*CacheEntry, error) {
				return nil, ErrPluginNotInstalled
			},
		}

		svc := newTestService(cache, &mockManifestManager{}, dl, []PluginSource{
			{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
		})

		// Update all plugins
		result, err := svc.Update(ctx, UpdateOptions{})

		// Verify results
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 2, result.UpdatedCount)
		require.Equal(t, 0, result.SkippedCount)
		require.Equal(t, 0, result.FailedCount)
		require.Len(t, result.Plugins, 2)
	})

	t.Run("empty manifest returns error", func(t *testing.T) {
		ctx := context.Background()

		dl := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{Plugins: []PluginManifestEntry{}}, nil
			},
		}

		svc := newTestService(&mockCacheManager{}, &mockManifestManager{}, dl, []PluginSource{
			{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
		})

		result, err := svc.Update(ctx, UpdateOptions{})

		require.Error(t, err)
		require.Nil(t, result)
		require.ErrorIs(t, err, ErrNoPluginsFound)
	})
}

func TestService_Update_ByCategory(t *testing.T) {
	t.Run("update plugins in specific category", func(t *testing.T) {
		ctx := context.Background()

		dl := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{
					Plugins: []PluginManifestEntry{
						{
							ID:         "ssh-plugin-1",
							Name:       "SSH Plugin 1",
							Version:    "1.0.0",
							Categories: []Category{CategorySSH},
						},
						{
							ID:         "ssh-plugin-2",
							Name:       "SSH Plugin 2",
							Version:    "2.0.0",
							Categories: []Category{CategorySSH},
						},
						{
							ID:         "http-plugin",
							Name:       "HTTP Plugin",
							Version:    "1.0.0",
							Categories: []Category{CategoryHTTP},
						},
					},
				}, nil
			},
			downloadFunc: func(ctx context.Context, id, version string) (*CacheEntry, error) {
				return &CacheEntry{}, nil
			},
		}

		cache := &mockCacheManager{
			getEntryFunc: func(name, version string) (*CacheEntry, error) {
				return nil, ErrPluginNotInstalled
			},
		}

		svc := newTestService(cache, &mockManifestManager{}, dl, []PluginSource{
			{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
		})

		// Update only SSH plugins
		result, err := svc.Update(ctx, UpdateOptions{Category: CategorySSH})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 2, result.UpdatedCount, "should update 2 SSH plugins")
		require.Equal(t, 0, result.SkippedCount)
	})

	t.Run("no plugins in category", func(t *testing.T) {
		ctx := context.Background()

		dl := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{
					Plugins: []PluginManifestEntry{
						{
							ID:         "http-plugin",
							Name:       "HTTP Plugin",
							Version:    "1.0.0",
							Categories: []Category{CategoryHTTP},
						},
					},
				}, nil
			},
		}

		svc := newTestService(&mockCacheManager{}, &mockManifestManager{}, dl, []PluginSource{
			{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
		})

		result, err := svc.Update(ctx, UpdateOptions{Category: CategoryTLS})

		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "no plugins match criteria")
	})
}

func TestService_Update_SkipCached(t *testing.T) {
	t.Run("skip already cached plugins", func(t *testing.T) {
		ctx := context.Background()

		dl := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{
					Plugins: []PluginManifestEntry{
						{
							ID:         "cached-plugin",
							Name:       "Cached Plugin",
							Version:    "1.0.0",
							Categories: []Category{CategorySSH},
						},
						{
							ID:         "new-plugin",
							Name:       "New Plugin",
							Version:    "1.0.0",
							Categories: []Category{CategorySSH},
						},
					},
				}, nil
			},
			downloadFunc: func(ctx context.Context, id, version string) (*CacheEntry, error) {
				return &CacheEntry{}, nil
			},
		}

		// Mock cache - first plugin is cached, second is not
		cache := &mockCacheManager{
			getEntryFunc: func(name, version string) (*CacheEntry, error) {
				if name == "Cached Plugin" {
					return &CacheEntry{Name: name, Version: version}, nil
				}
				return nil, ErrPluginNotInstalled
			},
		}

		svc := newTestService(cache, &mockManifestManager{}, dl, []PluginSource{
			{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
		})

		result, err := svc.Update(ctx, UpdateOptions{})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 1, result.UpdatedCount, "one new plugin")
		require.Equal(t, 1, result.SkippedCount, "one cached plugin")
	})

	t.Run("force re-download cached plugins", func(t *testing.T) {
		ctx := context.Background()

		dl := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{
					Plugins: []PluginManifestEntry{
						{
							ID:         "cached-plugin",
							Name:       "Cached Plugin",
							Version:    "1.0.0",
							Categories: []Category{CategorySSH},
						},
					},
				}, nil
			},
			downloadFunc: func(ctx context.Context, id, version string) (*CacheEntry, error) {
				return &CacheEntry{}, nil
			},
		}

		// Mock cache - plugin is cached
		cache := &mockCacheManager{
			getEntryFunc: func(name, version string) (*CacheEntry, error) {
				return &CacheEntry{Name: name, Version: version}, nil
			},
		}

		svc := newTestService(cache, &mockManifestManager{}, dl, []PluginSource{
			{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
		})

		// Force re-download
		result, err := svc.Update(ctx, UpdateOptions{Force: true})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 1, result.UpdatedCount, "should re-download with force")
		require.Equal(t, 0, result.SkippedCount)
	})
}

func TestService_Update_DryRun(t *testing.T) {
	t.Run("dry run does not download", func(t *testing.T) {
		ctx := context.Background()

		downloadCalled := false

		dl := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{
					Plugins: []PluginManifestEntry{
						{
							ID:         "test-plugin",
							Name:       "Test Plugin",
							Version:    "1.0.0",
							Categories: []Category{CategorySSH},
						},
					},
				}, nil
			},
			downloadFunc: func(ctx context.Context, id, version string) (*CacheEntry, error) {
				downloadCalled = true
				return &CacheEntry{}, nil
			},
		}

		cache := &mockCacheManager{
			getEntryFunc: func(name, version string) (*CacheEntry, error) {
				return nil, ErrPluginNotInstalled
			},
		}

		svc := newTestService(cache, &mockManifestManager{}, dl, []PluginSource{
			{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
		})

		result, err := svc.Update(ctx, UpdateOptions{DryRun: true})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 1, result.UpdatedCount, "counts as updated in dry run")
		require.False(t, downloadCalled, "should not download in dry run")
		require.Len(t, result.Plugins, 1, "should list plugins that would be updated")
	})
}

func TestService_Update_SourceFilter(t *testing.T) {
	t.Run("update from specific source", func(t *testing.T) {
		ctx := context.Background()

		officialCalled := false
		communityCalled := false

		dl := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				if src.Name == "official" {
					officialCalled = true
					return &PluginManifest{
						Plugins: []PluginManifestEntry{
							{ID: "official-plugin", Name: "Official Plugin", Version: "1.0.0", Categories: []Category{CategorySSH}},
						},
					}, nil
				}
				if src.Name == "community" {
					communityCalled = true
					return &PluginManifest{
						Plugins: []PluginManifestEntry{
							{ID: "community-plugin", Name: "Community Plugin", Version: "1.0.0", Categories: []Category{CategorySSH}},
						},
					}, nil
				}
				return &PluginManifest{}, nil
			},
			downloadFunc: func(ctx context.Context, id, version string) (*CacheEntry, error) {
				return &CacheEntry{}, nil
			},
		}

		cache := &mockCacheManager{
			getEntryFunc: func(name, version string) (*CacheEntry, error) {
				return nil, ErrPluginNotInstalled
			},
		}

		svc := newTestService(cache, &mockManifestManager{}, dl, []PluginSource{
			{Name: "official", URL: "https://official.com/manifest.yaml", Enabled: true},
			{Name: "community", URL: "https://community.com/manifest.yaml", Enabled: true},
		})

		result, err := svc.Update(ctx, UpdateOptions{Source: "official"})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 1, result.UpdatedCount)
		require.True(t, officialCalled, "official source should be called")
		require.False(t, communityCalled, "community source should NOT be called")
	})

	t.Run("source not found error", func(t *testing.T) {
		ctx := context.Background()

		svc := newTestService(&mockCacheManager{}, &mockManifestManager{}, &mockDownloader{}, []PluginSource{
			{Name: "official", URL: "https://official.com/manifest.yaml", Enabled: true},
		})

		result, err := svc.Update(ctx, UpdateOptions{Source: "non-existent"})

		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "source 'non-existent' not found")
	})
}

func TestService_Update_PartialFailures(t *testing.T) {
	t.Run("some plugins succeed, some fail", func(t *testing.T) {
		ctx := context.Background()

		dl := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{
					Plugins: []PluginManifestEntry{
						{ID: "success-plugin", Name: "Success Plugin", Version: "1.0.0", Categories: []Category{CategorySSH}},
						{ID: "fail-plugin", Name: "Fail Plugin", Version: "1.0.0", Categories: []Category{CategorySSH}},
					},
				}, nil
			},
			downloadFunc: func(ctx context.Context, id, version string) (*CacheEntry, error) {
				if id == "fail-plugin" {
					return nil, fmt.Errorf("download failed")
				}
				return &CacheEntry{}, nil
			},
		}

		cache := &mockCacheManager{
			getEntryFunc: func(name, version string) (*CacheEntry, error) {
				return nil, ErrPluginNotInstalled
			},
		}

		svc := newTestService(cache, &mockManifestManager{}, dl, []PluginSource{
			{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
		})

		result, err := svc.Update(ctx, UpdateOptions{})

		require.NoError(t, err, "should not error on partial failure")
		require.NotNil(t, result)
		require.Equal(t, 1, result.UpdatedCount)
		require.Equal(t, 1, result.FailedCount)
		require.Len(t, result.Errors, 1)
		require.Contains(t, result.Errors[0].Error(), "download failed")
	})
}

func TestService_Update_ContextCancellation(t *testing.T) {
	t.Run("context cancelled during update", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		downloadCount := 0

		dl := &mockDownloader{
			fetchManifestFunc: func(ctx context.Context, src PluginSource) (*PluginManifest, error) {
				return &PluginManifest{
					Plugins: []PluginManifestEntry{
						{ID: "plugin-1", Name: "Plugin 1", Version: "1.0.0", Categories: []Category{CategorySSH}},
						{ID: "plugin-2", Name: "Plugin 2", Version: "1.0.0", Categories: []Category{CategorySSH}},
						{ID: "plugin-3", Name: "Plugin 3", Version: "1.0.0", Categories: []Category{CategorySSH}},
					},
				}, nil
			},
			downloadFunc: func(ctx context.Context, id, version string) (*CacheEntry, error) {
				downloadCount++
				if downloadCount == 2 {
					cancel() // Cancel after second download
				}
				return &CacheEntry{}, nil
			},
		}

		cache := &mockCacheManager{
			getEntryFunc: func(name, version string) (*CacheEntry, error) {
				return nil, ErrPluginNotInstalled
			},
		}

		svc := newTestService(cache, &mockManifestManager{}, dl, []PluginSource{
			{Name: "official", URL: "https://example.com/manifest.yaml", Enabled: true},
		})

		result, err := svc.Update(ctx, UpdateOptions{})

		require.Error(t, err)
		require.Equal(t, context.Canceled, err)
		require.NotNil(t, result)
		require.Equal(t, 2, result.UpdatedCount, "should update 2 before cancellation")
	})
}

// ============================================================================
// Uninstall() Method Tests
// ============================================================================

func TestService_Uninstall_ByPluginID(t *testing.T) {
	t.Run("uninstall specific plugin successfully", func(t *testing.T) {
		ctx := context.Background()

		removedID := ""

		manifest := &mockManifestManager{
			listFunc: func() ([]*ManifestEntry, error) {
				return []*ManifestEntry{
					{
						ID:      "test-plugin",
						Name:    "Test Plugin",
						Version: "1.0.0",
						Tags:    []string{"ssh"},
					},
					{
						ID:      "other-plugin",
						Name:    "Other Plugin",
						Version: "2.0.0",
						Tags:    []string{"http"},
					},
				}, nil
			},
			removeFunc: func(id string) error {
				removedID = id
				return nil
			},
		}

		svc := newTestService(&mockCacheManager{}, manifest, &mockDownloader{}, []PluginSource{})

		result, err := svc.Uninstall(ctx, "test-plugin", UninstallOptions{})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 1, result.RemovedCount)
		require.Equal(t, 0, result.FailedCount)
		require.Equal(t, 1, result.RemainingCount)
		require.Equal(t, "test-plugin", removedID)
	})

	t.Run("plugin not found error", func(t *testing.T) {
		ctx := context.Background()

		manifest := &mockManifestManager{
			listFunc: func() ([]*ManifestEntry, error) {
				return []*ManifestEntry{
					{ID: "existing-plugin", Name: "Existing", Version: "1.0.0"},
				}, nil
			},
		}

		svc := newTestService(&mockCacheManager{}, manifest, &mockDownloader{}, []PluginSource{})

		result, err := svc.Uninstall(ctx, "non-existent-plugin", UninstallOptions{})

		require.Error(t, err)
		require.Nil(t, result)
		require.ErrorIs(t, err, ErrPluginNotFound)
		require.Contains(t, err.Error(), "not found (not installed)")
	})
}

func TestService_Uninstall_ByCategory(t *testing.T) {
	t.Run("uninstall all plugins in category", func(t *testing.T) {
		ctx := context.Background()

		removedIDs := []string{}

		manifest := &mockManifestManager{
			listFunc: func() ([]*ManifestEntry, error) {
				return []*ManifestEntry{
					{ID: "ssh-plugin-1", Name: "SSH Plugin 1", Version: "1.0.0", Tags: []string{"ssh"}},
					{ID: "ssh-plugin-2", Name: "SSH Plugin 2", Version: "2.0.0", Tags: []string{"ssh"}},
					{ID: "http-plugin", Name: "HTTP Plugin", Version: "1.0.0", Tags: []string{"http"}},
				}, nil
			},
			removeFunc: func(id string) error {
				removedIDs = append(removedIDs, id)
				return nil
			},
		}

		svc := newTestService(&mockCacheManager{}, manifest, &mockDownloader{}, []PluginSource{})

		result, err := svc.Uninstall(ctx, "", UninstallOptions{Category: CategorySSH})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 2, result.RemovedCount, "should remove 2 SSH plugins")
		require.Equal(t, 0, result.FailedCount)
		require.Equal(t, 1, result.RemainingCount)
		require.Contains(t, removedIDs, "ssh-plugin-1")
		require.Contains(t, removedIDs, "ssh-plugin-2")
	})

	t.Run("no plugins in category", func(t *testing.T) {
		ctx := context.Background()

		manifest := &mockManifestManager{
			listFunc: func() ([]*ManifestEntry, error) {
				return []*ManifestEntry{
					{ID: "http-plugin", Name: "HTTP Plugin", Version: "1.0.0", Tags: []string{"http"}},
				}, nil
			},
		}

		svc := newTestService(&mockCacheManager{}, manifest, &mockDownloader{}, []PluginSource{})

		result, err := svc.Uninstall(ctx, "", UninstallOptions{Category: CategoryTLS})

		require.Error(t, err)
		require.Nil(t, result)
		require.ErrorIs(t, err, ErrNoPluginsFound)
		require.Contains(t, err.Error(), "no plugins found in category 'tls'")
	})
}

func TestService_Uninstall_All(t *testing.T) {
	t.Run("uninstall all plugins successfully", func(t *testing.T) {
		ctx := context.Background()

		removedIDs := []string{}

		manifest := &mockManifestManager{
			listFunc: func() ([]*ManifestEntry, error) {
				return []*ManifestEntry{
					{ID: "plugin-1", Name: "Plugin 1", Version: "1.0.0", Tags: []string{"ssh"}},
					{ID: "plugin-2", Name: "Plugin 2", Version: "2.0.0", Tags: []string{"http"}},
					{ID: "plugin-3", Name: "Plugin 3", Version: "3.0.0", Tags: []string{"tls"}},
				}, nil
			},
			removeFunc: func(id string) error {
				removedIDs = append(removedIDs, id)
				return nil
			},
		}

		svc := newTestService(&mockCacheManager{}, manifest, &mockDownloader{}, []PluginSource{})

		result, err := svc.Uninstall(ctx, "", UninstallOptions{All: true})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 3, result.RemovedCount, "should remove all 3 plugins")
		require.Equal(t, 0, result.FailedCount)
		require.Equal(t, 0, result.RemainingCount)
		require.Len(t, removedIDs, 3)
	})

	t.Run("empty manifest returns success", func(t *testing.T) {
		ctx := context.Background()

		manifest := &mockManifestManager{
			listFunc: func() ([]*ManifestEntry, error) {
				return []*ManifestEntry{}, nil
			},
		}

		svc := newTestService(&mockCacheManager{}, manifest, &mockDownloader{}, []PluginSource{})

		result, err := svc.Uninstall(ctx, "", UninstallOptions{All: true})

		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 0, result.RemovedCount)
		require.Equal(t, 0, result.FailedCount)
		require.Equal(t, 0, result.RemainingCount)
	})
}

func TestService_Uninstall_ValidationErrors(t *testing.T) {
	t.Run("no mode specified error", func(t *testing.T) {
		ctx := context.Background()

		svc := newTestService(&mockCacheManager{}, &mockManifestManager{}, &mockDownloader{}, []PluginSource{})

		result, err := svc.Uninstall(ctx, "", UninstallOptions{})

		require.Error(t, err)
		require.Nil(t, result)
		require.ErrorIs(t, err, ErrInvalidInput)
		require.Contains(t, err.Error(), "must specify plugin ID, category, or --all")
	})

	t.Run("multiple modes specified - target and category", func(t *testing.T) {
		ctx := context.Background()

		svc := newTestService(&mockCacheManager{}, &mockManifestManager{}, &mockDownloader{}, []PluginSource{})

		result, err := svc.Uninstall(ctx, "plugin-id", UninstallOptions{Category: CategorySSH})

		require.Error(t, err)
		require.Nil(t, result)
		require.ErrorIs(t, err, ErrInvalidInput)
		require.Contains(t, err.Error(), "cannot specify multiple uninstall modes")
	})

	t.Run("multiple modes specified - target and all", func(t *testing.T) {
		ctx := context.Background()

		svc := newTestService(&mockCacheManager{}, &mockManifestManager{}, &mockDownloader{}, []PluginSource{})

		result, err := svc.Uninstall(ctx, "plugin-id", UninstallOptions{All: true})

		require.Error(t, err)
		require.Nil(t, result)
		require.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("multiple modes specified - category and all", func(t *testing.T) {
		ctx := context.Background()

		svc := newTestService(&mockCacheManager{}, &mockManifestManager{}, &mockDownloader{}, []PluginSource{})

		result, err := svc.Uninstall(ctx, "", UninstallOptions{Category: CategorySSH, All: true})

		require.Error(t, err)
		require.Nil(t, result)
		require.ErrorIs(t, err, ErrInvalidInput)
	})
}

func TestService_Uninstall_PartialFailures(t *testing.T) {
	t.Run("some plugins succeed, some fail", func(t *testing.T) {
		ctx := context.Background()

		manifest := &mockManifestManager{
			listFunc: func() ([]*ManifestEntry, error) {
				return []*ManifestEntry{
					{ID: "success-plugin", Name: "Success Plugin", Version: "1.0.0", Tags: []string{"ssh"}},
					{ID: "fail-plugin", Name: "Fail Plugin", Version: "2.0.0", Tags: []string{"ssh"}},
				}, nil
			},
			removeFunc: func(id string) error {
				if id == "fail-plugin" {
					return fmt.Errorf("removal failed")
				}
				return nil
			},
		}

		svc := newTestService(&mockCacheManager{}, manifest, &mockDownloader{}, []PluginSource{})

		result, err := svc.Uninstall(ctx, "", UninstallOptions{Category: CategorySSH})

		require.NoError(t, err, "should not error on partial failure")
		require.NotNil(t, result)
		require.Equal(t, 1, result.RemovedCount)
		require.Equal(t, 1, result.FailedCount)
		require.Len(t, result.Errors, 1)
		require.Contains(t, result.Errors[0].Error(), "removal failed")
	})
}

func TestService_Uninstall_ContextCancellation(t *testing.T) {
	t.Run("context cancelled during uninstall", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		removeCount := 0

		manifest := &mockManifestManager{
			listFunc: func() ([]*ManifestEntry, error) {
				return []*ManifestEntry{
					{ID: "plugin-1", Name: "Plugin 1", Version: "1.0.0", Tags: []string{"ssh"}},
					{ID: "plugin-2", Name: "Plugin 2", Version: "1.0.0", Tags: []string{"ssh"}},
					{ID: "plugin-3", Name: "Plugin 3", Version: "1.0.0", Tags: []string{"ssh"}},
				}, nil
			},
			removeFunc: func(id string) error {
				removeCount++
				if removeCount == 2 {
					cancel() // Cancel after second removal
				}
				return nil
			},
		}

		svc := newTestService(&mockCacheManager{}, manifest, &mockDownloader{}, []PluginSource{})

		result, err := svc.Uninstall(ctx, "", UninstallOptions{All: true})

		require.Error(t, err)
		require.Equal(t, context.Canceled, err)
		require.NotNil(t, result)
		require.Equal(t, 2, result.RemovedCount, "should remove 2 before cancellation")
		require.Equal(t, 1, result.RemainingCount, "one plugin should remain")
	})
}

func TestService_Uninstall_ManifestErrors(t *testing.T) {
	t.Run("manifest list error", func(t *testing.T) {
		ctx := context.Background()

		manifest := &mockManifestManager{
			listFunc: func() ([]*ManifestEntry, error) {
				return nil, fmt.Errorf("failed to read manifest")
			},
		}

		svc := newTestService(&mockCacheManager{}, manifest, &mockDownloader{}, []PluginSource{})

		result, err := svc.Uninstall(ctx, "", UninstallOptions{All: true})

		require.Error(t, err)
		require.Nil(t, result)
		require.Contains(t, err.Error(), "list installed plugins")
		require.Contains(t, err.Error(), "failed to read manifest")
	})

	t.Run("manifest save error after successful removal", func(t *testing.T) {
		ctx := context.Background()

		manifest := &mockManifestManager{
			listFunc: func() ([]*ManifestEntry, error) {
				return []*ManifestEntry{
					{ID: "test-plugin", Name: "Test Plugin", Version: "1.0.0"},
				}, nil
			},
			removeFunc: func(id string) error {
				return nil
			},
			saveFunc: func() error {
				return fmt.Errorf("failed to save manifest")
			},
		}

		svc := newTestService(&mockCacheManager{}, manifest, &mockDownloader{}, []PluginSource{})

		result, err := svc.Uninstall(ctx, "test-plugin", UninstallOptions{})

		require.NoError(t, err, "should not fail even if save fails")
		require.NotNil(t, result)
		require.Equal(t, 1, result.RemovedCount)
		require.Len(t, result.Errors, 1, "should collect save error")
		require.Contains(t, result.Errors[0].Error(), "save manifest")
	})
}

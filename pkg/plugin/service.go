// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package plugin

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pentora-ai/pentora/pkg/storage"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Service handles plugin lifecycle operations (install, update, uninstall, list).
//
// The service layer encapsulates all business logic for plugin management,
// allowing both CLI commands and API handlers to reuse the same implementation.
//
// Example usage:
//
//	svc, err := plugin.NewService(cacheDir)
//	if err != nil {
//	    return err
//	}
//
//	result, err := svc.Install(ctx, "ssh", plugin.InstallOptions{})
//	if err != nil {
//	    return err
//	}
//
//	fmt.Printf("Installed %d plugins\n", result.InstalledCount)
type Service struct {
	// Required dependencies
	cache      *CacheManager
	manifest   *ManifestManager
	downloader *Downloader

	// Plugin sources (default or custom)
	sources []PluginSource

	// Optional dependencies (injected via fluent API)
	storage storage.Backend
	logger  zerolog.Logger
}

// NewService creates a new plugin service with default configuration.
//
// Parameters:
//   - cacheDir: Directory for plugin cache (e.g., ~/.pentora/plugins/cache)
//
// Returns a fully configured service with sensible defaults:
//   - CacheManager for managing cached plugins
//   - ManifestManager for tracking installed plugins
//   - Default plugin sources (official repository)
//   - Default logger (zerolog)
//
// Use With* methods to inject optional dependencies (storage, custom logger, custom sources).
//
// Example:
//
//	svc, err := plugin.NewService("/path/to/cache")
//	if err != nil {
//	    return fmt.Errorf("create service: %w", err)
//	}
func NewService(cacheDir string) (*Service, error) {
	if cacheDir == "" {
		// Use default cache directory if not specified
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home directory: %w", err)
		}
		cacheDir = filepath.Join(homeDir, ".pentora", "plugins", "cache")
	}

	// Create cache manager
	cache, err := NewCacheManager(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("create cache manager: %w", err)
	}

	// Create manifest manager (registry.json in parent directory of cache)
	manifestPath := filepath.Join(filepath.Dir(cacheDir), "registry.json")
	manifest, err := NewManifestManager(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("create manifest manager: %w", err)
	}

	// Create service with defaults
	svc := &Service{
		cache:    cache,
		manifest: manifest,
		sources:  defaultSources(),
		logger:   log.Logger,
	}

	// Create downloader with default sources
	svc.downloader = NewDownloader(cache, WithSources(svc.sources))

	return svc, nil
}

// WithStorage injects an optional storage backend for advanced features.
//
// The storage backend enables features like:
//   - Storing plugin metadata in database
//   - Cross-referencing plugins with scan results
//   - Multi-tenancy support (Enterprise)
//
// Example:
//
//	backend, _ := storage.NewBackend(ctx, cfg)
//	svc := plugin.NewService(cacheDir).WithStorage(backend)
func (s *Service) WithStorage(backend storage.Backend) *Service {
	s.storage = backend
	return s
}

// WithLogger injects a custom logger.
//
// Example:
//
//	customLogger := zerolog.New(os.Stdout).With().
//	    Str("service", "plugin").
//	    Logger()
//	svc := plugin.NewService(cacheDir).WithLogger(customLogger)
func (s *Service) WithLogger(logger zerolog.Logger) *Service {
	s.logger = logger
	return s
}

// WithSources injects custom plugin sources.
//
// This allows using alternative plugin repositories or mirrors.
//
// Example:
//
//	customSources := []plugin.PluginSource{
//	    {Name: "enterprise", URL: "https://enterprise.example.com/plugins.yaml", Enabled: true},
//	}
//	svc := plugin.NewService(cacheDir).WithSources(customSources)
func (s *Service) WithSources(sources []PluginSource) *Service {
	s.sources = sources
	// Recreate downloader with new sources
	s.downloader = NewDownloader(s.cache, WithSources(s.sources))
	return s
}

// defaultSources returns the default plugin sources.
//
// By default, we use the official Pentora plugin repository with a GitHub mirror.
func defaultSources() []PluginSource {
	return []PluginSource{
		{
			Name:     "official",
			URL:      "https://plugins.pentora.ai/manifest.yaml",
			Enabled:  true,
			Priority: 1,
		},
	}
}

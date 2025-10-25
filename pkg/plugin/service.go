// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
			Mirrors: []string{
				"https://raw.githubusercontent.com/pentora-ai/pentora-plugins/main/manifest.yaml",
			},
		},
	}
}

// Install installs plugins by category or plugin ID.
//
// The target parameter can be:
//   - A category name (e.g., "ssh", "http", "tls")
//   - A specific plugin ID (e.g., "ssh-default-credentials")
//
// The method:
//  1. Fetches manifests from all enabled sources
//  2. Determines if target is a category or plugin ID
//  3. Downloads and caches matching plugins
//  4. Updates the manifest (registry.json)
//  5. Collects errors for partial failures (doesn't fail fast)
//
// Example:
//
//	result, err := svc.Install(ctx, "ssh", plugin.InstallOptions{})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Installed %d plugins\n", result.InstalledCount)
func (s *Service) Install(ctx context.Context, target string, opts InstallOptions) (*InstallResult, error) {
	s.logger.Info().
		Str("component", "plugin-service").
		Str("operation", "install").
		Str("target", target).
		Bool("force", opts.Force).
		Msg("Starting plugin installation")

	result := &InstallResult{
		Plugins: []*PluginInfo{},
		Errors:  []error{},
	}

	// Fetch manifests from sources
	allPlugins, err := s.fetchPlugins(ctx, opts.Source)
	if err != nil {
		return nil, fmt.Errorf("fetch plugins: %w", err)
	}

	if len(allPlugins) == 0 {
		return nil, fmt.Errorf("%w: no plugins found in any source", ErrNoPluginsFound)
	}

	// Determine if target is category or plugin ID
	var toInstall []PluginManifestEntry

	if opts.Category != "" && opts.Category.IsValid() {
		// Use category from options if specified
		toInstall = s.filterByCategory(allPlugins, opts.Category)
	} else if category := Category(target); category.IsValid() {
		// Target is a category
		toInstall = s.filterByCategory(allPlugins, category)
	} else {
		// Target is a plugin ID
		plugin, err := s.findPluginByID(allPlugins, target)
		if err != nil {
			return nil, err
		}
		toInstall = []PluginManifestEntry{plugin}
	}

	if len(toInstall) == 0 {
		return nil, fmt.Errorf("%w: no plugins match criteria", ErrNoPluginsFound)
	}

	// Install each plugin
	for _, p := range toInstall {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		if err := s.installOne(ctx, p, opts); err != nil {
			// Check if plugin was already installed (not an error)
			if err == ErrPluginAlreadyInstalled {
				result.SkippedCount++
				s.logger.Debug().
					Str("plugin", p.Name).
					Msg("Plugin already installed")
			} else {
				result.FailedCount++
				result.Errors = append(result.Errors, fmt.Errorf("install %s: %w", p.Name, err))
				s.logger.Warn().
					Str("plugin", p.Name).
					Err(err).
					Msg("Failed to install plugin")
			}
		} else {
			result.InstalledCount++
			result.Plugins = append(result.Plugins, pluginInfoFromManifestEntry(&p))
			s.logger.Info().
				Str("plugin", p.Name).
				Str("version", p.Version).
				Msg("Plugin installed successfully")
		}
	}

	s.logger.Info().
		Str("component", "plugin-service").
		Str("operation", "install").
		Int("installed", result.InstalledCount).
		Int("skipped", result.SkippedCount).
		Int("failed", result.FailedCount).
		Msg("Plugin installation completed")

	return result, nil
}

// fetchPlugins fetches plugin manifests from all enabled sources.
func (s *Service) fetchPlugins(ctx context.Context, sourceName string) ([]PluginManifestEntry, error) {
	var allPlugins []PluginManifestEntry

	// Filter sources if specific source is requested
	sources := s.sources
	if sourceName != "" {
		var filteredSources []PluginSource
		for _, src := range s.sources {
			if src.Name == sourceName {
				filteredSources = append(filteredSources, src)
				break
			}
		}
		if len(filteredSources) == 0 {
			return nil, fmt.Errorf("%w: source '%s' not found", ErrSourceNotAvailable, sourceName)
		}
		sources = filteredSources
	}

	// Fetch from each enabled source
	for _, src := range sources {
		if !src.Enabled {
			continue
		}

		manifest, err := s.downloader.FetchManifest(ctx, src)
		if err != nil {
			s.logger.Warn().
				Str("source", src.Name).
				Err(err).
				Msg("Failed to fetch manifest from source")
			continue
		}

		allPlugins = append(allPlugins, manifest.Plugins...)
	}

	return allPlugins, nil
}

// filterByCategory filters plugins by category.
func (s *Service) filterByCategory(plugins []PluginManifestEntry, category Category) []PluginManifestEntry {
	var filtered []PluginManifestEntry

	for _, p := range plugins {
		for _, cat := range p.Categories {
			if cat == category {
				filtered = append(filtered, p)
				break
			}
		}
	}

	return filtered
}

// findPluginByID finds a plugin by its ID (case-insensitive).
func (s *Service) findPluginByID(plugins []PluginManifestEntry, id string) (PluginManifestEntry, error) {
	idLower := strings.ToLower(id)

	for _, p := range plugins {
		if p.ID == idLower {
			return p, nil
		}
	}

	return PluginManifestEntry{}, fmt.Errorf("%w: plugin '%s' not found", ErrPluginNotFound, id)
}

// installOne installs a single plugin.
func (s *Service) installOne(ctx context.Context, p PluginManifestEntry, opts InstallOptions) error {
	// Check if already cached (unless force reinstall)
	if !opts.Force {
		if _, err := s.cache.GetEntry(p.Name, p.Version); err == nil {
			s.logger.Debug().
				Str("plugin", p.Name).
				Str("version", p.Version).
				Msg("Plugin already installed (skipping)")
			return ErrPluginAlreadyInstalled
		}
	}

	// Return early if dry run
	if opts.DryRun {
		s.logger.Info().
			Str("plugin", p.Name).
			Bool("dry_run", true).
			Msg("Would install plugin (dry run)")
		return nil
	}

	// Download plugin
	_, err := s.downloader.Download(ctx, p.ID, p.Version)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}

	// Prepare manifest entry
	categoryTags := make([]string, len(p.Categories))
	for i, cat := range p.Categories {
		categoryTags[i] = string(cat)
	}

	manifestEntry := &ManifestEntry{
		ID:          p.ID,
		Name:        p.Name,
		Version:     p.Version,
		Type:        "evaluation", // Default type
		Author:      p.Author,
		Checksum:    p.Checksum,
		DownloadURL: p.URL,
		InstalledAt: time.Now(),
		Path:        filepath.Join(p.ID, p.Version, "plugin.yaml"),
		Tags:        categoryTags,
		Severity:    "medium", // Default severity (overridden when plugin loads)
	}

	// Add to manifest
	if err := s.manifest.Add(manifestEntry); err != nil {
		s.logger.Warn().
			Str("plugin", p.Name).
			Err(err).
			Msg("Failed to add to manifest (plugin still downloaded)")
		// Don't return error - plugin is downloaded, manifest update is best-effort
	}

	// Save manifest
	if err := s.manifest.Save(); err != nil {
		s.logger.Warn().
			Err(err).
			Msg("Failed to save manifest")
		// Don't return error - plugin is downloaded
	}

	return nil
}

// pluginInfoFromManifestEntry converts a PluginManifestEntry to PluginInfo.
func pluginInfoFromManifestEntry(entry *PluginManifestEntry) *PluginInfo {
	tags := make([]string, len(entry.Categories))
	for i, cat := range entry.Categories {
		tags[i] = string(cat)
	}

	return &PluginInfo{
		ID:          entry.ID,
		Name:        entry.Name,
		Version:     entry.Version,
		Author:      entry.Author,
		Checksum:    entry.Checksum,
		DownloadURL: entry.URL,
		Tags:        tags,
		InstalledAt: time.Now(),
	}
}

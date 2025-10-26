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

// Interfaces for dependency injection (useful for testing)

// CacheInterface defines the cache operations needed by Service
type CacheInterface interface {
	GetEntry(name, version string) (*CacheEntry, error)
	Size() (int64, error)
	Prune(olderThan time.Duration) (int, error)
	Remove(id string, version string) error
}

// ManifestInterface defines the manifest operations needed by Service
type ManifestInterface interface {
	Add(entry *ManifestEntry) error
	Save() error
	List() ([]*ManifestEntry, error)
	Remove(id string) error
	Get(id string) (*ManifestEntry, error)
}

// DownloaderInterface defines the downloader operations needed by Service
type DownloaderInterface interface {
	FetchManifest(ctx context.Context, src PluginSource) (*PluginManifest, error)
	Download(ctx context.Context, id, version string) (*CacheEntry, error)
}

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
	// Required dependencies (interfaces for testability)
	cache      CacheInterface
	manifest   ManifestInterface
	downloader DownloaderInterface

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
//
// Note: This only updates the sources list. If you need to recreate the downloader
// with new sources, create a new Service instance instead.
func (s *Service) WithSources(sources []PluginSource) *Service {
	s.sources = sources
	// Note: We don't recreate the downloader here because s.cache is an interface.
	// In practice, sources are set during initialization via NewService.
	// If you need to change sources after creation, create a new Service instance.
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
	// Validate inputs (defense-in-depth)
	if err := validateTarget(target); err != nil {
		return nil, err
	}
	if err := validateCategory(opts.Category); err != nil {
		return nil, err
	}
	if err := validateSource(opts.Source); err != nil {
		return nil, err
	}

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

	// Return partial failure if any plugins failed
	if result.FailedCount > 0 {
		return result, ErrPartialFailure
	}

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

// Update updates plugins from remote repositories.
//
// Unlike Install which targets specific plugins or categories, Update fetches
// the latest manifest and downloads all available plugins (or those matching
// the specified category).
//
// Parameters:
//   - ctx: Context for cancellation
//   - opts: Update options (source, category, force, dry-run)
//
// Returns:
//   - UpdateResult with counts and updated plugin info
//   - Error if manifest fetch fails or all downloads fail
//
// Behavior:
//   - Fetches manifests from all sources (or specific source if opts.Source is set)
//   - Filters by category if opts.Category is set
//   - Skips already cached plugins unless opts.Force is true
//   - In dry-run mode (opts.DryRun=true), simulates update without downloading
//   - Collects errors but doesn't fail fast - returns partial results
//
// Example:
//
//	result, err := svc.Update(ctx, UpdateOptions{Category: CategorySSH})
//	fmt.Printf("Updated %d plugins\n", result.UpdatedCount)
func (s *Service) Update(ctx context.Context, opts UpdateOptions) (*UpdateResult, error) {
	// Validate inputs (defense-in-depth)
	if err := validateCategory(opts.Category); err != nil {
		return nil, err
	}
	if err := validateSource(opts.Source); err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("component", "plugin-service").
		Str("operation", "update").
		Str("source", opts.Source).
		Str("category", string(opts.Category)).
		Bool("force", opts.Force).
		Bool("dry_run", opts.DryRun).
		Msg("Starting plugin update")

	result := &UpdateResult{
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

	// Filter by category if specified
	var toUpdate []PluginManifestEntry
	if opts.Category != "" && opts.Category.IsValid() {
		toUpdate = s.filterByCategory(allPlugins, opts.Category)
	} else {
		toUpdate = allPlugins
	}

	if len(toUpdate) == 0 {
		return nil, fmt.Errorf("%w: no plugins match criteria", ErrNoPluginsFound)
	}

	s.logger.Debug().
		Int("total_plugins", len(toUpdate)).
		Msg("Plugins to update")

	// Update each plugin
	for _, p := range toUpdate {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		// Check if already cached (unless force)
		if !opts.Force {
			if _, err := s.cache.GetEntry(p.Name, p.Version); err == nil {
				result.SkippedCount++
				s.logger.Debug().
					Str("plugin", p.Name).
					Str("version", p.Version).
					Msg("Plugin already cached, skipping")
				continue
			}
		}

		// Dry run mode
		if opts.DryRun {
			result.UpdatedCount++
			result.Plugins = append(result.Plugins, pluginInfoFromManifestEntry(&p))
			s.logger.Info().
				Str("plugin", p.Name).
				Bool("dry_run", true).
				Msg("Would update plugin (dry run)")
			continue
		}

		// Download plugin
		_, err := s.downloader.Download(ctx, p.ID, p.Version)
		if err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Errorf("update %s: %w", p.Name, err))
			s.logger.Warn().
				Str("plugin", p.Name).
				Err(err).
				Msg("Failed to update plugin")
			continue
		}

		// Add to manifest
		categoryTags := make([]string, len(p.Categories))
		for i, cat := range p.Categories {
			categoryTags[i] = string(cat)
		}

		manifestEntry := &ManifestEntry{
			ID:          p.ID,
			Name:        p.Name,
			Version:     p.Version,
			Type:        "evaluation",
			Author:      p.Author,
			Checksum:    p.Checksum,
			DownloadURL: p.URL,
			InstalledAt: time.Now(),
			Path:        filepath.Join(p.ID, p.Version, "plugin.yaml"),
			Tags:        categoryTags,
			Severity:    "medium",
		}

		if err := s.manifest.Add(manifestEntry); err != nil {
			s.logger.Warn().
				Str("plugin", p.Name).
				Err(err).
				Msg("Failed to add to manifest (plugin still downloaded)")
		}

		if err := s.manifest.Save(); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to save manifest")
		}

		result.UpdatedCount++
		result.Plugins = append(result.Plugins, pluginInfoFromManifestEntry(&p))
		s.logger.Info().
			Str("plugin", p.Name).
			Str("version", p.Version).
			Msg("Plugin updated successfully")
	}

	s.logger.Info().
		Str("component", "plugin-service").
		Str("operation", "update").
		Int("updated", result.UpdatedCount).
		Int("skipped", result.SkippedCount).
		Int("failed", result.FailedCount).
		Msg("Plugin update completed")

	// Return partial failure if any plugins failed
	if result.FailedCount > 0 {
		return result, ErrPartialFailure
	}

	return result, nil
}

// Uninstall removes plugins from the cache and manifest.
//
// Supports three modes:
//   - Uninstall specific plugin by ID
//   - Uninstall all plugins in a category (opts.Category)
//   - Uninstall all plugins (opts.All)
//
// Parameters:
//   - ctx: Context for cancellation
//   - target: Plugin ID to uninstall (ignored if opts.All or opts.Category is set)
//   - opts: Uninstall options (All, Category)
//
// Returns:
//   - UninstallResult with counts and remaining plugins
//   - Error if validation fails or all uninstalls fail
//
// Behavior:
//   - Validates that only one mode is specified (target XOR category XOR all)
//   - Lists installed plugins from manifest
//   - Filters by target/category/all
//   - Removes plugin files from cache directory
//   - Removes entries from manifest
//   - Saves manifest to disk
//   - Collects errors but doesn't fail fast
//
// Example:
//
//	// Uninstall specific plugin
//	result, err := svc.Uninstall(ctx, "ssh-plugin", UninstallOptions{})
//
//	// Uninstall all SSH plugins
//	result, err := svc.Uninstall(ctx, "", UninstallOptions{Category: CategorySSH})
//
//	// Uninstall all plugins
//	result, err := svc.Uninstall(ctx, "", UninstallOptions{All: true})
func (s *Service) Uninstall(ctx context.Context, target string, opts UninstallOptions) (*UninstallResult, error) {
	// Validate inputs (defense-in-depth)
	// Target is optional when using category or all flags
	if target != "" {
		if err := validateTarget(target); err != nil {
			return nil, err
		}
	}
	if err := validateCategory(opts.Category); err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("component", "plugin-service").
		Str("operation", "uninstall").
		Str("target", target).
		Bool("all", opts.All).
		Str("category", string(opts.Category)).
		Msg("Starting plugin uninstall")

	result := &UninstallResult{
		Errors: []error{},
	}

	// Validate input - only one mode allowed
	hasTarget := target != ""
	hasCategory := opts.Category != "" && opts.Category.IsValid()
	hasAll := opts.All

	modesCount := 0
	if hasTarget {
		modesCount++
	}
	if hasCategory {
		modesCount++
	}
	if hasAll {
		modesCount++
	}

	if modesCount == 0 {
		return nil, fmt.Errorf("%w: must specify plugin ID, category, or --all", ErrInvalidInput)
	}

	if modesCount > 1 {
		return nil, fmt.Errorf("%w: cannot specify multiple uninstall modes", ErrInvalidInput)
	}

	// Get installed plugins from manifest
	entries, err := s.manifest.List()
	if err != nil {
		return nil, fmt.Errorf("list installed plugins: %w", err)
	}

	if len(entries) == 0 {
		s.logger.Info().Msg("No plugins installed")
		return result, nil
	}

	// Determine which plugins to uninstall
	var toUninstall []*ManifestEntry

	if hasAll {
		toUninstall = entries
		s.logger.Info().Int("count", len(entries)).Msg("Uninstalling all plugins")
	} else if hasCategory {
		toUninstall = s.filterManifestByCategory(entries, opts.Category)
		if len(toUninstall) == 0 {
			return nil, fmt.Errorf("%w: no plugins found in category '%s'", ErrNoPluginsFound, opts.Category)
		}
		s.logger.Info().
			Str("category", string(opts.Category)).
			Int("count", len(toUninstall)).
			Msg("Uninstalling plugins by category")
	} else {
		// Uninstall specific plugin by ID
		targetLower := strings.ToLower(target)
		found := false
		for _, entry := range entries {
			if entry.ID == targetLower {
				toUninstall = append(toUninstall, entry)
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("%w: plugin '%s' not found (not installed)", ErrPluginNotFound, target)
		}
		s.logger.Info().Str("plugin", target).Msg("Uninstalling specific plugin")
	}

	// Uninstall each plugin
	for _, entry := range toUninstall {
		select {
		case <-ctx.Done():
			result.RemainingCount = len(entries) - result.RemovedCount
			return result, ctx.Err()
		default:
		}

		if err := s.uninstallOne(entry); err != nil {
			result.FailedCount++
			result.Errors = append(result.Errors, fmt.Errorf("uninstall %s: %w", entry.Name, err))
			s.logger.Warn().
				Str("plugin", entry.Name).
				Err(err).
				Msg("Failed to uninstall plugin")
			continue
		}

		result.RemovedCount++
		s.logger.Info().
			Str("plugin", entry.Name).
			Str("version", entry.Version).
			Msg("Plugin uninstalled successfully")
	}

	// Save manifest to disk if any plugins were removed
	if result.RemovedCount > 0 {
		if err := s.manifest.Save(); err != nil {
			s.logger.Warn().Err(err).Msg("Failed to save manifest after uninstall")
			result.Errors = append(result.Errors, fmt.Errorf("save manifest: %w", err))
		}
	}

	result.RemainingCount = len(entries) - result.RemovedCount

	s.logger.Info().
		Str("component", "plugin-service").
		Str("operation", "uninstall").
		Int("removed", result.RemovedCount).
		Int("failed", result.FailedCount).
		Int("remaining", result.RemainingCount).
		Msg("Plugin uninstall completed")

	// Return partial failure if any plugins failed
	if result.FailedCount > 0 {
		return result, ErrPartialFailure
	}
	return result, nil
}

// uninstallOne removes a single plugin from cache and manifest
func (s *Service) uninstallOne(entry *ManifestEntry) error {
	// Remove plugin files from cache directory
	if err := s.cache.Remove(entry.ID, entry.Version); err != nil {
		s.logger.Warn().
			Err(err).
			Str("plugin_id", entry.ID).
			Str("version", entry.Version).
			Msg("Failed to remove plugin from cache (will still remove from manifest)")
		// Continue even if cache removal fails - we still want to update manifest
	}

	// Remove from manifest (registry)
	if err := s.manifest.Remove(entry.ID); err != nil {
		return fmt.Errorf("remove from manifest: %w", err)
	}

	return nil
}

// filterManifestByCategory filters manifest entries by category
func (s *Service) filterManifestByCategory(entries []*ManifestEntry, category Category) []*ManifestEntry {
	var filtered []*ManifestEntry
	categoryStr := string(category)

	for _, entry := range entries {
		for _, tag := range entry.Tags {
			if tag == categoryStr {
				filtered = append(filtered, entry)
				break
			}
		}
	}

	return filtered
}

// List returns information about all installed plugins.
//
// This method retrieves all plugins from the manifest and converts them
// to PluginInfo structs with detailed metadata.
//
// Returns:
//   - []*PluginInfo: List of installed plugins
//   - error: Any error encountered during operation
//
// Example:
//
//	plugins, err := svc.List(ctx)
//	if err != nil {
//	    return fmt.Errorf("list plugins: %w", err)
//	}
//	fmt.Printf("Found %d installed plugins\n", len(plugins))
func (s *Service) List(ctx context.Context) ([]*PluginInfo, error) {
	s.logger.Debug().
		Str("component", "plugin-service").
		Str("operation", "list").
		Msg("Listing installed plugins")

	// Get all entries from manifest
	entries, err := s.manifest.List()
	if err != nil {
		s.logger.Error().
			Err(err).
			Msg("Failed to list manifest entries")
		return nil, fmt.Errorf("list manifest: %w", err)
	}

	// Convert to PluginInfo
	plugins := make([]*PluginInfo, 0, len(entries))
	for _, entry := range entries {
		// Check context cancellation
		select {
		case <-ctx.Done():
			s.logger.Warn().Msg("List operation cancelled")
			return nil, ctx.Err()
		default:
		}

		info := &PluginInfo{
			ID:           entry.ID,
			Name:         entry.Name,
			Version:      entry.Version,
			Type:         entry.Type,
			Author:       entry.Author,
			Severity:     entry.Severity,
			Tags:         entry.Tags,
			Checksum:     entry.Checksum,
			DownloadURL:  entry.DownloadURL,
			InstalledAt:  entry.InstalledAt,
			LastVerified: entry.LastVerified,
			Path:         entry.Path,
			// CacheDir and CacheSize not calculated for list (performance)
		}
		plugins = append(plugins, info)
	}

	s.logger.Info().
		Str("component", "plugin-service").
		Str("operation", "list").
		Int("count", len(plugins)).
		Msg("Plugin list completed")

	return plugins, nil
}

// GetInfo returns detailed information about a specific plugin.
//
// This method retrieves plugin metadata from the manifest and calculates
// additional information like cache directory size.
//
// Parameters:
//   - ctx: Context for cancellation
//   - pluginID: Plugin identifier (slug)
//
// Returns:
//   - *PluginInfo: Detailed plugin information
//   - error: ErrPluginNotFound if plugin not installed, or other errors
//
// Example:
//
//	info, err := svc.GetInfo(ctx, "ssh-weak-cipher")
//	if plugin.IsNotFound(err) {
//	    fmt.Println("Plugin not installed")
//	    return
//	}
//	fmt.Printf("Plugin: %s v%s (Size: %d bytes)\n", info.Name, info.Version, info.CacheSize)
func (s *Service) GetInfo(ctx context.Context, pluginID string) (*PluginInfo, error) {
	// Validate inputs (defense-in-depth)
	if err := validatePluginID(pluginID); err != nil {
		return nil, err
	}

	s.logger.Debug().
		Str("component", "plugin-service").
		Str("operation", "get-info").
		Str("plugin_id", pluginID).
		Msg("Getting plugin info")

	// Get all entries from manifest
	entries, err := s.manifest.List()
	if err != nil {
		s.logger.Error().
			Err(err).
			Msg("Failed to list manifest entries")
		return nil, fmt.Errorf("list manifest: %w", err)
	}

	// Find the plugin
	var entry *ManifestEntry
	for _, e := range entries {
		if e.ID == pluginID {
			entry = e
			break
		}
	}

	if entry == nil {
		s.logger.Warn().
			Str("plugin_id", pluginID).
			Msg("Plugin not found in manifest")
		return nil, ErrPluginNotFound
	}

	// Check context cancellation
	select {
	case <-ctx.Done():
		s.logger.Warn().Msg("GetInfo operation cancelled")
		return nil, ctx.Err()
	default:
	}

	// Build PluginInfo with basic metadata
	info := &PluginInfo{
		ID:           entry.ID,
		Name:         entry.Name,
		Version:      entry.Version,
		Type:         entry.Type,
		Author:       entry.Author,
		Severity:     entry.Severity,
		Tags:         entry.Tags,
		Checksum:     entry.Checksum,
		DownloadURL:  entry.DownloadURL,
		InstalledAt:  entry.InstalledAt,
		LastVerified: entry.LastVerified,
		Path:         entry.Path,
	}

	// Calculate cache directory and size
	// Cache structure: <cacheDir>/<plugin-id>/<version>/
	cacheDir := filepath.Join(entry.Path, "..", "..") // Go up from plugin.yaml to cache root
	cacheDir, err = filepath.Abs(cacheDir)
	if err != nil {
		s.logger.Warn().
			Err(err).
			Str("path", entry.Path).
			Msg("Failed to resolve cache directory path")
		// Continue without cache info
	} else {
		info.CacheDir = cacheDir

		// Calculate directory size
		size, err := calculateDirSize(cacheDir)
		if err != nil {
			s.logger.Warn().
				Err(err).
				Str("cache_dir", cacheDir).
				Msg("Failed to calculate cache directory size")
			// Continue without size info
		} else {
			info.CacheSize = size
		}
	}

	s.logger.Info().
		Str("component", "plugin-service").
		Str("operation", "get-info").
		Str("plugin_id", pluginID).
		Str("version", info.Version).
		Int64("cache_size", info.CacheSize).
		Msg("Plugin info retrieved successfully")

	return info, nil
}

// calculateDirSize recursively calculates the total size of a directory in bytes.
//
// Parameters:
//   - path: Directory path to calculate size for
//
// Returns:
//   - int64: Total size in bytes
//   - error: Any error encountered during traversal
func calculateDirSize(path string) (int64, error) {
	var size int64

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			// Log but continue on individual file errors
			return nil
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("walk directory: %w", err)
	}

	return size, nil
}

// Clean removes old plugin cache entries based on age.
//
// Example:
//
//	result, err := svc.Clean(ctx, CleanOptions{
//	    OlderThan: 720 * time.Hour, // 30 days
//	    DryRun: true,
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Would free %d bytes\n", result.Freed)
func (s *Service) Clean(ctx context.Context, opts CleanOptions) (*CleanResult, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("component", "plugin-service").
		Str("operation", "clean").
		Dur("older_than", opts.OlderThan).
		Bool("dry_run", opts.DryRun).
		Msg("Cleaning plugin cache")

	// Calculate size before cleaning
	sizeBefore, err := s.cache.Size()
	if err != nil {
		s.logger.Debug().Err(err).Msg("Failed to calculate cache size before cleaning")
		sizeBefore = 0
	}

	// Dry run: return early without actually pruning
	if opts.DryRun {
		result := &CleanResult{
			RemovedCount: 0,
			SizeBefore:   sizeBefore,
			SizeAfter:    sizeBefore,
			Freed:        0,
		}
		s.logger.Info().
			Int("removed", 0).
			Int64("freed", 0).
			Msg("Cache cleaning completed")
		return result, nil
	}

	// Run prune operation
	removed, err := s.cache.Prune(opts.OlderThan)
	if err != nil {
		return nil, fmt.Errorf("prune cache: %w", err)
	}

	// Calculate size after cleaning
	sizeAfter, err := s.cache.Size()
	if err != nil {
		s.logger.Debug().Err(err).Msg("Failed to calculate cache size after cleaning")
		sizeAfter = 0
	}

	freed := sizeBefore - sizeAfter

	result := &CleanResult{
		RemovedCount: removed,
		SizeBefore:   sizeBefore,
		SizeAfter:    sizeAfter,
		Freed:        freed,
	}

	s.logger.Info().
		Int("removed", removed).
		Int64("freed", freed).
		Msg("Cache cleaning completed")

	return result, nil
}

// Verify checks the integrity of installed plugins by verifying their checksums.
//
// Example:
//
//	result, err := svc.Verify(ctx, VerifyOptions{
//	    PluginID: "ssh-cve-2024-6387", // Or empty for all plugins
//	})
//	if err != nil {
//	    return err
//	}
//	fmt.Printf("Verified %d plugins, %d failed\n", result.TotalCount, result.FailedCount)
func (s *Service) Verify(ctx context.Context, opts VerifyOptions) (*VerifyResult, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.logger.Info().
		Str("component", "plugin-service").
		Str("operation", "verify").
		Str("plugin_id", opts.PluginID).
		Msg("Verifying plugin checksums")

	// Get plugins to verify
	var entries []*ManifestEntry
	if opts.PluginID != "" {
		// Verify specific plugin
		entry, err := s.manifest.Get(opts.PluginID)
		if err != nil {
			return nil, ErrPluginNotFound
		}
		entries = []*ManifestEntry{entry}
	} else {
		// Verify all plugins
		allEntries, err := s.manifest.List()
		if err != nil {
			return nil, fmt.Errorf("list plugins: %w", err)
		}
		entries = allEntries
	}

	// Create verifier
	verifier := NewVerifier()

	// Verify each plugin
	results := make([]PluginVerifyResult, 0, len(entries))
	successCount := 0

	for _, entry := range entries {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		result := PluginVerifyResult{
			ID:      entry.ID,
			Version: entry.Version,
		}

		// Get plugin file path
		pluginFile, err := s.cache.GetEntry(entry.ID, entry.Version)
		if err != nil {
			result.Valid = false
			result.Error = fmt.Errorf("file not found")
			result.ErrorType = "missing"
			results = append(results, result)
			continue
		}

		// Verify checksum
		valid, err := verifier.VerifyFile(pluginFile.Path, entry.Checksum)
		if err != nil {
			result.Valid = false
			result.Error = err
			result.ErrorType = "error"
			results = append(results, result)
			continue
		}

		if !valid {
			result.Valid = false
			result.Error = fmt.Errorf("checksum mismatch")
			result.ErrorType = "checksum"
			results = append(results, result)
			continue
		}

		// Verification successful
		result.Valid = true
		successCount++
		results = append(results, result)
	}

	verifyResult := &VerifyResult{
		TotalCount:   len(entries),
		SuccessCount: successCount,
		FailedCount:  len(entries) - successCount,
		Results:      results,
	}

	s.logger.Info().
		Int("total", verifyResult.TotalCount).
		Int("success", verifyResult.SuccessCount).
		Int("failed", verifyResult.FailedCount).
		Msg("Plugin verification completed")

	return verifyResult, nil
}

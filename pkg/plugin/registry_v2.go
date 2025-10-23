// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package plugin

import (
	"fmt"
	"sync"
)

// RegistryV2 manages YAML-based plugins with metadata, categories, and caching.
// This is the OSS in-memory implementation.
// Enterprise can replace with distributed registry backed by database.
type RegistryV2 struct {
	// In-memory plugin storage (plugin.Name -> *YAMLPlugin)
	plugins map[string]*YAMLPlugin

	// Category index (category -> []plugin names)
	categories map[string][]string

	// Metadata index (for quick lookups)
	metadata map[string]*PluginMetadata

	// Thread-safe access
	mu sync.RWMutex
}

// NewRegistryV2 creates a new plugin registry.
func NewRegistryV2() *RegistryV2 {
	return &RegistryV2{
		plugins:    make(map[string]*YAMLPlugin),
		categories: make(map[string][]string),
		metadata:   make(map[string]*PluginMetadata),
	}
}

// Register adds a plugin to the registry.
// Returns error if plugin with same name already exists.
func (r *RegistryV2) Register(plugin *YAMLPlugin) error {
	if plugin == nil {
		return fmt.Errorf("cannot register nil plugin")
	}

	if plugin.Name == "" {
		return fmt.Errorf("plugin name cannot be empty")
	}

	// Validate plugin before registration
	if err := plugin.Validate(); err != nil {
		return fmt.Errorf("plugin validation failed: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicates
	if _, exists := r.plugins[plugin.Name]; exists {
		return fmt.Errorf("plugin '%s' already registered", plugin.Name)
	}

	// Register plugin
	r.plugins[plugin.Name] = plugin
	r.metadata[plugin.Name] = &plugin.Metadata

	// Index by category (using tags as categories)
	for _, tag := range plugin.Metadata.Tags {
		r.categories[tag] = append(r.categories[tag], plugin.Name)
	}

	return nil
}

// Unregister removes a plugin from the registry.
func (r *RegistryV2) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	plugin, exists := r.plugins[name]
	if !exists {
		return fmt.Errorf("plugin '%s' not found", name)
	}

	// Remove from category index
	for _, tag := range plugin.Metadata.Tags {
		r.removeFromCategory(tag, name)
	}

	// Remove from registry
	delete(r.plugins, name)
	delete(r.metadata, name)

	return nil
}

// Get retrieves a plugin by name.
func (r *RegistryV2) Get(name string) (*YAMLPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, exists := r.plugins[name]
	return plugin, exists
}

// List returns all registered plugins.
func (r *RegistryV2) List() []*YAMLPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugins := make([]*YAMLPlugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// ListByCategory returns plugins belonging to a specific category (tag).
func (r *RegistryV2) ListByCategory(category string) []*YAMLPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names, exists := r.categories[category]
	if !exists {
		return []*YAMLPlugin{}
	}

	plugins := make([]*YAMLPlugin, 0, len(names))
	for _, name := range names {
		if plugin, ok := r.plugins[name]; ok {
			plugins = append(plugins, plugin)
		}
	}

	return plugins
}

// Categories returns all available categories with plugin counts.
func (r *RegistryV2) Categories() map[string]int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]int)
	for category, plugins := range r.categories {
		result[category] = len(plugins)
	}

	return result
}

// Count returns the total number of registered plugins.
func (r *RegistryV2) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.plugins)
}

// Clear removes all plugins from the registry.
func (r *RegistryV2) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.plugins = make(map[string]*YAMLPlugin)
	r.categories = make(map[string][]string)
	r.metadata = make(map[string]*PluginMetadata)
}

// RegisterBulk registers multiple plugins at once.
// Returns the number of successfully registered plugins and any errors encountered.
func (r *RegistryV2) RegisterBulk(plugins []*YAMLPlugin) (int, []error) {
	var errors []error
	successCount := 0

	for _, plugin := range plugins {
		if err := r.Register(plugin); err != nil {
			errors = append(errors, fmt.Errorf("%s: %w", plugin.Name, err))
		} else {
			successCount++
		}
	}

	return successCount, errors
}

// removeFromCategory removes a plugin name from a category slice.
// Must be called with lock held.
func (r *RegistryV2) removeFromCategory(category, name string) {
	names, exists := r.categories[category]
	if !exists {
		return
	}

	// Remove name from slice
	for i, n := range names {
		if n == name {
			r.categories[category] = append(names[:i], names[i+1:]...)
			break
		}
	}

	// Remove category if empty
	if len(r.categories[category]) == 0 {
		delete(r.categories, category)
	}
}

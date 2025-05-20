// pkg/plugin/plugin.go
// Package plugin provides registration and execution of pluggable scan modules.
// Plugins are registered at init time and managed by a central registry.
package plugin

import (
	"context"
	"fmt"
	"sync"
)

// Plugin defines the interface all scanning plugins must implement.
type Plugin interface {
	Name() string
	Init(ctx context.Context) error
	Run(ctx context.Context, pc PluginContext) error
	Tags() []string
	DependsOn() []string
}

type RegistryInterface interface {
	Register(p Plugin)
	Get(name string) (Plugin, bool)
	All() []Plugin
}

// Registry holds all registered plugins and provides access to them.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
}

// GlobalRegistry is the shared default plugin registry.
var GlobalRegistry = NewRegistry()

// NewRegistry creates a new, empty plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]Plugin),
	}
}

// Register adds a new plugin to the registry.
func (r *Registry) Register(p Plugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := p.Name()
	if _, exists := r.plugins[name]; exists {
		panic(fmt.Sprintf("plugin with name '%s' already registered", name))
	}
	r.plugins[name] = p
}

// Get retrieves a plugin by name.
func (r *Registry) Get(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[name]
	return p, ok
}

// All returns a copy of all registered plugins.
func (r *Registry) All() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []Plugin
	for _, p := range r.plugins {
		list = append(list, p)
	}
	return list
}

// FilterByTag returns all plugins that include the specified tag.
func (r *Registry) FilterByTag(tag string) []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []Plugin
	for _, p := range r.plugins {
		for _, t := range p.Tags() {
			if t == tag {
				matched = append(matched, p)
				break
			}
		}
	}
	return matched
}

// FilterByTags returns all plugins that match at least one of the given tags.
func (r *Registry) FilterByTags(tags []string) []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tagSet := make(map[string]struct{})
	for _, tag := range tags {
		tagSet[tag] = struct{}{}
	}

	var matched []Plugin
	for _, p := range r.plugins {
		for _, t := range p.Tags() {
			if _, ok := tagSet[t]; ok {
				matched = append(matched, p)
				break
			}
		}
	}
	return matched
}

// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package plugin

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistryV2_NewRegistry(t *testing.T) {
	r := NewRegistryV2()
	require.NotNil(t, r)
	require.Equal(t, 0, r.Count())
}

func TestRegistryV2_Register(t *testing.T) {
	r := NewRegistryV2()

	plugin := &YAMLPlugin{
		Name:    "test-plugin",
		Version: "1.0.0",
		Type:    EvaluationType,
		Author:  "test",
		Metadata: PluginMetadata{
			Severity: HighSeverity,
			Tags:     []string{"ssh", "security"},
		},
		Output: OutputBlock{
			Message: "Test",
		},
	}

	err := r.Register(plugin)
	require.NoError(t, err)
	require.Equal(t, 1, r.Count())

	// Verify plugin can be retrieved
	retrieved, ok := r.Get("test-plugin")
	require.True(t, ok)
	require.Equal(t, plugin.Name, retrieved.Name)
}

func TestRegistryV2_Register_NilPlugin(t *testing.T) {
	r := NewRegistryV2()

	err := r.Register(nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "cannot register nil plugin")
}

func TestRegistryV2_Register_EmptyName(t *testing.T) {
	r := NewRegistryV2()

	plugin := &YAMLPlugin{
		Name:    "", // Empty name
		Version: "1.0.0",
		Type:    EvaluationType,
		Author:  "test",
		Metadata: PluginMetadata{
			Severity: HighSeverity,
		},
		Output: OutputBlock{
			Message: "Test",
		},
	}

	err := r.Register(plugin)
	require.Error(t, err)
	require.Contains(t, err.Error(), "name cannot be empty")
}

func TestRegistryV2_Register_InvalidPlugin(t *testing.T) {
	r := NewRegistryV2()

	plugin := &YAMLPlugin{
		Name: "invalid-plugin",
		// Missing required fields for validation
	}

	err := r.Register(plugin)
	require.Error(t, err)
	require.Contains(t, err.Error(), "validation failed")
}

func TestRegistryV2_Register_Duplicate(t *testing.T) {
	r := NewRegistryV2()

	plugin := &YAMLPlugin{
		Name:    "test-plugin",
		Version: "1.0.0",
		Type:    EvaluationType,
		Author:  "test",
		Metadata: PluginMetadata{
			Severity: HighSeverity,
		},
		Output: OutputBlock{
			Message: "Test",
		},
	}

	// First registration should succeed
	err := r.Register(plugin)
	require.NoError(t, err)

	// Second registration should fail
	err = r.Register(plugin)
	require.Error(t, err)
	require.Contains(t, err.Error(), "already registered")
}

func TestRegistryV2_Unregister(t *testing.T) {
	r := NewRegistryV2()

	plugin := &YAMLPlugin{
		Name:    "test-plugin",
		Version: "1.0.0",
		Type:    EvaluationType,
		Author:  "test",
		Metadata: PluginMetadata{
			Severity: HighSeverity,
			Tags:     []string{"ssh"},
		},
		Output: OutputBlock{
			Message: "Test",
		},
	}

	err := r.Register(plugin)
	require.NoError(t, err)
	require.Equal(t, 1, r.Count())

	// Unregister
	err = r.Unregister("test-plugin")
	require.NoError(t, err)
	require.Equal(t, 0, r.Count())

	// Verify plugin is gone
	_, ok := r.Get("test-plugin")
	require.False(t, ok)
}

func TestRegistryV2_Unregister_NotFound(t *testing.T) {
	r := NewRegistryV2()

	err := r.Unregister("non-existent")
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestRegistryV2_Get(t *testing.T) {
	r := NewRegistryV2()

	plugin := &YAMLPlugin{
		Name:    "test-plugin",
		Version: "1.0.0",
		Type:    EvaluationType,
		Author:  "test",
		Metadata: PluginMetadata{
			Severity: HighSeverity,
		},
		Output: OutputBlock{
			Message: "Test",
		},
	}

	err := r.Register(plugin)
	require.NoError(t, err)

	// Get existing
	retrieved, ok := r.Get("test-plugin")
	require.True(t, ok)
	require.Equal(t, "test-plugin", retrieved.Name)

	// Get non-existing
	_, ok = r.Get("non-existent")
	require.False(t, ok)
}

func TestRegistryV2_List(t *testing.T) {
	r := NewRegistryV2()

	// Register multiple plugins
	plugins := []*YAMLPlugin{
		{
			Name:    "plugin1",
			Version: "1.0.0",
			Type:    EvaluationType,
			Author:  "test",
			Metadata: PluginMetadata{
				Severity: HighSeverity,
			},
			Output: OutputBlock{Message: "Test1"},
		},
		{
			Name:    "plugin2",
			Version: "1.0.0",
			Type:    EvaluationType,
			Author:  "test",
			Metadata: PluginMetadata{
				Severity: MediumSeverity,
			},
			Output: OutputBlock{Message: "Test2"},
		},
	}

	for _, p := range plugins {
		err := r.Register(p)
		require.NoError(t, err)
	}

	// List all
	list := r.List()
	require.Len(t, list, 2)
}

func TestRegistryV2_ListByCategory(t *testing.T) {
	r := NewRegistryV2()

	// Register plugins with different categories
	plugin1 := &YAMLPlugin{
		Name:    "ssh-plugin",
		Version: "1.0.0",
		Type:    EvaluationType,
		Author:  "test",
		Metadata: PluginMetadata{
			Severity: HighSeverity,
			Tags:     []string{"ssh", "security"},
		},
		Output: OutputBlock{Message: "Test"},
	}

	plugin2 := &YAMLPlugin{
		Name:    "http-plugin",
		Version: "1.0.0",
		Type:    EvaluationType,
		Author:  "test",
		Metadata: PluginMetadata{
			Severity: MediumSeverity,
			Tags:     []string{"http", "web"},
		},
		Output: OutputBlock{Message: "Test"},
	}

	err := r.Register(plugin1)
	require.NoError(t, err)
	err = r.Register(plugin2)
	require.NoError(t, err)

	// List by category
	sshPlugins := r.ListByCategory("ssh")
	require.Len(t, sshPlugins, 1)
	require.Equal(t, "ssh-plugin", sshPlugins[0].Name)

	httpPlugins := r.ListByCategory("http")
	require.Len(t, httpPlugins, 1)
	require.Equal(t, "http-plugin", httpPlugins[0].Name)

	// Non-existent category
	empty := r.ListByCategory("nonexistent")
	require.Len(t, empty, 0)
}

func TestRegistryV2_Categories(t *testing.T) {
	r := NewRegistryV2()

	plugin1 := &YAMLPlugin{
		Name:    "plugin1",
		Version: "1.0.0",
		Type:    EvaluationType,
		Author:  "test",
		Metadata: PluginMetadata{
			Severity: HighSeverity,
			Tags:     []string{"ssh", "security"},
		},
		Output: OutputBlock{Message: "Test"},
	}

	plugin2 := &YAMLPlugin{
		Name:    "plugin2",
		Version: "1.0.0",
		Type:    EvaluationType,
		Author:  "test",
		Metadata: PluginMetadata{
			Severity: MediumSeverity,
			Tags:     []string{"ssh", "compliance"},
		},
		Output: OutputBlock{Message: "Test"},
	}

	err := r.Register(plugin1)
	require.NoError(t, err)
	err = r.Register(plugin2)
	require.NoError(t, err)

	categories := r.Categories()
	require.Equal(t, 3, len(categories)) // ssh, security, compliance
	require.Equal(t, 2, categories["ssh"])
	require.Equal(t, 1, categories["security"])
	require.Equal(t, 1, categories["compliance"])
}

func TestRegistryV2_Clear(t *testing.T) {
	r := NewRegistryV2()

	plugin := &YAMLPlugin{
		Name:    "test-plugin",
		Version: "1.0.0",
		Type:    EvaluationType,
		Author:  "test",
		Metadata: PluginMetadata{
			Severity: HighSeverity,
		},
		Output: OutputBlock{Message: "Test"},
	}

	err := r.Register(plugin)
	require.NoError(t, err)
	require.Equal(t, 1, r.Count())

	r.Clear()
	require.Equal(t, 0, r.Count())
	require.Len(t, r.List(), 0)
	require.Len(t, r.Categories(), 0)
}

func TestRegistryV2_RegisterBulk(t *testing.T) {
	r := NewRegistryV2()

	plugins := []*YAMLPlugin{
		{
			Name:    "plugin1",
			Version: "1.0.0",
			Type:    EvaluationType,
			Author:  "test",
			Metadata: PluginMetadata{
				Severity: HighSeverity,
			},
			Output: OutputBlock{Message: "Test1"},
		},
		{
			Name:    "plugin2",
			Version: "1.0.0",
			Type:    EvaluationType,
			Author:  "test",
			Metadata: PluginMetadata{
				Severity: MediumSeverity,
			},
			Output: OutputBlock{Message: "Test2"},
		},
		{
			Name: "invalid-plugin",
			// Missing required fields
		},
	}

	count, errors := r.RegisterBulk(plugins)
	require.Equal(t, 2, count) // 2 valid plugins
	require.Len(t, errors, 1)  // 1 invalid plugin
	require.Contains(t, errors[0].Error(), "invalid-plugin")
}

func TestRegistryV2_ConcurrentAccess(t *testing.T) {
	r := NewRegistryV2()

	// Concurrent registration
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			plugin := &YAMLPlugin{
				Name:    fmt.Sprintf("plugin-%d", idx),
				Version: "1.0.0",
				Type:    EvaluationType,
				Author:  "test",
				Metadata: PluginMetadata{
					Severity: HighSeverity,
				},
				Output: OutputBlock{Message: "Test"},
			}

			err := r.Register(plugin)
			require.NoError(t, err)
		}(i)
	}

	wg.Wait()
	require.Equal(t, 10, r.Count())
}

func TestRegistryV2_Unregister_CleansUpCategories(t *testing.T) {
	r := NewRegistryV2()

	plugin := &YAMLPlugin{
		Name:    "test-plugin",
		Version: "1.0.0",
		Type:    EvaluationType,
		Author:  "test",
		Metadata: PluginMetadata{
			Severity: HighSeverity,
			Tags:     []string{"ssh", "unique-category"},
		},
		Output: OutputBlock{Message: "Test"},
	}

	err := r.Register(plugin)
	require.NoError(t, err)

	categories := r.Categories()
	require.Equal(t, 2, len(categories))

	// Unregister should remove empty categories
	err = r.Unregister("test-plugin")
	require.NoError(t, err)

	categories = r.Categories()
	require.Equal(t, 0, len(categories))
}

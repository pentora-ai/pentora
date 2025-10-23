// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoader_Load(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid YAML plugin
	validYAML := `name: Test Plugin
version: 1.0.0
type: evaluation
author: pentora-test

metadata:
  severity: high
  tags: [test, ssh]
  cve: CVE-2024-TEST

triggers:
  - data_key: ssh.version
    condition: exists
    value: true

match:
  logic: AND
  rules:
    - field: ssh.version
      operator: version_lt
      value: "8.0"

output:
  vulnerability: true
  message: "Test vulnerability found"
  remediation: "Upgrade SSH"
`

	yamlPath := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(yamlPath, []byte(validYAML), 0o644)
	require.NoError(t, err)

	loader := NewLoader(tmpDir)

	// Test loading YAML
	plugin, err := loader.Load(yamlPath)
	require.NoError(t, err)
	require.NotNil(t, plugin)
	require.Equal(t, "Test Plugin", plugin.Name)
	require.Equal(t, "1.0.0", plugin.Version)
	require.Equal(t, EvaluationType, plugin.Type)
	require.Equal(t, "pentora-test", plugin.Author)
	require.Equal(t, HighSeverity, plugin.Metadata.Severity)
	require.Equal(t, "CVE-2024-TEST", plugin.Metadata.CVE)
	require.Len(t, plugin.Triggers, 1)
	require.NotNil(t, plugin.Match)
	require.Equal(t, "Test vulnerability found", plugin.Output.Message)

	// Test cache
	cached, ok := loader.GetCached(yamlPath)
	require.True(t, ok)
	require.Equal(t, plugin, cached)
}

func TestLoader_Load_JSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Create valid JSON plugin
	validJSON := `{
  "name": "Test JSON Plugin",
  "version": "1.0.0",
  "type": "evaluation",
  "author": "pentora-test",
  "metadata": {
    "severity": "medium",
    "tags": ["test", "http"]
  },
  "triggers": [
    {
      "data_key": "http.server",
      "condition": "exists",
      "value": true
    }
  ],
  "match": {
    "logic": "OR",
    "rules": [
      {
        "field": "http.server",
        "operator": "contains",
        "value": "Apache"
      }
    ]
  },
  "output": {
    "vulnerability": true,
    "message": "Apache server detected"
  }
}`

	jsonPath := filepath.Join(tmpDir, "test.json")
	err := os.WriteFile(jsonPath, []byte(validJSON), 0o644)
	require.NoError(t, err)

	loader := NewLoader(tmpDir)

	// Test loading JSON
	plugin, err := loader.Load(jsonPath)
	require.NoError(t, err)
	require.NotNil(t, plugin)
	require.Equal(t, "Test JSON Plugin", plugin.Name)
	require.Equal(t, "1.0.0", plugin.Version)
	require.Equal(t, MediumSeverity, plugin.Metadata.Severity)
}

func TestLoader_Load_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()

	invalidYAML := `name: Invalid
invalid yaml syntax here {{{
`

	yamlPath := filepath.Join(tmpDir, "invalid.yaml")
	err := os.WriteFile(yamlPath, []byte(invalidYAML), 0o644)
	require.NoError(t, err)

	loader := NewLoader(tmpDir)

	_, err = loader.Load(yamlPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to parse YAML plugin")
}

func TestLoader_Load_ValidationFailure(t *testing.T) {
	tmpDir := t.TempDir()

	// Missing required fields
	invalidYAML := `name: Invalid Plugin
# Missing version, type, author, etc.
`

	yamlPath := filepath.Join(tmpDir, "invalid.yaml")
	err := os.WriteFile(yamlPath, []byte(invalidYAML), 0o644)
	require.NoError(t, err)

	loader := NewLoader(tmpDir)

	_, err = loader.Load(yamlPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "plugin validation failed")
}

func TestLoader_Load_UnsupportedExtension(t *testing.T) {
	tmpDir := t.TempDir()

	txtPath := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(txtPath, []byte("test"), 0o644)
	require.NoError(t, err)

	loader := NewLoader(tmpDir)

	_, err = loader.Load(txtPath)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unsupported file extension")
}

func TestLoader_LoadAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple plugins
	plugin1 := `name: Plugin 1
version: 1.0.0
type: evaluation
author: test
metadata:
  severity: high
  tags: [test]
output:
  vulnerability: true
  message: "Test 1"
`

	plugin2 := `name: Plugin 2
version: 1.0.0
type: evaluation
author: test
metadata:
  severity: medium
  tags: [test]
output:
  vulnerability: true
  message: "Test 2"
`

	err := os.WriteFile(filepath.Join(tmpDir, "plugin1.yaml"), []byte(plugin1), 0o644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tmpDir, "plugin2.yml"), []byte(plugin2), 0o644)
	require.NoError(t, err)

	// Create a non-plugin file (should be ignored)
	err = os.WriteFile(filepath.Join(tmpDir, "readme.txt"), []byte("readme"), 0o644)
	require.NoError(t, err)

	loader := NewLoader(tmpDir)

	plugins, err := loader.LoadAll(tmpDir)
	require.NoError(t, err)
	require.Len(t, plugins, 2)

	// Verify plugins
	names := []string{plugins[0].Name, plugins[1].Name}
	require.Contains(t, names, "Plugin 1")
	require.Contains(t, names, "Plugin 2")
}

func TestLoader_LoadRecursive(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.MkdirAll(subDir, 0o755)
	require.NoError(t, err)

	// Create plugins in both directories
	plugin1 := `name: Root Plugin
version: 1.0.0
type: evaluation
author: test
metadata:
  severity: high
  tags: [test]
output:
  vulnerability: true
  message: "Root plugin"
`

	plugin2 := `name: Sub Plugin
version: 1.0.0
type: evaluation
author: test
metadata:
  severity: medium
  tags: [test]
output:
  vulnerability: true
  message: "Sub plugin"
`

	err = os.WriteFile(filepath.Join(tmpDir, "plugin1.yaml"), []byte(plugin1), 0o644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(subDir, "plugin2.yaml"), []byte(plugin2), 0o644)
	require.NoError(t, err)

	loader := NewLoader(tmpDir)

	plugins, err := loader.LoadRecursive(tmpDir)
	require.NoError(t, err)
	require.Len(t, plugins, 2)

	// Verify plugins
	names := []string{plugins[0].Name, plugins[1].Name}
	require.Contains(t, names, "Root Plugin")
	require.Contains(t, names, "Sub Plugin")
}

func TestLoader_ClearCache(t *testing.T) {
	tmpDir := t.TempDir()

	validYAML := `name: Test
version: 1.0.0
type: evaluation
author: test
metadata:
  severity: high
  tags: [test]
output:
  vulnerability: true
  message: "Test"
`

	yamlPath := filepath.Join(tmpDir, "test.yaml")
	err := os.WriteFile(yamlPath, []byte(validYAML), 0o644)
	require.NoError(t, err)

	loader := NewLoader(tmpDir)

	// Load plugin
	_, err = loader.Load(yamlPath)
	require.NoError(t, err)

	// Verify cached
	_, ok := loader.GetCached(yamlPath)
	require.True(t, ok)

	// Clear cache
	loader.ClearCache()

	// Verify cache cleared
	_, ok = loader.GetCached(yamlPath)
	require.False(t, ok)
}

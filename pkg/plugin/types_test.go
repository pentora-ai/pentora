// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package plugin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlugin_Validate(t *testing.T) {
	tests := []struct {
		name    string
		plugin  *YAMLPlugin
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid plugin",
			plugin: &YAMLPlugin{
				ID:      "test",
				Name:    "Test Plugin",
				Version: "1.0.0",
				Type:    EvaluationType,
				Author:  "test-author",
				Metadata: PluginMetadata{
					Severity: HighSeverity,
					Tags:     []string{"test"},
				},
				Output: OutputBlock{
					Message: "Test message",
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			plugin: &YAMLPlugin{
				ID:      "test",
				Version: "1.0.0",
				Type:    EvaluationType,
				Author:  "test",
				Metadata: PluginMetadata{
					Severity: HighSeverity,
				},
				Output: OutputBlock{
					Message: "Test",
				},
			},
			wantErr: true,
			errMsg:  "plugin name is required",
		},
		{
			name: "missing version",
			plugin: &YAMLPlugin{
				ID:     "test",
				Name:   "Test",
				Type:   EvaluationType,
				Author: "test",
				Metadata: PluginMetadata{
					Severity: HighSeverity,
				},
				Output: OutputBlock{
					Message: "Test",
				},
			},
			wantErr: true,
			errMsg:  "plugin version is required",
		},
		{
			name: "missing type",
			plugin: &YAMLPlugin{
				ID:      "test",
				Name:    "Test",
				Version: "1.0.0",
				Author:  "test",
				Metadata: PluginMetadata{
					Severity: HighSeverity,
				},
				Output: OutputBlock{
					Message: "Test",
				},
			},
			wantErr: true,
			errMsg:  "plugin type is required",
		},
		{
			name: "missing author",
			plugin: &YAMLPlugin{
				ID:      "test",
				Name:    "Test",
				Version: "1.0.0",
				Type:    EvaluationType,
				Metadata: PluginMetadata{
					Severity: HighSeverity,
				},
				Output: OutputBlock{
					Message: "Test",
				},
			},
			wantErr: true,
			errMsg:  "plugin author is required",
		},
		{
			name: "missing severity",
			plugin: &YAMLPlugin{
				ID:       "test",
				Name:     "Test",
				Version:  "1.0.0",
				Type:     EvaluationType,
				Author:   "test",
				Metadata: PluginMetadata{},
				Output: OutputBlock{
					Message: "Test",
				},
			},
			wantErr: true,
			errMsg:  "plugin severity is required",
		},
		{
			name: "invalid severity",
			plugin: &YAMLPlugin{
				ID:      "test",
				Name:    "Test",
				Version: "1.0.0",
				Type:    EvaluationType,
				Author:  "test",
				Metadata: PluginMetadata{
					Severity: "super-critical",
				},
				Output: OutputBlock{
					Message: "Test",
				},
			},
			wantErr: true,
			errMsg:  "invalid severity",
		},
		{
			name: "missing output message",
			plugin: &YAMLPlugin{
				ID:      "test",
				Name:    "Test",
				Version: "1.0.0",
				Type:    EvaluationType,
				Author:  "test",
				Metadata: PluginMetadata{
					Severity: HighSeverity,
				},
				Output: OutputBlock{},
			},
			wantErr: true,
			errMsg:  "output message is required",
		},
		{
			name: "trigger missing data_key",
			plugin: &YAMLPlugin{
				ID:      "test",
				Name:    "Test",
				Version: "1.0.0",
				Type:    EvaluationType,
				Author:  "test",
				Metadata: PluginMetadata{
					Severity: HighSeverity,
				},
				Triggers: []Trigger{
					{Condition: "exists", Value: true},
				},
				Output: OutputBlock{
					Message: "Test",
				},
			},
			wantErr: true,
			errMsg:  "data_key is required",
		},
		{
			name: "trigger missing condition",
			plugin: &YAMLPlugin{
				ID:      "test",
				Name:    "Test",
				Version: "1.0.0",
				Type:    EvaluationType,
				Author:  "test",
				Metadata: PluginMetadata{
					Severity: HighSeverity,
				},
				Triggers: []Trigger{
					{DataKey: "test", Value: true},
				},
				Output: OutputBlock{
					Message: "Test",
				},
			},
			wantErr: true,
			errMsg:  "condition is required",
		},
		{
			name: "invalid match block",
			plugin: &YAMLPlugin{
				ID:      "test",
				Name:    "Test",
				Version: "1.0.0",
				Type:    EvaluationType,
				Author:  "test",
				Metadata: PluginMetadata{
					Severity: HighSeverity,
				},
				Match: &MatchBlock{
					Logic: "INVALID",
					Rules: []MatchRule{
						{Field: "test", Operator: "equals", Value: "test"},
					},
				},
				Output: OutputBlock{
					Message: "Test",
				},
			},
			wantErr: true,
			errMsg:  "match block validation failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.plugin.Validate()

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMatchBlock_Validate(t *testing.T) {
	tests := []struct {
		name    string
		match   *MatchBlock
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid match block",
			match: &MatchBlock{
				Logic: "AND",
				Rules: []MatchRule{
					{Field: "test", Operator: "equals", Value: "test"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing logic",
			match: &MatchBlock{
				Rules: []MatchRule{
					{Field: "test", Operator: "equals", Value: "test"},
				},
			},
			wantErr: true,
			errMsg:  "match logic is required",
		},
		{
			name: "invalid logic",
			match: &MatchBlock{
				Logic: "XOR",
				Rules: []MatchRule{
					{Field: "test", Operator: "equals", Value: "test"},
				},
			},
			wantErr: true,
			errMsg:  "invalid match logic",
		},
		{
			name: "empty rules",
			match: &MatchBlock{
				Logic: "AND",
				Rules: []MatchRule{},
			},
			wantErr: true,
			errMsg:  "match rules cannot be empty",
		},
		{
			name: "rule missing field",
			match: &MatchBlock{
				Logic: "AND",
				Rules: []MatchRule{
					{Operator: "equals", Value: "test"},
				},
			},
			wantErr: true,
			errMsg:  "field is required",
		},
		{
			name: "rule missing operator",
			match: &MatchBlock{
				Logic: "AND",
				Rules: []MatchRule{
					{Field: "test", Value: "test"},
				},
			},
			wantErr: true,
			errMsg:  "operator is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.match.Validate()

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSeverityValues(t *testing.T) {
	// Test that all severity constants are defined
	require.Equal(t, Severity("critical"), CriticalSeverity)
	require.Equal(t, Severity("high"), HighSeverity)
	require.Equal(t, Severity("medium"), MediumSeverity)
	require.Equal(t, Severity("low"), LowSeverity)
	require.Equal(t, Severity("info"), InfoSeverity)
}

func TestPluginTypeValues(t *testing.T) {
	// Test that all plugin type constants are defined
	require.Equal(t, PluginType("evaluation"), EvaluationType)
	require.Equal(t, PluginType("output"), OutputType)
	require.Equal(t, PluginType("integration"), IntegrationType)
}

func TestPlugin_IsCompatibleWithPentora(t *testing.T) {
	tests := []struct {
		name             string
		minVersion       string
		pentoraVersion   string
		expectCompatible bool
		expectError      bool
	}{
		{
			name:             "no version constraint",
			minVersion:       "",
			pentoraVersion:   "0.1.0",
			expectCompatible: true,
			expectError:      false,
		},
		{
			name:             "compatible version (greater)",
			minVersion:       "0.1.0",
			pentoraVersion:   "0.2.0",
			expectCompatible: true,
			expectError:      false,
		},
		{
			name:             "compatible version (equal)",
			minVersion:       "0.1.0",
			pentoraVersion:   "0.1.0",
			expectCompatible: true,
			expectError:      false,
		},
		{
			name:             "incompatible version (lower)",
			minVersion:       "0.2.0",
			pentoraVersion:   "0.1.0",
			expectCompatible: false,
			expectError:      true,
		},
		{
			name:             "version with v prefix",
			minVersion:       "v0.1.0",
			pentoraVersion:   "v0.2.0",
			expectCompatible: true,
			expectError:      false,
		},
		{
			name:             "mixed v prefix (min has v, pentora doesn't)",
			minVersion:       "v0.1.0",
			pentoraVersion:   "0.2.0",
			expectCompatible: true,
			expectError:      false,
		},
		{
			name:             "mixed v prefix (pentora has v, min doesn't)",
			minVersion:       "0.1.0",
			pentoraVersion:   "v0.2.0",
			expectCompatible: true,
			expectError:      false,
		},
		{
			name:             "patch version matters",
			minVersion:       "0.1.5",
			pentoraVersion:   "0.1.4",
			expectCompatible: false,
			expectError:      true,
		},
		{
			name:             "patch version compatible",
			minVersion:       "0.1.5",
			pentoraVersion:   "0.1.6",
			expectCompatible: true,
			expectError:      false,
		},
		{
			name:             "dev version (treated as vdev)",
			minVersion:       "0.1.0",
			pentoraVersion:   "dev",
			expectCompatible: false, // "vdev" < "v0.1.0" lexicographically
			expectError:      true,
		},
		{
			name:             "invalid pentora version",
			minVersion:       "0.1.0",
			pentoraVersion:   "invalid",
			expectCompatible: false,
			expectError:      true,
		},
		{
			name:             "invalid min version",
			minVersion:       "not-a-version",
			pentoraVersion:   "0.1.0",
			expectCompatible: false,
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := &YAMLPlugin{
				ID:                "test-plugin",
				Name:              "Test Plugin",
				Version:           "1.0.0",
				Type:              EvaluationType,
				Author:            "test",
				MinPentoraVersion: tt.minVersion,
				Metadata: PluginMetadata{
					Severity: HighSeverity,
					Tags:     []string{"test"},
				},
				Output: OutputBlock{
					Message: "Test",
				},
			}

			compatible, err := plugin.IsCompatibleWithPentora(tt.pentoraVersion)

			if tt.expectError {
				require.Error(t, err)
				require.False(t, compatible)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expectCompatible, compatible)
			}
		})
	}
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"with v prefix", "v0.1.0", "v0.1.0"},
		{"without v prefix", "0.1.0", "v0.1.0"},
		{"dev version", "dev", "vdev"},
		{"empty string", "", ""},
		{"v prefix with patch", "v1.2.3", "v1.2.3"},
		{"no prefix with patch", "1.2.3", "v1.2.3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizeVersion(tt.input)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidSemver(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		{"valid semver", "0.1.0", true},
		{"valid semver with v", "v0.1.0", true},
		{"valid semver with patch", "1.2.3", true},
		{"short version (Go allows)", "0.1", true}, // Go's semver allows "v0.1"
		{"invalid semver (text)", "invalid", false},
		{"invalid semver (empty)", "", false},
		{"valid semver with prerelease", "v1.0.0-alpha", true},
		{"valid semver with build metadata", "v1.0.0+build.123", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidSemver(tt.version)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestPlugin_Validate_WithMinPentoraVersion(t *testing.T) {
	tests := []struct {
		name        string
		minVersion  string
		expectErr   bool
		errContains string
	}{
		{
			name:       "valid min version",
			minVersion: "0.1.0",
			expectErr:  false,
		},
		{
			name:       "valid min version with v prefix",
			minVersion: "v0.1.0",
			expectErr:  false,
		},
		{
			name:       "no min version",
			minVersion: "",
			expectErr:  false,
		},
		{
			name:        "invalid min version",
			minVersion:  "not-a-version",
			expectErr:   true,
			errContains: "invalid pentora_min_version format",
		},
		{
			name:       "short version (Go allows)",
			minVersion: "0.1",
			expectErr:  false, // Go's semver allows "v0.1"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugin := &YAMLPlugin{
				ID:                "test-plugin",
				Name:              "Test Plugin",
				Version:           "1.0.0",
				Type:              EvaluationType,
				Author:            "test",
				MinPentoraVersion: tt.minVersion,
				Metadata: PluginMetadata{
					Severity: HighSeverity,
					Tags:     []string{"test"},
				},
				Output: OutputBlock{
					Message: "Test",
				},
			}

			err := plugin.Validate()

			if tt.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

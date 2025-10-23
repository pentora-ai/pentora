// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package plugin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTriggerEvaluator_ShouldTrigger(t *testing.T) {
	te := NewTriggerEvaluator()

	tests := []struct {
		name     string
		triggers []Trigger
		context  map[string]any
		want     bool
		wantErr  bool
	}{
		{
			name:     "no triggers - always trigger",
			triggers: []Trigger{},
			context:  map[string]any{},
			want:     true,
		},
		{
			name: "exists - key exists",
			triggers: []Trigger{
				{DataKey: "ssh.version", Condition: "exists", Value: true},
			},
			context: map[string]any{
				"ssh.version": "7.4.0",
			},
			want: true,
		},
		{
			name: "exists - key does not exist",
			triggers: []Trigger{
				{DataKey: "ssh.version", Condition: "exists", Value: true},
			},
			context: map[string]any{},
			want:    false,
		},
		{
			name: "equals - match",
			triggers: []Trigger{
				{DataKey: "service", Condition: "equals", Value: "ssh"},
			},
			context: map[string]any{
				"service": "ssh",
			},
			want: true,
		},
		{
			name: "equals - no match",
			triggers: []Trigger{
				{DataKey: "service", Condition: "equals", Value: "ssh"},
			},
			context: map[string]any{
				"service": "http",
			},
			want: false,
		},
		{
			name: "contains - match",
			triggers: []Trigger{
				{DataKey: "banner", Condition: "contains", Value: "OpenSSH"},
			},
			context: map[string]any{
				"banner": "OpenSSH_7.4p1 Ubuntu",
			},
			want: true,
		},
		{
			name: "version_lt - true",
			triggers: []Trigger{
				{DataKey: "ssh.version", Condition: "version_lt", Value: "8.0"},
			},
			context: map[string]any{
				"ssh.version": "7.4.0",
			},
			want: true,
		},
		{
			name: "version_lt - false",
			triggers: []Trigger{
				{DataKey: "ssh.version", Condition: "version_lt", Value: "7.0"},
			},
			context: map[string]any{
				"ssh.version": "7.4.0",
			},
			want: false,
		},
		{
			name: "multiple triggers - all satisfied",
			triggers: []Trigger{
				{DataKey: "service", Condition: "equals", Value: "ssh"},
				{DataKey: "ssh.version", Condition: "version_lt", Value: "8.0"},
			},
			context: map[string]any{
				"service":     "ssh",
				"ssh.version": "7.4.0",
			},
			want: true,
		},
		{
			name: "multiple triggers - one not satisfied",
			triggers: []Trigger{
				{DataKey: "service", Condition: "equals", Value: "ssh"},
				{DataKey: "ssh.version", Condition: "version_lt", Value: "7.0"},
			},
			context: map[string]any{
				"service":     "ssh",
				"ssh.version": "7.4.0",
			},
			want: false,
		},
		{
			name: "in - match",
			triggers: []Trigger{
				{DataKey: "service", Condition: "in", Value: []any{"ssh", "telnet", "ftp"}},
			},
			context: map[string]any{
				"service": "ssh",
			},
			want: true,
		},
		{
			name: "notIn - match",
			triggers: []Trigger{
				{DataKey: "service", Condition: "notIn", Value: []any{"http", "https"}},
			},
			context: map[string]any{
				"service": "ssh",
			},
			want: true,
		},
		{
			name: "gt - true",
			triggers: []Trigger{
				{DataKey: "port", Condition: "gt", Value: 1000},
			},
			context: map[string]any{
				"port": 8080,
			},
			want: true,
		},
		{
			name: "unknown condition",
			triggers: []Trigger{
				{DataKey: "service", Condition: "unknown_condition", Value: "ssh"},
			},
			context: map[string]any{
				"service": "ssh",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := te.ShouldTrigger(tt.triggers, tt.context)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

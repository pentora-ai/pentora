// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package plugin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllCategories(t *testing.T) {
	categories := AllCategories()
	require.Len(t, categories, 8)
	require.Contains(t, categories, CategorySSH)
	require.Contains(t, categories, CategoryHTTP)
	require.Contains(t, categories, CategoryWeb)
	require.Contains(t, categories, CategoryTLS)
	require.Contains(t, categories, CategoryDatabase)
	require.Contains(t, categories, CategoryIoT)
	require.Contains(t, categories, CategoryNetwork)
	require.Contains(t, categories, CategoryMisc)
}

func TestCategory_String(t *testing.T) {
	require.Equal(t, "ssh", CategorySSH.String())
	require.Equal(t, "http", CategoryHTTP.String())
	require.Equal(t, "database", CategoryDatabase.String())
}

func TestCategory_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		category Category
		want     bool
	}{
		{"valid ssh", CategorySSH, true},
		{"valid http", CategoryHTTP, true},
		{"valid database", CategoryDatabase, true},
		{"invalid custom", Category("custom"), false},
		{"invalid empty", Category(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, tt.category.IsValid())
		})
	}
}

func TestCategoryFromString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Category
	}{
		{"ssh", "ssh", CategorySSH},
		{"http", "http", CategoryHTTP},
		{"database", "database", CategoryDatabase},
		{"invalid falls back to misc", "invalid", CategoryMisc},
		{"empty falls back to misc", "", CategoryMisc},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CategoryFromString(tt.input)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestPortToCategories(t *testing.T) {
	tests := []struct {
		name string
		port int
		want []Category
	}{
		{"SSH port 22", 22, []Category{CategorySSH}},
		{"HTTP port 80", 80, []Category{CategoryHTTP, CategoryWeb}},
		{"HTTPS port 443", 443, []Category{CategoryHTTP, CategoryWeb, CategoryTLS}},
		{"MySQL port 3306", 3306, []Category{CategoryDatabase}},
		{"PostgreSQL port 5432", 5432, []Category{CategoryDatabase}},
		{"MongoDB port 27017", 27017, []Category{CategoryDatabase}},
		{"Redis port 6379", 6379, []Category{CategoryDatabase}},
		{"MQTT port 1883", 1883, []Category{CategoryIoT}},
		{"Unknown port", 9999, []Category{CategoryMisc}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PortToCategories(tt.port)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestServiceToCategories(t *testing.T) {
	tests := []struct {
		name    string
		service string
		want    []Category
	}{
		{"ssh", "ssh", []Category{CategorySSH}},
		{"openssh", "openssh", []Category{CategorySSH}},
		{"http", "http", []Category{CategoryHTTP, CategoryWeb}},
		{"https", "https", []Category{CategoryHTTP, CategoryWeb}},
		{"tls", "tls", []Category{CategoryTLS}},
		{"mysql", "mysql", []Category{CategoryDatabase}},
		{"postgresql", "postgresql", []Category{CategoryDatabase}},
		{"mqtt", "mqtt", []Category{CategoryIoT}},
		{"ftp", "ftp", []Category{CategoryNetwork}},
		{"unknown", "unknown", []Category{CategoryMisc}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ServiceToCategories(tt.service)
			require.Equal(t, tt.want, got)
		})
	}
}

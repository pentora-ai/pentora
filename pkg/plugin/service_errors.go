// Copyright 2025 Pentora Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");

package plugin

import "errors"

// Service layer errors
// These are domain-specific errors that can be checked using errors.Is()

var (
	// ErrPluginNotFound is returned when a requested plugin cannot be found
	ErrPluginNotFound = errors.New("plugin not found")

	// ErrPluginAlreadyInstalled is returned when trying to install an already cached plugin without force
	ErrPluginAlreadyInstalled = errors.New("plugin already installed")

	// ErrPluginNotInstalled is returned when trying to operate on a plugin that isn't installed
	ErrPluginNotInstalled = errors.New("plugin not installed")

	// ErrInvalidCategory is returned when an invalid category is specified
	ErrInvalidCategory = errors.New("invalid category")

	// ErrInvalidPluginID is returned when a plugin ID is malformed
	ErrInvalidPluginID = errors.New("invalid plugin ID")

	// ErrNoPluginsFound is returned when no plugins match the criteria
	ErrNoPluginsFound = errors.New("no plugins found")

	// ErrSourceNotAvailable is returned when a plugin source cannot be reached
	ErrSourceNotAvailable = errors.New("plugin source not available")

	// ErrChecksumMismatch is returned when downloaded plugin checksum doesn't match
	ErrChecksumMismatch = errors.New("checksum mismatch")

	// ErrInvalidInput is returned when input validation fails
	ErrInvalidInput = errors.New("invalid input")
)

// IsNotFound checks if error is a "not found" error
func IsNotFound(err error) bool {
	return errors.Is(err, ErrPluginNotFound) || errors.Is(err, ErrPluginNotInstalled)
}

// IsAlreadyExists checks if error is an "already exists" error
func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrPluginAlreadyInstalled)
}

// IsInvalidInput checks if error is an "invalid input" error
func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidCategory) || errors.Is(err, ErrInvalidPluginID)
}

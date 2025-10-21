package storage

import (
	"errors"
	"fmt"
)

// Common errors returned by storage operations.
var (
	// ErrNotFound is returned when a requested resource does not exist.
	ErrNotFound = errors.New("not found")

	// ErrAlreadyExists is returned when attempting to create a resource that already exists.
	ErrAlreadyExists = errors.New("already exists")

	// ErrNotSupported is returned when an operation is not supported by the backend.
	// For example, OSS edition returns this for Enterprise-only features.
	ErrNotSupported = errors.New("operation not supported in this edition")

	// ErrInvalidInput is returned when input validation fails.
	ErrInvalidInput = errors.New("invalid input")

	// ErrClosed is returned when attempting to use a closed backend.
	ErrClosed = errors.New("backend is closed")
)

// NotFoundError wraps ErrNotFound with additional context.
type NotFoundError struct {
	ResourceType string // "scan", "org", "user", etc.
	ResourceID   string
}

// Error implements the error interface.
func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.ResourceType, e.ResourceID)
}

// Unwrap returns the underlying error.
func (e *NotFoundError) Unwrap() error {
	return ErrNotFound
}

// Is checks if the error matches ErrNotFound.
func (e *NotFoundError) Is(target error) bool {
	return target == ErrNotFound
}

// AlreadyExistsError wraps ErrAlreadyExists with additional context.
type AlreadyExistsError struct {
	ResourceType string
	ResourceID   string
}

// Error implements the error interface.
func (e *AlreadyExistsError) Error() string {
	return fmt.Sprintf("%s already exists: %s", e.ResourceType, e.ResourceID)
}

// Unwrap returns the underlying error.
func (e *AlreadyExistsError) Unwrap() error {
	return ErrAlreadyExists
}

// Is checks if the error matches ErrAlreadyExists.
func (e *AlreadyExistsError) Is(target error) bool {
	return target == ErrAlreadyExists
}

// InvalidInputError wraps ErrInvalidInput with details.
type InvalidInputError struct {
	Field  string // Field name that failed validation
	Reason string // Why validation failed
}

// Error implements the error interface.
func (e *InvalidInputError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("invalid input for field %q: %s", e.Field, e.Reason)
	}
	return fmt.Sprintf("invalid input: %s", e.Reason)
}

// Unwrap returns the underlying error.
func (e *InvalidInputError) Unwrap() error {
	return ErrInvalidInput
}

// Is checks if the error matches ErrInvalidInput.
func (e *InvalidInputError) Is(target error) bool {
	return target == ErrInvalidInput
}

// Helper functions for creating errors.

// NewNotFoundError creates a NotFoundError.
func NewNotFoundError(resourceType, resourceID string) error {
	return &NotFoundError{
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}
}

// NewAlreadyExistsError creates an AlreadyExistsError.
func NewAlreadyExistsError(resourceType, resourceID string) error {
	return &AlreadyExistsError{
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}
}

// NewInvalidInputError creates an InvalidInputError.
func NewInvalidInputError(field, reason string) error {
	return &InvalidInputError{
		Field:  field,
		Reason: reason,
	}
}

// IsNotFound checks if an error is or wraps ErrNotFound.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsAlreadyExists checks if an error is or wraps ErrAlreadyExists.
func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrAlreadyExists)
}

// IsNotSupported checks if an error is or wraps ErrNotSupported.
func IsNotSupported(err error) bool {
	return errors.Is(err, ErrNotSupported)
}

// IsInvalidInput checks if an error is or wraps ErrInvalidInput.
func IsInvalidInput(err error) bool {
	return errors.Is(err, ErrInvalidInput)
}

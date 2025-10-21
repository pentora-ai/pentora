package storage

import (
	"errors"
	"testing"
)

func TestNotFoundError(t *testing.T) {
	err := NewNotFoundError("scan", "scan-123")

	// Check error message
	expected := "scan not found: scan-123"
	if err.Error() != expected {
		t.Errorf("Error() = %q, expected %q", err.Error(), expected)
	}

	// Check errors.Is
	if !errors.Is(err, ErrNotFound) {
		t.Error("errors.Is(err, ErrNotFound) = false, expected true")
	}

	// Check IsNotFound helper
	if !IsNotFound(err) {
		t.Error("IsNotFound(err) = false, expected true")
	}

	// Check errors.As
	var notFoundErr *NotFoundError
	if !errors.As(err, &notFoundErr) {
		t.Error("errors.As failed to extract NotFoundError")
	}
	if notFoundErr.ResourceType != "scan" {
		t.Errorf("ResourceType = %q, expected %q", notFoundErr.ResourceType, "scan")
	}
	if notFoundErr.ResourceID != "scan-123" {
		t.Errorf("ResourceID = %q, expected %q", notFoundErr.ResourceID, "scan-123")
	}
}

func TestAlreadyExistsError(t *testing.T) {
	err := NewAlreadyExistsError("scan", "scan-456")

	// Check error message
	expected := "scan already exists: scan-456"
	if err.Error() != expected {
		t.Errorf("Error() = %q, expected %q", err.Error(), expected)
	}

	// Check errors.Is
	if !errors.Is(err, ErrAlreadyExists) {
		t.Error("errors.Is(err, ErrAlreadyExists) = false, expected true")
	}

	// Check IsAlreadyExists helper
	if !IsAlreadyExists(err) {
		t.Error("IsAlreadyExists(err) = false, expected true")
	}

	// Check errors.As
	var alreadyExistsErr *AlreadyExistsError
	if !errors.As(err, &alreadyExistsErr) {
		t.Error("errors.As failed to extract AlreadyExistsError")
	}
}

func TestInvalidInputError(t *testing.T) {
	t.Run("with field", func(t *testing.T) {
		err := NewInvalidInputError("target", "must not be empty")

		expected := `invalid input for field "target": must not be empty`
		if err.Error() != expected {
			t.Errorf("Error() = %q, expected %q", err.Error(), expected)
		}

		if !errors.Is(err, ErrInvalidInput) {
			t.Error("errors.Is(err, ErrInvalidInput) = false, expected true")
		}

		if !IsInvalidInput(err) {
			t.Error("IsInvalidInput(err) = false, expected true")
		}
	})

	t.Run("without field", func(t *testing.T) {
		err := &InvalidInputError{
			Reason: "invalid data format",
		}

		expected := "invalid input: invalid data format"
		if err.Error() != expected {
			t.Errorf("Error() = %q, expected %q", err.Error(), expected)
		}
	})
}

func TestIsNotSupported(t *testing.T) {
	// Direct error
	if !IsNotSupported(ErrNotSupported) {
		t.Error("IsNotSupported(ErrNotSupported) = false, expected true")
	}

	// Wrapped error
	wrapped := errors.Join(ErrNotSupported, errors.New("additional context"))
	if !IsNotSupported(wrapped) {
		t.Error("IsNotSupported(wrapped) = false, expected true")
	}

	// Different error
	if IsNotSupported(ErrNotFound) {
		t.Error("IsNotSupported(ErrNotFound) = true, expected false")
	}
}

func TestErrorHelpers(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		checker  func(error) bool
		expected bool
	}{
		{
			name:     "IsNotFound - true",
			err:      NewNotFoundError("scan", "123"),
			checker:  IsNotFound,
			expected: true,
		},
		{
			name:     "IsNotFound - false",
			err:      ErrAlreadyExists,
			checker:  IsNotFound,
			expected: false,
		},
		{
			name:     "IsAlreadyExists - true",
			err:      NewAlreadyExistsError("scan", "456"),
			checker:  IsAlreadyExists,
			expected: true,
		},
		{
			name:     "IsAlreadyExists - false",
			err:      ErrNotFound,
			checker:  IsAlreadyExists,
			expected: false,
		},
		{
			name:     "IsInvalidInput - true",
			err:      NewInvalidInputError("field", "reason"),
			checker:  IsInvalidInput,
			expected: true,
		},
		{
			name:     "IsInvalidInput - false",
			err:      ErrNotFound,
			checker:  IsInvalidInput,
			expected: false,
		},
		{
			name:     "IsNotSupported - true",
			err:      ErrNotSupported,
			checker:  IsNotSupported,
			expected: true,
		},
		{
			name:     "IsNotSupported - false",
			err:      ErrNotFound,
			checker:  IsNotSupported,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.checker(tt.err)
			if got != tt.expected {
				t.Errorf("checker(%v) = %v, expected %v", tt.err, got, tt.expected)
			}
		})
	}
}

func TestNotFoundErrorUnwrap(t *testing.T) {
	t.Run("returns ErrNotFound for concrete error", func(t *testing.T) {
		err := NewNotFoundError("scan", "scan-123")
		unwrapped := errors.Unwrap(err)
		if unwrapped != ErrNotFound {
			t.Fatalf("errors.Unwrap(err) = %v, expected %v", unwrapped, ErrNotFound)
		}
	})

	t.Run("works with nil receiver", func(t *testing.T) {
		var nf *NotFoundError
		unwrapped := errors.Unwrap(nf)
		if unwrapped != ErrNotFound {
			t.Fatalf("errors.Unwrap(nil *NotFoundError) = %v, expected %v", unwrapped, ErrNotFound)
		}
	})
}

func TestInvalidInputErrorUnwrap(t *testing.T) {
	t.Run("returns ErrInvalidInput for concrete error", func(t *testing.T) {
		err := NewInvalidInputError("field", "reason")
		unwrapped := errors.Unwrap(err)
		if unwrapped != ErrInvalidInput {
			t.Fatalf("errors.Unwrap(err) = %v, expected %v", unwrapped, ErrInvalidInput)
		}
	})

	t.Run("works with nil receiver", func(t *testing.T) {
		var ii *InvalidInputError
		unwrapped := errors.Unwrap(ii)
		if unwrapped != ErrInvalidInput {
			t.Fatalf("errors.Unwrap(nil *InvalidInputError) = %v, expected %v", unwrapped, ErrInvalidInput)
		}
	})
}

func TestAlreadyExistsErrorUnwrap(t *testing.T) {
	t.Run("returns ErrAlreadyExists for concrete error", func(t *testing.T) {
		err := NewAlreadyExistsError("scan", "scan-456")
		unwrapped := errors.Unwrap(err)
		if unwrapped != ErrAlreadyExists {
			t.Fatalf("errors.Unwrap(err) = %v, expected %v", unwrapped, ErrAlreadyExists)
		}
	})

	t.Run("works with nil receiver", func(t *testing.T) {
		var ae *AlreadyExistsError
		unwrapped := errors.Unwrap(ae)
		if unwrapped != ErrAlreadyExists {
			t.Fatalf("errors.Unwrap(nil *AlreadyExistsError) = %v, expected %v", unwrapped, ErrAlreadyExists)
		}
	})
}

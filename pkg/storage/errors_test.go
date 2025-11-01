package storage

import (
	"errors"
	"fmt"
	"os"
	"testing"
)

func TestNotFoundError(t *testing.T) {
	err := NewNotFoundError("scan", "123")
	if err.Error() != "scan not found: 123" {
		t.Errorf("unexpected error message: %v", err)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected Is(err, ErrNotFound) to be true")
	}
	if !IsNotFound(err) {
		t.Errorf("expected IsNotFound to be true")
	}
}

func TestAlreadyExistsError(t *testing.T) {
	err := NewAlreadyExistsError("user", "bob")
	if err.Error() != "user already exists: bob" {
		t.Errorf("unexpected error message: %v", err)
	}
	if !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("expected Is(err, ErrAlreadyExists)")
	}
	if !IsAlreadyExists(err) {
		t.Errorf("expected IsAlreadyExists true")
	}
}

func TestInvalidInputError(t *testing.T) {
	err1 := NewInvalidInputError("field", "bad value")
	if err1.Error() != `invalid input for field "field": bad value` {
		t.Errorf("unexpected message: %v", err1)
	}
	err2 := NewInvalidInputError("", "something wrong")
	if err2.Error() != "invalid input: something wrong" {
		t.Errorf("unexpected message: %v", err2)
	}
	if !errors.Is(err1, ErrInvalidInput) {
		t.Errorf("expected Is(err1, ErrInvalidInput)")
	}
	if !IsInvalidInput(err1) {
		t.Errorf("expected IsInvalidInput true")
	}
}

func TestWithErrorCodeAndErrorCode(t *testing.T) {
	// Nil error should return nil
	if WithErrorCode(nil, "X") != nil {
		t.Errorf("expected nil")
	}

	base := fmt.Errorf("base")
	wrapped := WithErrorCode(base, "CODE123")
	if ErrorCode(wrapped) != "CODE123" {
		t.Errorf("expected CODE123, got %v", ErrorCode(wrapped))
	}
	if wrapped.(*storageCodeError).Code() != "CODE123" {
		t.Errorf("wrong code from Code()")
	}
	if wrapped.Error() != "base" {
		t.Errorf("unexpected message")
	}
	if !errors.Is(wrapped, base) {
		t.Errorf("unwrap mismatch")
	}
}

func TestErrorCodeBranches(t *testing.T) {
	if code := ErrorCode(ErrRetentionPolicyNotConfigured); code != errorCodeNoRetention {
		t.Errorf("expected %s, got %s", errorCodeNoRetention, code)
	}

	err := NewInvalidInputError("workspace_root", "invalid")
	if c := ErrorCode(err); c != errorCodeWorkspaceInvalid {
		t.Errorf("expected workspace_invalid, got %s", c)
	}

	err2 := NewInvalidInputError("other", "bad")
	if c := ErrorCode(err2); c != errorCodeStorageInvalidInput {
		t.Errorf("expected storage_invalid_input, got %s", c)
	}

	if c := ErrorCode(ErrNotSupported); c != errorCodeStorageFailure {
		t.Errorf("expected storage_failure, got %s", c)
	}

	if c := ErrorCode(ErrClosed); c != errorCodeStorageFailure {
		t.Errorf("expected storage_failure, got %s", c)
	}

	pathErr := &os.PathError{Err: os.ErrPermission}
	if c := ErrorCode(pathErr); c != errorCodeWorkspacePerm {
		t.Errorf("expected workspace_perm, got %s", c)
	}

	pathErr2 := &os.PathError{Err: errors.New("other")}
	if c := ErrorCode(pathErr2); c != errorCodeWorkspaceInvalid {
		t.Errorf("expected workspace_invalid, got %s", c)
	}

	// fallback
	if c := ErrorCode(errors.New("random")); c != errorCodeStorageFailure {
		t.Errorf("expected storage_failure fallback, got %s", c)
	}

	if c := ErrorCode(nil); c != "" {
		t.Errorf("expected empty code for nil")
	}
}

func TestExitCode(t *testing.T) {
	tests := []struct {
		err      error
		expected int
	}{
		{nil, 0},
		{WithErrorCode(errors.New("x"), errorCodeNoRetention), 2},
		{WithErrorCode(errors.New("x"), errorCodeInvalidRetention), 2},
		{WithErrorCode(errors.New("x"), errorCodeStorageInvalidInput), 2},
		{WithErrorCode(errors.New("x"), errorCodeWorkspacePerm), 1},
		{WithErrorCode(errors.New("x"), errorCodeWorkspaceInvalid), 1},
		{WithErrorCode(errors.New("x"), errorCodeStorageFailure), 1},
		{errors.New("unknown"), 1},
	}
	for _, tt := range tests {
		if got := ExitCode(tt.err); got != tt.expected {
			t.Errorf("ExitCode(%v)=%d, want %d", tt.err, got, tt.expected)
		}
	}
}

func TestHTTPStatus(t *testing.T) {
	tests := []struct {
		err      error
		expected int
	}{
		{nil, 200},
		{WithErrorCode(errors.New("x"), errorCodeNoRetention), 400},
		{WithErrorCode(errors.New("x"), errorCodeInvalidRetention), 400},
		{WithErrorCode(errors.New("x"), errorCodeStorageInvalidInput), 400},
		{WithErrorCode(errors.New("x"), errorCodeWorkspacePerm), 403},
		{WithErrorCode(errors.New("x"), errorCodeWorkspaceInvalid), 404},
		{WithErrorCode(errors.New("x"), errorCodeStorageFailure), 500},
		{errors.New("unknown"), 500},
	}
	for _, tt := range tests {
		if got := HTTPStatus(tt.err); got != tt.expected {
			t.Errorf("HTTPStatus(%v)=%d, want %d", tt.err, got, tt.expected)
		}
	}
}

func TestSuggestions(t *testing.T) {
	tests := []struct {
		code string
		want bool
	}{
		{errorCodeNoRetention, true},
		{errorCodeInvalidRetention, true},
		{errorCodeWorkspaceInvalid, true},
		{errorCodeWorkspacePerm, true},
		{errorCodeStorageInvalidInput, true},
		{errorCodeStorageFailure, true},
		{"unknown", false},
	}
	for _, tt := range tests {
		err := WithErrorCode(errors.New("x"), tt.code)
		sugs := Suggestions(err)
		if tt.want && len(sugs) == 0 {
			t.Errorf("expected suggestions for %v", tt.code)
		}
		if !tt.want && sugs != nil {
			t.Errorf("expected nil for %v", tt.code)
		}
	}
	if Suggestions(nil) != nil {
		t.Errorf("expected nil for nil error")
	}
}

func TestFormatRetentionValidationError(t *testing.T) {
	err := errors.New("invalid policy")
	wrapped := FormatRetentionValidationError(err)
	if ErrorCode(wrapped) != errorCodeInvalidRetention {
		t.Errorf("expected invalid_retention code")
	}
	if FormatRetentionValidationError(nil) != nil {
		t.Errorf("expected nil")
	}
}

func TestUnwraps(t *testing.T) {
	nf := NewNotFoundError("scan", "42").(*NotFoundError)
	if nf.Unwrap() != ErrNotFound {
		t.Errorf("expected ErrNotFound from Unwrap")
	}

	ae := NewAlreadyExistsError("file", "abc").(*AlreadyExistsError)
	if ae.Unwrap() != ErrAlreadyExists {
		t.Errorf("expected ErrAlreadyExists from Unwrap")
	}
}

func TestIsNotSupported(t *testing.T) {
	if !IsNotSupported(ErrNotSupported) {
		t.Errorf("expected true for ErrNotSupported")
	}
	if IsNotSupported(nil) {
		t.Errorf("expected false for nil")
	}
	if IsNotSupported(errors.New("other")) {
		t.Errorf("expected false for unrelated error")
	}
}

func TestExitCodeDefault(t *testing.T) {
	err := WithErrorCode(errors.New("x"), "UNKNOWN_CODE_SHOULD_HIT_DEFAULT")
	got := ExitCode(err)
	if got != 1 {
		t.Errorf("expected 1 for default branch, got %d", got)
	}
}

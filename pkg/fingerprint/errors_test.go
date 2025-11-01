package fingerprint

import (
	"errors"
	"testing"
)

func TestFingerprintError_WithErrorCodeAndUnwrap(t *testing.T) {
	if WithErrorCode(nil, "X") != nil {
		t.Errorf("expected nil when err is nil")
	}

	base := errors.New("base")
	wrapped := WithErrorCode(base, "CODE123")
	if wrapped.(*withCodeError).Code() != "CODE123" {
		t.Errorf("expected CODE123")
	}
	if !errors.Is(wrapped, base) {
		t.Errorf("unwrap mismatch")
	}
}

func TestFingerprintError_NewSourceRequiredError(t *testing.T) {
	err := NewSourceRequiredError()
	if !errors.Is(err, ErrSourceRequired) {
		t.Errorf("expected ErrSourceRequired")
	}
	if ErrorCode(err) != errorCodeSourceRequired {
		t.Errorf("expected code source required")
	}
}

func TestFingerprintError_NewSourceConflictError(t *testing.T) {
	err := NewSourceConflictError()
	if !errors.Is(err, ErrSourceConflict) {
		t.Errorf("expected ErrSourceConflict")
	}
	if ErrorCode(err) != errorCodeSourceConflict {
		t.Errorf("expected code source conflict")
	}
}

func TestFingerprintError_NewStorageDisabledError(t *testing.T) {
	err := NewStorageDisabledError()
	if !errors.Is(err, ErrStorageDisabled) {
		t.Errorf("expected ErrStorageDisabled")
	}
	if ErrorCode(err) != errorCodeStorageDisabled {
		t.Errorf("expected code storage disabled")
	}
}

func TestFingerprintError_WrapSyncError(t *testing.T) {
	if WrapSyncError(nil) != nil {
		t.Errorf("expected nil for nil input")
	}
	e := errors.New("sync fail")
	err := WrapSyncError(e)
	if !errors.Is(err, e) {
		t.Errorf("unwrap mismatch")
	}
	if ErrorCode(err) != errorCodeSyncFailed {
		t.Errorf("expected sync failed code")
	}
}

func TestFingerprintError_ErrorCodeBranches(t *testing.T) {
	if ErrorCode(nil) != "" {
		t.Errorf("expected empty for nil")
	}

	coded := WithErrorCode(errors.New("x"), "CUSTOM")
	if ErrorCode(coded) != "CUSTOM" {
		t.Errorf("expected CUSTOM")
	}

	if ErrorCode(ErrSourceRequired) != errorCodeSourceRequired {
		t.Errorf("expected source required code")
	}
	if ErrorCode(ErrSourceConflict) != errorCodeSourceConflict {
		t.Errorf("expected source conflict code")
	}
	if ErrorCode(ErrStorageDisabled) != errorCodeStorageDisabled {
		t.Errorf("expected storage disabled code")
	}
	if ErrorCode(errors.New("other")) != errorCodeSyncFailed {
		t.Errorf("expected sync failed default code")
	}
}

func TestFingerprintError_ExitCode(t *testing.T) {
	tests := []struct {
		err      error
		expected int
	}{
		{nil, 0},
		{ErrSourceRequired, 2},
		{ErrSourceConflict, 2},
		{ErrStorageDisabled, 7},
		{errors.New("other"), 1}, // default branch
	}
	for _, tt := range tests {
		got := ExitCode(tt.err)
		if got != tt.expected {
			t.Errorf("ExitCode(%v)=%d, want %d", tt.err, got, tt.expected)
		}
	}
}

func TestFingerprintError_HTTPStatus(t *testing.T) {
	tests := []struct {
		err      error
		expected int
	}{
		{nil, 200},
		{ErrSourceRequired, 400},
		{ErrSourceConflict, 400},
		{ErrStorageDisabled, 503},
		{errors.New("other"), 500}, // default branch
	}
	for _, tt := range tests {
		got := HTTPStatus(tt.err)
		if got != tt.expected {
			t.Errorf("HTTPStatus(%v)=%d, want %d", tt.err, got, tt.expected)
		}
	}
}

func TestFingerprintError_Suggestions(t *testing.T) {
	tests := []struct {
		code string
		want bool
	}{
		{errorCodeSourceRequired, true},
		{errorCodeSourceConflict, true},
		{errorCodeStorageDisabled, true},
		{errorCodeSyncFailed, true},
		{"UNKNOWN_CODE", false},
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
		t.Errorf("expected nil for nil err")
	}
}

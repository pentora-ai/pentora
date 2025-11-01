package scanexec

import (
	"errors"
	"testing"
)

func TestScanexecError_WithErrorCodeAndMethods(t *testing.T) {
	// nil input
	if WithErrorCode(nil, "X") != nil {
		t.Errorf("expected nil for nil input")
	}

	base := errors.New("base")
	wrapped := WithErrorCode(base, "CODE123").(*codedError)
	if wrapped.Code() != "CODE123" {
		t.Errorf("expected CODE123")
	}
	if wrapped.Error() != "base" {
		t.Errorf("unexpected message")
	}
	if !errors.Is(wrapped, base) {
		t.Errorf("unwrap mismatch")
	}
}

func TestScanexecError_ErrorCodeBranches(t *testing.T) {
	if ErrorCode(nil) != "" {
		t.Errorf("expected empty for nil")
	}

	// coded error path
	coded := WithErrorCode(errors.New("x"), "CUSTOM")
	if ErrorCode(coded) != "CUSTOM" {
		t.Errorf("expected CUSTOM")
	}

	if ErrorCode(ErrNoTargets) != errorCodeInvalidTarget {
		t.Errorf("expected invalid target code")
	}
	if ErrorCode(ErrConflictingDiscoveryFlags) != errorCodeConflictingDiscovery {
		t.Errorf("expected conflicting discovery code")
	}
	if ErrorCode(errors.New("random")) != errorCodeScanFailure {
		t.Errorf("expected scan failure default")
	}
}

func TestScanexecError_ExitCode(t *testing.T) {
	tests := []struct {
		err      error
		expected int
	}{
		{nil, 0},
		{WithErrorCode(errors.New("x"), errorCodeInvalidTarget), 2},
		{WithErrorCode(errors.New("x"), errorCodeConflictingDiscovery), 2},
		{WithErrorCode(errors.New("x"), "UNKNOWN"), 1}, // default
	}
	for _, tt := range tests {
		if got := ExitCode(tt.err); got != tt.expected {
			t.Errorf("ExitCode(%v)=%d, want %d", tt.err, got, tt.expected)
		}
	}
}

func TestScanexecError_HTTPStatus(t *testing.T) {
	tests := []struct {
		err      error
		expected int
	}{
		{nil, 200},
		{WithErrorCode(errors.New("x"), errorCodeInvalidTarget), 400},
		{WithErrorCode(errors.New("x"), errorCodeConflictingDiscovery), 400},
		{WithErrorCode(errors.New("x"), "OTHER"), 500}, // default
	}
	for _, tt := range tests {
		if got := HTTPStatus(tt.err); got != tt.expected {
			t.Errorf("HTTPStatus(%v)=%d, want %d", tt.err, got, tt.expected)
		}
	}
}

func TestScanexecError_Suggestions(t *testing.T) {
	tests := []struct {
		code string
		want int
	}{
		{errorCodeInvalidTarget, 2},
		{errorCodeConflictingDiscovery, 2},
		{errorCodeScanFailure, 2}, // default suggestions
	}
	for _, tt := range tests {
		err := WithErrorCode(errors.New("x"), tt.code)
		sugs := Suggestions(err)
		if len(sugs) != tt.want {
			t.Errorf("expected %d suggestions for %v, got %d", tt.want, tt.code, len(sugs))
		}
	}
	if Suggestions(nil) != nil {
		t.Errorf("expected nil for nil err")
	}
}

func TestScanexecError_NewInvalidTargetError(t *testing.T) {
	reason := errors.New("reason")
	// input provided
	err1 := NewInvalidTargetError("host", reason)
	if ErrorCode(err1) != errorCodeInvalidTarget {
		t.Errorf("expected invalid target code")
	}
	if err1.Error() == "" {
		t.Errorf("expected message")
	}
	// empty input, should wrap ErrNoTargets
	err2 := NewInvalidTargetError("", reason)
	if !errors.Is(err2, ErrNoTargets) {
		t.Errorf("expected ErrNoTargets")
	}
}

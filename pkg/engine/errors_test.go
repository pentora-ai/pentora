package engine

import (
	"errors"
	"testing"
)

func TestEngineError_WithErrorCodeAndUnwrap(t *testing.T) {
	if WithErrorCode(nil, "X") != nil {
		t.Errorf("expected nil for nil input")
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

func TestEngineError_WrapLoadError(t *testing.T) {
	if WrapLoadError(nil) != nil {
		t.Errorf("expected nil for nil input")
	}
	base := errors.New("loadfail")
	err := WrapLoadError(base)
	if !errors.Is(err, ErrLoadFailed) {
		t.Errorf("expected ErrLoadFailed")
	}
	if ErrorCode(err) != errorCodeLoadFailed {
		t.Errorf("expected load failed code")
	}
}

func TestEngineError_NewUnsupportedFormatError(t *testing.T) {
	err := NewUnsupportedFormatError("xml")
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Errorf("expected unsupported format error")
	}
	if ErrorCode(err) != errorCodeUnsupportedFormat {
		t.Errorf("expected unsupported format code")
	}
}

func TestEngineError_WrapMarshalError(t *testing.T) {
	if WrapMarshalError(nil) != nil {
		t.Errorf("expected nil for nil input")
	}
	e := errors.New("marshal")
	err := WrapMarshalError(e)
	if !errors.Is(err, e) {
		t.Errorf("expected unwrap to original error")
	}
	if ErrorCode(err) != errorCodeMarshalFailed {
		t.Errorf("expected marshal failed code")
	}
}

func TestEngineError_WrapWriteError(t *testing.T) {
	if WrapWriteError(nil) != nil {
		t.Errorf("expected nil for nil input")
	}
	e := errors.New("write")
	err := WrapWriteError(e)
	if !errors.Is(err, ErrWriteFailed) {
		t.Errorf("expected ErrWriteFailed")
	}
	if ErrorCode(err) != errorCodeWriteFailed {
		t.Errorf("expected write failed code")
	}
}

func TestEngineError_WrapInvalidDAG(t *testing.T) {
	if WrapInvalidDAG(nil) != nil {
		t.Errorf("expected nil for nil input")
	}
	e := errors.New("invalid")
	err := WrapInvalidDAG(e)
	if !errors.Is(err, ErrInvalidDAG) {
		t.Errorf("expected ErrInvalidDAG")
	}
	if ErrorCode(err) != errorCodeInvalidDAG {
		t.Errorf("expected invalid dag code")
	}
}

func TestEngineError_ErrorCodeBranches(t *testing.T) {
	if ErrorCode(nil) != "" {
		t.Errorf("expected empty for nil")
	}

	coded := WithErrorCode(errors.New("x"), "CUSTOM")
	if ErrorCode(coded) != "CUSTOM" {
		t.Errorf("expected CUSTOM")
	}

	if ErrorCode(ErrLoadFailed) != errorCodeLoadFailed {
		t.Errorf("expected load failed code")
	}
	if ErrorCode(ErrUnsupportedFormat) != errorCodeUnsupportedFormat {
		t.Errorf("expected unsupported format code")
	}
	if ErrorCode(ErrWriteFailed) != errorCodeWriteFailed {
		t.Errorf("expected write failed code")
	}
	if ErrorCode(ErrInvalidDAG) != errorCodeInvalidDAG {
		t.Errorf("expected invalid dag code")
	}
	if ErrorCode(errors.New("other")) != errorCodeMarshalFailed {
		t.Errorf("expected marshal failed default code")
	}
}

func TestEngineError_ExitCode(t *testing.T) {
	tests := []struct {
		err      error
		expected int
	}{
		{nil, 0},
		{ErrUnsupportedFormat, 2},
		{ErrLoadFailed, 4},
		{ErrWriteFailed, 1},
		{ErrInvalidDAG, 2},
		{errors.New("unknown"), 1}, // default branch
	}
	for _, tt := range tests {
		got := ExitCode(tt.err)
		if got != tt.expected {
			t.Errorf("ExitCode(%v)=%d, want %d", tt.err, got, tt.expected)
		}
	}
}

func TestEngineError_HTTPStatus(t *testing.T) {
	tests := []struct {
		err      error
		expected int
	}{
		{nil, 200},
		{ErrUnsupportedFormat, 400},
		{ErrInvalidDAG, 400},
		{ErrLoadFailed, 404},
		{ErrWriteFailed, 500},
		{errors.New("other"), 500}, // default branch
	}
	for _, tt := range tests {
		got := HTTPStatus(tt.err)
		if got != tt.expected {
			t.Errorf("HTTPStatus(%v)=%d, want %d", tt.err, got, tt.expected)
		}
	}
}

func TestEngineError_Suggestions(t *testing.T) {
	tests := []struct {
		code string
		want bool
	}{
		{errorCodeLoadFailed, true},
		{errorCodeUnsupportedFormat, true},
		{errorCodeWriteFailed, true},
		{errorCodeInvalidDAG, true},
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

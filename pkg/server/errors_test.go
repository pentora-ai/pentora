package server

import (
	"errors"
	"testing"
)

func TestServerError_WithErrorCodeAndUnwrap(t *testing.T) {
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

func TestServerError_NewInvalidPortError(t *testing.T) {
	err := NewInvalidPortError(99999)
	if !errors.Is(err, ErrInvalidPort) {
		t.Errorf("expected invalid port err")
	}
	if ErrorCode(err) != errorCodeInvalidPort {
		t.Errorf("expected code invalid_port")
	}
}

func TestServerError_NewInvalidConcurrencyError(t *testing.T) {
	err := NewInvalidConcurrencyError(0)
	if !errors.Is(err, ErrInvalidConcurrency) {
		t.Errorf("expected invalid concurrency err")
	}
	if ErrorCode(err) != errorCodeInvalidConcurrency {
		t.Errorf("expected code invalid_concurrency")
	}
}

func TestServerError_NewFeaturesDisabledError(t *testing.T) {
	err := NewFeaturesDisabledError()
	if !errors.Is(err, ErrFeaturesDisabled) {
		t.Errorf("expected features disabled err")
	}
	if ErrorCode(err) != errorCodeFeaturesDisabled {
		t.Errorf("expected code features_disabled")
	}
}

func TestServerError_WrapInvalidConfig(t *testing.T) {
	if WrapInvalidConfig(nil) != nil {
		t.Errorf("expected nil for nil input")
	}
	e := errors.New("bad")
	err := WrapInvalidConfig(e)
	if !errors.Is(err, e) {
		t.Errorf("unwrap mismatch")
	}
	if ErrorCode(err) != errorCodeInvalidConfig {
		t.Errorf("expected invalid_config code")
	}
}

func TestServerError_WrapStorageInit(t *testing.T) {
	if WrapStorageInit(nil) != nil {
		t.Errorf("expected nil for nil input")
	}
	err := WrapStorageInit(errors.New("s"))
	if ErrorCode(err) != errorCodeStorageInitFailed {
		t.Errorf("expected storage_init_failed code")
	}
}

func TestServerError_WrapPluginInit(t *testing.T) {
	if WrapPluginInit(nil) != nil {
		t.Errorf("expected nil for nil input")
	}
	err := WrapPluginInit(errors.New("s"))
	if ErrorCode(err) != errorCodePluginInitFailed {
		t.Errorf("expected plugin_init_failed code")
	}
}

func TestServerError_WrapAppInit(t *testing.T) {
	if WrapAppInit(nil) != nil {
		t.Errorf("expected nil for nil input")
	}
	err := WrapAppInit(errors.New("s"))
	if ErrorCode(err) != errorCodeAppInitFailed {
		t.Errorf("expected app_init_failed code")
	}
}

func TestServerError_WrapRuntime(t *testing.T) {
	if WrapRuntime(nil) != nil {
		t.Errorf("expected nil for nil input")
	}
	err := WrapRuntime(errors.New("s"))
	if ErrorCode(err) != errorCodeRuntimeFailed {
		t.Errorf("expected runtime_failed code")
	}
}

func TestServerError_ErrorCodeBranches(t *testing.T) {
	if ErrorCode(nil) != "" {
		t.Errorf("expected empty for nil")
	}

	coded := WithErrorCode(errors.New("x"), "CUSTOM")
	if ErrorCode(coded) != "CUSTOM" {
		t.Errorf("expected CUSTOM")
	}

	if ErrorCode(ErrInvalidPort) != errorCodeInvalidPort {
		t.Errorf("expected invalid port code")
	}
	if ErrorCode(ErrInvalidConcurrency) != errorCodeInvalidConcurrency {
		t.Errorf("expected invalid concurrency code")
	}
	if ErrorCode(ErrFeaturesDisabled) != errorCodeFeaturesDisabled {
		t.Errorf("expected features disabled code")
	}
	if ErrorCode(ErrConfigUnavailable) != errorCodeConfigUnavailable {
		t.Errorf("expected config unavailable code")
	}
	if ErrorCode(errors.New("random")) != errorCodeRuntimeFailed {
		t.Errorf("expected runtime_failed fallback")
	}
}

func TestServerError_ExitCode(t *testing.T) {
	tests := []struct {
		err      error
		expected int
	}{
		{nil, 0},
		{ErrInvalidPort, 2},
		{ErrInvalidConcurrency, 2},
		{ErrFeaturesDisabled, 2},
		{ErrConfigUnavailable, 1},
		{WithErrorCode(errors.New("x"), errorCodeStorageInitFailed), 7},
		{WithErrorCode(errors.New("x"), errorCodePluginInitFailed), 7},
		{WithErrorCode(errors.New("x"), errorCodeAppInitFailed), 7},
		{WithErrorCode(errors.New("x"), "UNKNOWN_CODE"), 1}, // default branch
	}
	for _, tt := range tests {
		if got := ExitCode(tt.err); got != tt.expected {
			t.Errorf("ExitCode(%v)=%d, want %d", tt.err, got, tt.expected)
		}
	}
}

func TestServerError_HTTPStatus(t *testing.T) {
	tests := []struct {
		err      error
		expected int
	}{
		{nil, 200},
		{ErrInvalidPort, 400},
		{ErrInvalidConcurrency, 400},
		{ErrFeaturesDisabled, 400},
		{ErrConfigUnavailable, 500},
		{errors.New("x"), 500}, // default branch
	}
	for _, tt := range tests {
		if got := HTTPStatus(tt.err); got != tt.expected {
			t.Errorf("HTTPStatus(%v)=%d, want %d", tt.err, got, tt.expected)
		}
	}
}

func TestServerError_Suggestions(t *testing.T) {
	tests := []struct {
		code string
		want bool
	}{
		{errorCodeInvalidPort, true},
		{errorCodeInvalidConcurrency, true},
		{errorCodeFeaturesDisabled, true},
		{errorCodeConfigUnavailable, true},
		{errorCodeInvalidConfig, true},
		{errorCodeStorageInitFailed, true},
		{errorCodePluginInitFailed, true},
		{errorCodeAppInitFailed, true},
		{errorCodeRuntimeFailed, true},
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

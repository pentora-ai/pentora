package fingerprint

import (
	"context"
	"errors"
	"testing"
)

// --- Mock Resolver ---

type mockResolver struct{}

func (m mockResolver) Resolve(ctx context.Context, input Input) (Result, error) {
	return Result{}, nil
}

// --- Test Helpers ---
func resetGlobals() {
	WarmWithExternalFunc = WarmWithExternal
	GetFingerprintResolverFn = GetFingerprintResolver
}

func TestNewResolverFactory(t *testing.T) {
	rules := []StaticRule{{Match: "apache", Product: "Apache"}}
	f := NewResolverFactory(rules, true)

	if f == nil {
		t.Fatalf("expected non-nil factory")
	}
	if !f.enableAI {
		t.Errorf("expected enableAI=true, got false")
	}
	if len(f.staticRules) != 1 {
		t.Errorf("expected 1 static rule, got %d", len(f.staticRules))
	}
	if f.staticRules[0].Match != "apache" {
		t.Errorf("expected Match='apache', got %q", f.staticRules[0].Match)
	}
}

func TestResolverFactory_Get_AIEnabled(t *testing.T) {
	f := &ResolverFactory{enableAI: true}
	res, err := f.Get()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, errors.New("AIResolver is not available in this build")) &&
		err.Error() != "AIResolver is not available in this build" {
		t.Errorf("unexpected error: %v", err)
	}
	if res != nil {
		t.Errorf("expected nil resolver, got %v", res)
	}
}

func TestResolverFactory_Get_NoRules(t *testing.T) {
	calledWarm := false
	calledGet := false
	resetGlobals()

	WarmWithExternalFunc = func(arg string) {
		calledWarm = true
	}
	GetFingerprintResolverFn = func() Resolver {
		calledGet = true
		return mockResolver{}
	}

	f := &ResolverFactory{staticRules: nil, enableAI: false}
	res, err := f.Get()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !calledWarm {
		t.Errorf("expected WarmWithExternalFunc to be called")
	}
	if !calledGet {
		t.Errorf("expected GetFingerprintResolverFn to be called")
	}
	if _, ok := res.(mockResolver); !ok {
		t.Errorf("expected mockResolver, got %T", res)
	}
	resetGlobals()
}

func TestResolverFactory_Get_WithRules(t *testing.T) {
	resetGlobals()
	rule := StaticRule{Match: "nginx", Product: "Nginx"}
	f := &ResolverFactory{staticRules: []StaticRule{rule}, enableAI: false}

	res, err := f.Get()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rb, ok := res.(*RuleBasedResolver)
	if !ok {
		t.Fatalf("expected *RuleBasedResolver, got %T", res)
	}

	if len(rb.rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rb.rules))
	}
	got := rb.rules[0]
	if got.Match != rule.Match || got.Product != rule.Product {
		t.Errorf("expected Match=%q Product=%q, got Match=%q Product=%q",
			rule.Match, rule.Product, got.Match, got.Product)
	}
}

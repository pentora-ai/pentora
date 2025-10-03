// Package fingerprint provides resolver selection mechanism.
package fingerprint

import (
	"errors"
)

// ResolverType defines which implementation to use.
type ResolverType string

// ResolverRuleBased represents a resolver type that uses rule-based logic for fingerprint resolution.
const (
	ResolverRuleBased ResolverType = "rule_based"
	ResolverAI        ResolverType = "ai"
)

// ResolverFactory creates a fingerprint resolver dynamically based on runtime configuration.
// In OSS, AI resolver is never registered or returned.
type ResolverFactory struct {
	staticRules []StaticRule
	enableAI    bool
}

// NewResolverFactory creates a factory with options.
func NewResolverFactory(staticRules []StaticRule, enableAI bool) *ResolverFactory {
	return &ResolverFactory{
		staticRules: staticRules,
		enableAI:    enableAI,
	}
}

// Get returns the correct resolver implementation based on configuration or feature flag.
func (f *ResolverFactory) Get() (Resolver, error) {
	if f.enableAI {
		// In OSS build, this must return an error to avoid dependency leaks.
		return nil, errors.New("AIResolver is not available in this build")
	}

	if len(f.staticRules) == 0 {
		WarmWithExternal("")
		return GetFingerprintResolver(), nil
	}

	return NewRuleBasedResolver(f.staticRules), nil
}

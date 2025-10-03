// Package fingerprint provides mechanisms for managing and resolving fingerprinting rules.
// It maintains a registry for the currently active fingerprint resolver, allowing dynamic
// switching between built-in and externally loaded rule sets. The package initializes with
// a default rule-based resolver using built-in rules, but can be warmed with external rules
// if available. It exposes functions to register a new resolver and retrieve the currently
// active resolver for fingerprint operations.
package fingerprint

// Holds the currently active resolver (default: rule-based)
var activeResolver Resolver

// init initializes the activeResolver with a new RuleBasedResolver using the built-in rules.
// This function is automatically invoked when the package is initialized.
func init() {
	activeResolver = NewRuleBasedResolver(loadBuiltinRules())
}

// WarmWithExternal attempts to load fingerprinting rules from an external catalog located in the specified cacheDir.
// If successful and rules are found, it registers a new RuleBasedResolver with these external rules.
// Otherwise, it falls back to registering a RuleBasedResolver with the built-in rules.
func WarmWithExternal(cacheDir string) {
	if rules, err := loadExternalCatalog(cacheDir); err == nil && len(rules) > 0 {
		RegisterFingerprintResolver(NewRuleBasedResolver(rules))
		return
	}
	RegisterFingerprintResolver(NewRuleBasedResolver(loadBuiltinRules()))
}

// RegisterFingerprintResolver sets the active fingerprint resolver to the provided resolver.
// This function replaces any previously registered resolver with the new one.
//
// r: The FingerprintResolver implementation to register as the active resolver.
func RegisterFingerprintResolver(r Resolver) {
	activeResolver = r
}

// GetFingerprintResolver returns the currently active FingerprintResolver instance.
// This function provides access to the resolver used for fingerprint operations.
func GetFingerprintResolver() Resolver {
	return activeResolver
}

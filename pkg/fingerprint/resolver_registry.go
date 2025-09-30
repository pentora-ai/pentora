package fingerprint

// Holds the currently active resolver (default: rule-based)
var activeResolver FingerprintResolver

func init() {
	activeResolver = NewRuleBasedResolver(loadBuiltinRules())
}

// WarmWithExternal attempts to preload catalog from cacheDir, falling back to builtin rules.
func WarmWithExternal(cacheDir string) {
	if rules, err := loadExternalCatalog(cacheDir); err == nil && len(rules) > 0 {
		RegisterFingerprintResolver(NewRuleBasedResolver(rules))
		return
	}
	RegisterFingerprintResolver(NewRuleBasedResolver(loadBuiltinRules()))
}

// Allows external systems (e.g., AI module) to override the active resolver
func RegisterFingerprintResolver(r FingerprintResolver) {
	activeResolver = r
}

// Returns the currently active resolver (used by orchestrator and modules)
func GetFingerprintResolver() FingerprintResolver {
	return activeResolver
}

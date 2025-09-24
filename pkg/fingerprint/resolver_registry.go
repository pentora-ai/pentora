package fingerprint

// Holds the currently active resolver (default: rule-based)
var activeResolver FingerprintResolver

func init() {
	activeResolver = NewRuleBasedResolver(loadBuiltinRules())
}

// Allows external systems (e.g., AI module) to override the active resolver
func RegisterFingerprintResolver(r FingerprintResolver) {
	activeResolver = r
}

// Returns the currently active resolver (used by orchestrator and modules)
func GetFingerprintResolver() FingerprintResolver {
	return activeResolver
}

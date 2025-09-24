// pkg/fingerprint/loader.go
// Package fingerprint provides functionality to resolve service banners into structured metadata.
package fingerprint

import (
	_ "embed"
	"fmt"
)

//go:embed data/fingerprint_db.yaml
var embeddedFingerprintYAML []byte

// loadBuiltinRules loads fingerprint rules embedded in the binary.
func loadBuiltinRules() []StaticRule {
	rules, err := parseFingerprintYAML(embeddedFingerprintYAML)
	if err != nil {
		fmt.Printf("Failed to load embedded fingerprint rules: %v\n", err)
		return nil
	}
	return rules
}

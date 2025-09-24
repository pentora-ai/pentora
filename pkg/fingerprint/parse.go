// pkg/fingerprint/parse.go
package fingerprint

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// parseFingerprintYAML parses raw YAML bytes into a list of StaticRule entries.
// Each rule defines how to identify a software or service by inspecting banner content.
func parseFingerprintYAML(data []byte) ([]StaticRule, error) {
	var rules []StaticRule

	// Unmarshal YAML data into []StaticRule
	err := yaml.Unmarshal(data, &rules)
	if err != nil {
		return nil, fmt.Errorf("failed to parse fingerprint YAML: %w", err)
	}

	// Optional validation: ensure required fields are set
	for i, rule := range rules {
		if rule.Protocol == "" || rule.Vendor == "" || rule.Product == "" || rule.Match == "" || rule.CPE == "" {
			return nil, fmt.Errorf("invalid fingerprint rule at index %d: missing required fields", i)
		}
	}

	return rules, nil
}

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

	if err := yaml.Unmarshal(data, &rules); err == nil && len(rules) > 0 {
		return rules, validateRules(rules)
	}

	var wrapper struct {
		Rules []StaticRule `yaml:"rules"`
	}
	if err := yaml.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("failed to parse fingerprint YAML: %w", err)
	}
	rules = wrapper.Rules
	return rules, validateRules(rules)
}

func validateRules(rules []StaticRule) error {
	if len(rules) == 0 {
		return fmt.Errorf("no fingerprint rules found")
	}

	// Optional validation: ensure required fields are set
	for i, rule := range rules {
		if rule.Protocol == "" || rule.Vendor == "" || rule.Product == "" || rule.Match == "" || rule.CPE == "" {
			return fmt.Errorf("invalid fingerprint rule at index %d: missing required fields", i)
		}
	}
	return nil
}

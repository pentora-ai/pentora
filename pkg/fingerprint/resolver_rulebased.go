// Package fingerprint provides a static, rule-based fingerprint resolver implementation.
package fingerprint

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// StaticRule defines a fingerprint rule loaded from fingerprint_db.yaml.
type StaticRule struct {
	ID                string `yaml:"id"`
	Protocol          string `yaml:"protocol"`
	Description       string `yaml:"description"`
	Product           string `yaml:"product"`
	Vendor            string `yaml:"vendor"`
	CPE               string `yaml:"cpe"`
	Match             string `yaml:"match"`              // regex or plain string
	VersionExtraction string `yaml:"version_extraction"` // regex with capturing group

	// Anti-patterns and exclusions
	ExcludePatterns     []string `yaml:"exclude_patterns"`
	SoftExcludePatterns []string `yaml:"soft_exclude_patterns"`

	// Confidence and scoring metadata
	PatternStrength float64 `yaml:"pattern_strength"`
	PortBonuses     []int   `yaml:"port_bonuses"`

	// Binary verification fields
	BinaryMinLength int      `yaml:"binary_min_length"`
	BinaryMagic     []string `yaml:"binary_magic"`

	// Compiled expressions (not serialized)
	matchRegex   *regexp.Regexp
	versionRegex *regexp.Regexp
	excludeRegex []*regexp.Regexp
	softExRegex  []*regexp.Regexp
}

// RuleBasedResolver uses a preloaded list of static rules to resolve banners into metadata.
type RuleBasedResolver struct {
	rules []StaticRule
}

// NewRuleBasedResolver initializes a resolver using fingerprint rules loaded from a YAML file.
func NewRuleBasedResolver(rules []StaticRule) *RuleBasedResolver {
	return &RuleBasedResolver{rules: prepareRules(rules)}
}

// Resolve attempts to identify a fingerprint based on the provided FingerprintInput.
// It normalizes the input banner, iterates through the resolver's rules, and checks for a matching protocol and banner pattern.
// If a rule matches, it extracts the version (if available) using the rule's versionRegex, and returns a FingerprintResult
// populated with the rule's metadata and a high confidence score. If no rule matches, it returns an error.
//
// Parameters:
//
//	ctx - The context for cancellation and deadlines.
//	in  - The FingerprintInput containing protocol and banner information.
//
// Returns:
//
//	Result - The result of the fingerprinting process, populated if a rule matches.
//	error             - An error if no matching rule is found.
func (r *RuleBasedResolver) Resolve(_ context.Context, in Input) (Result, error) {
	normalizedBanner := strings.ToLower(in.Banner)

	for _, rule := range r.rules {
		if rule.Protocol != in.Protocol {
			continue // skip unrelated protocol
		}

		if rule.matchRegex.MatchString(normalizedBanner) {
			version := ""
			matches := rule.versionRegex.FindStringSubmatch(normalizedBanner)
			if len(matches) >= 2 {
				version = matches[1]
			}

			return Result{
				Product:     rule.Product,
				Vendor:      rule.Vendor,
				Version:     version,
				CPE:         rule.CPE,
				Confidence:  1.0, // static match is high confidence
				Technique:   "static",
				Description: rule.Description,
			}, nil
		}
	}

	return Result{}, fmt.Errorf("no matching rule found")
}

func prepareRules(rules []StaticRule) []StaticRule {
	compiled := make([]StaticRule, 0, len(rules))
	for _, rule := range rules {
		copy := rule
		if copy.matchRegex == nil {
			copy.matchRegex = regexp.MustCompile(copy.Match)
		}
		if copy.versionRegex == nil && copy.VersionExtraction != "" {
			copy.versionRegex = regexp.MustCompile(copy.VersionExtraction)
		}
		// Defaults
		if copy.PatternStrength == 0 {
			copy.PatternStrength = 0.80
		}
		// Compile exclude patterns
		if len(copy.ExcludePatterns) > 0 && copy.excludeRegex == nil {
			for _, p := range copy.ExcludePatterns {
				copy.excludeRegex = append(copy.excludeRegex, regexp.MustCompile(p))
			}
		}
		if len(copy.SoftExcludePatterns) > 0 && copy.softExRegex == nil {
			for _, p := range copy.SoftExcludePatterns {
				copy.softExRegex = append(copy.softExRegex, regexp.MustCompile(p))
			}
		}
		compiled = append(compiled, copy)
	}
	return compiled
}

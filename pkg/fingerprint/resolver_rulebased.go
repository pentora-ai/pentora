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

	// Compiled expressions (not serialized)
	matchRegex   *regexp.Regexp
	versionRegex *regexp.Regexp
}

// RuleBasedResolver uses a preloaded list of static rules to resolve banners into metadata.
type RuleBasedResolver struct {
	rules []StaticRule
}

// NewRuleBasedResolver initializes a resolver using fingerprint rules loaded from a YAML file.
func NewRuleBasedResolver(rules []StaticRule) *RuleBasedResolver {
	return &RuleBasedResolver{rules: rules}
}

// Resolve applies all matching rules to the input and returns a FingerprintResult if successful.
func (r *RuleBasedResolver) Resolve(ctx context.Context, in FingerprintInput) (FingerprintResult, error) {
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

			return FingerprintResult{
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

	return FingerprintResult{}, fmt.Errorf("no matching rule found")
}

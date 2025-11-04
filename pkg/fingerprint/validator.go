package fingerprint

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// ValidationError represents a validation error with severity and location information.
type ValidationError struct {
	RuleID   string // Rule ID where error occurred
	Field    string // Field name
	Message  string // Error message
	Severity string // "error" or "warning"
}

// DatabaseValidationResult contains the results of a database validation run.
type DatabaseValidationResult struct {
	Errors    []ValidationError
	Warnings  []ValidationError
	RuleCount int
}

// IsValid returns true if there are no errors (warnings are allowed).
func (r *DatabaseValidationResult) IsValid() bool {
	return len(r.Errors) == 0
}

// Validator validates fingerprint database YAML rules.
type Validator struct {
	strict bool // Treat warnings as errors
}

// NewValidator creates a new Validator instance.
func NewValidator(strict bool) *Validator {
	return &Validator{strict: strict}
}

// Validate validates a list of static rules.
func (v *Validator) Validate(rules []StaticRule) *DatabaseValidationResult {
	result := &DatabaseValidationResult{
		Errors:    make([]ValidationError, 0),
		Warnings:  make([]ValidationError, 0),
		RuleCount: len(rules),
	}

	seenIDs := make(map[string]bool)

	for _, rule := range rules {
		// Check required fields
		v.validateRequiredFields(rule, result)

		// Check for duplicate IDs
		v.validateDuplicateID(rule, seenIDs, result)

		// Validate regex patterns
		v.validateRegexPatterns(rule, result)

		// Validate CPE format
		v.validateCPEFormat(rule, result)

		// Validate confidence metadata
		v.validateConfidenceMetadata(rule, result)
	}

	return result
}

// validateRequiredFields checks that all required fields are present and non-empty.
func (v *Validator) validateRequiredFields(rule StaticRule, result *DatabaseValidationResult) {
	requiredFields := map[string]string{
		"id":       rule.ID,
		"protocol": rule.Protocol,
		"product":  rule.Product,
		"match":    rule.Match,
	}

	for field, value := range requiredFields {
		if strings.TrimSpace(value) == "" {
			result.Errors = append(result.Errors, ValidationError{
				RuleID:   rule.ID,
				Field:    field,
				Message:  fmt.Sprintf("required field '%s' is empty or missing", field),
				Severity: "error",
			})
		}
	}

	// Vendor is recommended but not required (warning only)
	if strings.TrimSpace(rule.Vendor) == "" {
		result.Warnings = append(result.Warnings, ValidationError{
			RuleID:   rule.ID,
			Field:    "vendor",
			Message:  "vendor field is empty (recommended)",
			Severity: "warning",
		})
	}

	// Description is recommended but not required (warning only)
	if strings.TrimSpace(rule.Description) == "" {
		result.Warnings = append(result.Warnings, ValidationError{
			RuleID:   rule.ID,
			Field:    "description",
			Message:  "description field is empty (recommended)",
			Severity: "warning",
		})
	}
}

// validateDuplicateID checks for duplicate rule IDs.
func (v *Validator) validateDuplicateID(rule StaticRule, seenIDs map[string]bool, result *DatabaseValidationResult) {
	if seenIDs[rule.ID] {
		result.Errors = append(result.Errors, ValidationError{
			RuleID:   rule.ID,
			Field:    "id",
			Message:  fmt.Sprintf("duplicate rule ID '%s'", rule.ID),
			Severity: "error",
		})
	}
	seenIDs[rule.ID] = true
}

// validateRegexPatterns validates regex syntax for match and version_extraction fields.
func (v *Validator) validateRegexPatterns(rule StaticRule, result *DatabaseValidationResult) {
	// Validate match pattern
	if rule.Match != "" {
		if _, err := regexp.Compile(rule.Match); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				RuleID:   rule.ID,
				Field:    "match",
				Message:  fmt.Sprintf("invalid regex syntax: %v", err),
				Severity: "error",
			})
		}
	}

	// Validate version_extraction pattern
	if rule.VersionExtraction != "" {
		re, err := regexp.Compile(rule.VersionExtraction)
		if err != nil {
			result.Errors = append(result.Errors, ValidationError{
				RuleID:   rule.ID,
				Field:    "version_extraction",
				Message:  fmt.Sprintf("invalid regex syntax: %v", err),
				Severity: "error",
			})
		} else if re.NumSubexp() == 0 {
			// Check for capturing group
			result.Warnings = append(result.Warnings, ValidationError{
				RuleID:   rule.ID,
				Field:    "version_extraction",
				Message:  "no capturing group found (version extraction will not work)",
				Severity: "warning",
			})
		}
	}

	// Validate exclude_patterns
	for _, pattern := range rule.ExcludePatterns {
		if _, err := regexp.Compile(pattern); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				RuleID:   rule.ID,
				Field:    "exclude_patterns",
				Message:  fmt.Sprintf("invalid regex syntax in exclude pattern '%s': %v", pattern, err),
				Severity: "error",
			})
		}
	}

	// Validate soft_exclude_patterns
	for _, pattern := range rule.SoftExcludePatterns {
		if _, err := regexp.Compile(pattern); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				RuleID:   rule.ID,
				Field:    "soft_exclude_patterns",
				Message:  fmt.Sprintf("invalid regex syntax in soft exclude pattern '%s': %v", pattern, err),
				Severity: "error",
			})
		}
	}
}

// validateCPEFormat validates CPE format (basic check).
func (v *Validator) validateCPEFormat(rule StaticRule, result *DatabaseValidationResult) {
	if rule.CPE == "" {
		result.Warnings = append(result.Warnings, ValidationError{
			RuleID:   rule.ID,
			Field:    "cpe",
			Message:  "CPE field is empty (recommended for vulnerability correlation)",
			Severity: "warning",
		})
		return
	}

	// Basic CPE format check (should start with "cpe:2.3:")
	if !strings.HasPrefix(rule.CPE, "cpe:2.3:") {
		result.Errors = append(result.Errors, ValidationError{
			RuleID:   rule.ID,
			Field:    "cpe",
			Message:  "CPE format should start with 'cpe:2.3:' (CPE 2.3 format)",
			Severity: "error",
		})
		return
	}

	// Check CPE structure (should have 13 components separated by colons)
	parts := strings.Split(rule.CPE, ":")
	if len(parts) != 13 {
		result.Warnings = append(result.Warnings, ValidationError{
			RuleID:   rule.ID,
			Field:    "cpe",
			Message:  fmt.Sprintf("CPE should have 13 colon-separated components (found %d)", len(parts)),
			Severity: "warning",
		})
	}
}

// validateConfidenceMetadata validates confidence scoring metadata.
func (v *Validator) validateConfidenceMetadata(rule StaticRule, result *DatabaseValidationResult) {
	// Check pattern_strength range (0.0 to 1.0)
	if rule.PatternStrength < 0.0 || rule.PatternStrength > 1.0 {
		result.Errors = append(result.Errors, ValidationError{
			RuleID:   rule.ID,
			Field:    "pattern_strength",
			Message:  fmt.Sprintf("pattern_strength must be between 0.0 and 1.0 (got %.2f)", rule.PatternStrength),
			Severity: "error",
		})
	}

	// Warn if pattern_strength is too low
	if rule.PatternStrength > 0 && rule.PatternStrength < 0.50 {
		result.Warnings = append(result.Warnings, ValidationError{
			RuleID:   rule.ID,
			Field:    "pattern_strength",
			Message:  fmt.Sprintf("pattern_strength is below threshold (%.2f < 0.50, rule may never match)", rule.PatternStrength),
			Severity: "warning",
		})
	}

	// Warn if no pattern_strength is set (defaults to 0.80 in prepareRules)
	if rule.PatternStrength == 0 {
		result.Warnings = append(result.Warnings, ValidationError{
			RuleID:   rule.ID,
			Field:    "pattern_strength",
			Message:  "pattern_strength not set (will default to 0.80)",
			Severity: "warning",
		})
	}

	// Check port_bonuses validity
	for _, port := range rule.PortBonuses {
		if port < 1 || port > 65535 {
			result.Errors = append(result.Errors, ValidationError{
				RuleID:   rule.ID,
				Field:    "port_bonuses",
				Message:  fmt.Sprintf("invalid port number %d (must be 1-65535)", port),
				Severity: "error",
			})
		}
	}
}

// LoadRulesFromFile loads static rules from a YAML file.
func LoadRulesFromFile(filePath string) ([]StaticRule, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Try parsing as wrapper struct with "rules:" key first
	var wrapper struct {
		Rules []StaticRule `yaml:"rules"`
	}
	if err := yaml.Unmarshal(data, &wrapper); err == nil && len(wrapper.Rules) > 0 {
		return wrapper.Rules, nil
	}

	// Fallback: try parsing as direct array
	var rules []StaticRule
	if err := yaml.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return rules, nil
}

// NewValidationError creates a new validation error with error and warning counts.
func NewValidationError(errorCount, warningCount int) error {
	return fmt.Errorf("validation failed: %d errors, %d warnings", errorCount, warningCount)
}

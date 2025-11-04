package fingerprint

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidator_ValidRule(t *testing.T) {
	validator := NewValidator(false)

	rules := []StaticRule{{
		ID:                "test.valid",
		Protocol:          "http",
		Product:           "TestProduct",
		Vendor:            "TestVendor",
		Description:       "Test description",
		CPE:               "cpe:2.3:a:vendor:product:*:*:*:*:*:*:*:*",
		Match:             `test`,
		VersionExtraction: `test\s+([\d\.]+)`,
		PatternStrength:   0.85,
		PortBonuses:       []int{80, 443},
	}}

	result := validator.Validate(rules)
	require.True(t, result.IsValid(), "valid rule should pass validation")
	require.Empty(t, result.Errors, "should have no errors")
}

func TestValidator_RequiredFields(t *testing.T) {
	validator := NewValidator(false)

	testCases := []struct {
		name          string
		rule          StaticRule
		expectedField string
	}{
		{
			name: "missing ID",
			rule: StaticRule{
				Protocol: "http",
				Product:  "Test",
				Match:    "test",
			},
			expectedField: "id",
		},
		{
			name: "missing protocol",
			rule: StaticRule{
				ID:      "test.missing_protocol",
				Product: "Test",
				Match:   "test",
			},
			expectedField: "protocol",
		},
		{
			name: "missing product",
			rule: StaticRule{
				ID:       "test.missing_product",
				Protocol: "http",
				Match:    "test",
			},
			expectedField: "product",
		},
		{
			name: "missing match",
			rule: StaticRule{
				ID:       "test.missing_match",
				Protocol: "http",
				Product:  "Test",
			},
			expectedField: "match",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.Validate([]StaticRule{tc.rule})
			require.False(t, result.IsValid(), "should fail validation")
			require.NotEmpty(t, result.Errors, "should have errors")

			found := false
			for _, err := range result.Errors {
				if err.Field == tc.expectedField {
					found = true
					break
				}
			}
			require.True(t, found, "should have error for field: %s", tc.expectedField)
		})
	}
}

func TestValidator_DuplicateID(t *testing.T) {
	validator := NewValidator(false)

	rules := []StaticRule{
		{
			ID:       "test.duplicate",
			Protocol: "http",
			Product:  "Test1",
			Match:    "test1",
		},
		{
			ID:       "test.duplicate", // Duplicate ID
			Protocol: "http",
			Product:  "Test2",
			Match:    "test2",
		},
	}

	result := validator.Validate(rules)
	require.False(t, result.IsValid(), "should fail validation")

	found := false
	for _, err := range result.Errors {
		if err.Field == "id" && err.RuleID == "test.duplicate" {
			found = true
			break
		}
	}
	require.True(t, found, "should detect duplicate ID")
}

func TestValidator_InvalidRegex(t *testing.T) {
	validator := NewValidator(false)

	testCases := []struct {
		name          string
		rule          StaticRule
		expectedField string
	}{
		{
			name: "invalid match regex",
			rule: StaticRule{
				ID:       "test.invalid_match",
				Protocol: "http",
				Product:  "Test",
				Match:    `[unclosed`,
			},
			expectedField: "match",
		},
		{
			name: "invalid version_extraction regex",
			rule: StaticRule{
				ID:                "test.invalid_version",
				Protocol:          "http",
				Product:           "Test",
				Match:             "test",
				VersionExtraction: `(unclosed`,
			},
			expectedField: "version_extraction",
		},
		{
			name: "invalid exclude pattern",
			rule: StaticRule{
				ID:              "test.invalid_exclude",
				Protocol:        "http",
				Product:         "Test",
				Match:           "test",
				ExcludePatterns: []string{`[invalid`},
			},
			expectedField: "exclude_patterns",
		},
		{
			name: "invalid soft exclude pattern",
			rule: StaticRule{
				ID:                  "test.invalid_soft_exclude",
				Protocol:            "http",
				Product:             "Test",
				Match:               "test",
				SoftExcludePatterns: []string{`[invalid`},
			},
			expectedField: "soft_exclude_patterns",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.Validate([]StaticRule{tc.rule})
			require.False(t, result.IsValid(), "should fail validation")

			found := false
			for _, err := range result.Errors {
				if err.Field == tc.expectedField {
					found = true
					break
				}
			}
			require.True(t, found, "should have error for field: %s", tc.expectedField)
		})
	}
}

func TestValidator_CPEFormat(t *testing.T) {
	validator := NewValidator(false)

	testCases := []struct {
		name        string
		cpe         string
		shouldError bool
	}{
		{
			name:        "valid CPE",
			cpe:         "cpe:2.3:a:vendor:product:*:*:*:*:*:*:*:*",
			shouldError: false,
		},
		{
			name:        "invalid prefix",
			cpe:         "cpe:2.2:a:vendor:product:*:*:*:*:*:*:*:*",
			shouldError: true,
		},
		{
			name:        "missing cpe prefix",
			cpe:         "a:vendor:product:*:*:*:*:*:*:*:*",
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rule := StaticRule{
				ID:       "test.cpe",
				Protocol: "http",
				Product:  "Test",
				Match:    "test",
				CPE:      tc.cpe,
			}

			result := validator.Validate([]StaticRule{rule})

			if tc.shouldError {
				require.False(t, result.IsValid(), "should fail validation for invalid CPE")
			} else {
				require.True(t, result.IsValid(), "should pass validation for valid CPE")
			}
		})
	}
}

func TestValidator_ConfidenceMetadata(t *testing.T) {
	validator := NewValidator(false)

	testCases := []struct {
		name            string
		rule            StaticRule
		shouldError     bool
		expectedMessage string
	}{
		{
			name: "pattern_strength out of range (negative)",
			rule: StaticRule{
				ID:              "test.negative_strength",
				Protocol:        "http",
				Product:         "Test",
				Match:           "test",
				PatternStrength: -0.5,
			},
			shouldError:     true,
			expectedMessage: "pattern_strength must be between 0.0 and 1.0",
		},
		{
			name: "pattern_strength out of range (>1.0)",
			rule: StaticRule{
				ID:              "test.high_strength",
				Protocol:        "http",
				Product:         "Test",
				Match:           "test",
				PatternStrength: 1.5,
			},
			shouldError:     true,
			expectedMessage: "pattern_strength must be between 0.0 and 1.0",
		},
		{
			name: "invalid port number (too high)",
			rule: StaticRule{
				ID:          "test.invalid_port",
				Protocol:    "http",
				Product:     "Test",
				Match:       "test",
				PortBonuses: []int{99999},
			},
			shouldError:     true,
			expectedMessage: "invalid port number",
		},
		{
			name: "invalid port number (zero)",
			rule: StaticRule{
				ID:          "test.zero_port",
				Protocol:    "http",
				Product:     "Test",
				Match:       "test",
				PortBonuses: []int{0},
			},
			shouldError:     true,
			expectedMessage: "invalid port number",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := validator.Validate([]StaticRule{tc.rule})

			if tc.shouldError {
				require.False(t, result.IsValid(), "should fail validation")
				require.NotEmpty(t, result.Errors, "should have errors")
			}
		})
	}
}

func TestValidator_VersionExtractionWithoutCapturingGroup(t *testing.T) {
	validator := NewValidator(false)

	rule := StaticRule{
		ID:                "test.no_capturing_group",
		Protocol:          "http",
		Product:           "Test",
		Match:             "test",
		VersionExtraction: `test\s+\d+\.\d+`, // No capturing group
	}

	result := validator.Validate([]StaticRule{rule})
	require.True(t, result.IsValid(), "should pass validation (warnings don't fail)")
	require.NotEmpty(t, result.Warnings, "should have warnings")

	found := false
	for _, warn := range result.Warnings {
		if warn.Field == "version_extraction" {
			found = true
			break
		}
	}
	require.True(t, found, "should warn about missing capturing group")
}

func TestValidator_RecommendedFields(t *testing.T) {
	validator := NewValidator(false)

	rule := StaticRule{
		ID:       "test.missing_recommended",
		Protocol: "http",
		Product:  "Test",
		Match:    "test",
		// Missing: Vendor, Description, CPE
	}

	result := validator.Validate([]StaticRule{rule})
	require.True(t, result.IsValid(), "missing recommended fields should not fail validation")
	require.NotEmpty(t, result.Warnings, "should have warnings for recommended fields")

	fields := make(map[string]bool)
	for _, warn := range result.Warnings {
		fields[warn.Field] = true
	}

	require.True(t, fields["vendor"], "should warn about missing vendor")
	require.True(t, fields["description"], "should warn about missing description")
	require.True(t, fields["cpe"], "should warn about missing CPE")
}

func TestValidator_StrictMode(t *testing.T) {
	strictValidator := NewValidator(true)

	rule := StaticRule{
		ID:       "test.strict",
		Protocol: "http",
		Product:  "Test",
		Match:    "test",
		// Missing recommended field: Vendor
	}

	result := strictValidator.Validate([]StaticRule{rule})
	// In strict mode, warnings don't automatically fail validation
	// (IsValid only checks errors, not warnings)
	// The caller should check warnings separately in strict mode
	require.True(t, result.IsValid(), "IsValid only checks errors")
	require.NotEmpty(t, result.Warnings, "should have warnings")
}

func TestValidator_MultipleRules(t *testing.T) {
	validator := NewValidator(false)

	rules := []StaticRule{
		{
			ID:       "test.valid1",
			Protocol: "http",
			Product:  "Test1",
			Match:    "test1",
		},
		{
			ID:       "test.valid2",
			Protocol: "ssh",
			Product:  "Test2",
			Match:    "test2",
		},
		{
			ID:       "test.invalid",
			Protocol: "ftp",
			Product:  "Test3",
			Match:    "[invalid",
		},
	}

	result := validator.Validate(rules)
	require.False(t, result.IsValid(), "should fail due to invalid regex")
	require.Equal(t, 3, result.RuleCount, "should count all rules")
	require.NotEmpty(t, result.Errors, "should have errors")

	// Check that error is for the correct rule
	found := false
	for _, err := range result.Errors {
		if err.RuleID == "test.invalid" && err.Field == "match" {
			found = true
			break
		}
	}
	require.True(t, found, "should have error for invalid rule")
}

func TestLoadRulesFromFile(t *testing.T) {
	t.Run("valid YAML file (direct array)", func(t *testing.T) {
		// Create temporary YAML file
		tmpFile, err := os.CreateTemp("", "test-rules-*.yaml")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		yamlContent := `- id: test.rule
  protocol: http
  product: TestProduct
  vendor: TestVendor
  match: "test.*pattern"
  version_extraction: "v(\\d+\\.\\d+)"
  pattern_strength: 0.75
`
		_, err = tmpFile.WriteString(yamlContent)
		require.NoError(t, err)
		tmpFile.Close()

		rules, err := LoadRulesFromFile(tmpFile.Name())
		require.NoError(t, err)
		require.Len(t, rules, 1)
		require.Equal(t, "test.rule", rules[0].ID)
		require.Equal(t, "http", rules[0].Protocol)
		require.Equal(t, "TestProduct", rules[0].Product)
	})

	t.Run("valid YAML file (wrapper format)", func(t *testing.T) {
		// Create temporary YAML file with wrapper format (rules: key)
		tmpFile, err := os.CreateTemp("", "test-rules-wrapper-*.yaml")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		yamlContent := `rules:
  - id: test.rule.wrapper
    protocol: http
    product: TestProduct
    vendor: TestVendor
    match: "test.*pattern"
    version_extraction: "v(\\d+\\.\\d+)"
    pattern_strength: 0.75
`
		_, err = tmpFile.WriteString(yamlContent)
		require.NoError(t, err)
		tmpFile.Close()

		rules, err := LoadRulesFromFile(tmpFile.Name())
		require.NoError(t, err)
		require.Len(t, rules, 1)
		require.Equal(t, "test.rule.wrapper", rules[0].ID)
		require.Equal(t, "http", rules[0].Protocol)
		require.Equal(t, "TestProduct", rules[0].Product)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := LoadRulesFromFile("/nonexistent/file.yaml")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to read file")
	})

	t.Run("invalid YAML", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-invalid-*.yaml")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString("invalid: yaml: content: [")
		require.NoError(t, err)
		tmpFile.Close()

		_, err = LoadRulesFromFile(tmpFile.Name())
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse YAML")
	})

	t.Run("empty file", func(t *testing.T) {
		tmpFile, err := os.CreateTemp("", "test-empty-*.yaml")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		tmpFile.Close()

		rules, err := LoadRulesFromFile(tmpFile.Name())
		require.NoError(t, err)
		require.Empty(t, rules)
	})
}

func TestNewValidationError(t *testing.T) {
	t.Run("with errors only", func(t *testing.T) {
		err := NewValidationError(5, 0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "5 errors")
		require.Contains(t, err.Error(), "0 warnings")
	})

	t.Run("with errors and warnings", func(t *testing.T) {
		err := NewValidationError(3, 7)
		require.Error(t, err)
		require.Contains(t, err.Error(), "3 errors")
		require.Contains(t, err.Error(), "7 warnings")
	})

	t.Run("with no errors", func(t *testing.T) {
		err := NewValidationError(0, 5)
		require.Error(t, err)
		require.Contains(t, err.Error(), "0 errors")
		require.Contains(t, err.Error(), "5 warnings")
	})
}

func TestValidateCPEFormat_MissingCases(t *testing.T) {
	validator := NewValidator(false)

	t.Run("empty CPE field", func(t *testing.T) {
		rule := StaticRule{
			ID:       "test.empty.cpe",
			Protocol: "http",
			Product:  "Test",
			Match:    "test",
			CPE:      "",
		}

		result := validator.Validate([]StaticRule{rule})
		require.True(t, result.IsValid())
		require.NotEmpty(t, result.Warnings)

		// Check for CPE warning
		found := false
		for _, warn := range result.Warnings {
			if warn.Field == "cpe" && warn.RuleID == "test.empty.cpe" {
				found = true
				require.Contains(t, warn.Message, "CPE field is empty")
				break
			}
		}
		require.True(t, found, "should have warning for empty CPE")
	})

	t.Run("valid CPE with 13 components", func(t *testing.T) {
		rule := StaticRule{
			ID:       "test.valid.cpe",
			Protocol: "http",
			Product:  "Test",
			Vendor:   "TestVendor", // Add vendor to avoid warning
			Match:    "test",
			CPE:      "cpe:2.3:a:vendor:product:1.0:*:*:*:*:*:*:*", // Exactly 13 components
		}

		result := validator.Validate([]StaticRule{rule})
		require.True(t, result.IsValid())

		// Should not have warnings for CPE field itself
		for _, warn := range result.Warnings {
			if warn.Field == "cpe" {
				t.Errorf("should not have CPE warning for valid CPE")
			}
		}
	})

	t.Run("CPE with wrong number of components", func(t *testing.T) {
		rule := StaticRule{
			ID:       "test.short.cpe",
			Protocol: "http",
			Product:  "Test",
			Match:    "test",
			CPE:      "cpe:2.3:a:vendor:product", // Only 5 components
		}

		result := validator.Validate([]StaticRule{rule})
		require.True(t, result.IsValid()) // Still valid, just a warning
		require.NotEmpty(t, result.Warnings)

		// Check for component count warning
		found := false
		for _, warn := range result.Warnings {
			if warn.Field == "cpe" && warn.RuleID == "test.short.cpe" {
				found = true
				require.Contains(t, warn.Message, "13 colon-separated components")
				break
			}
		}
		require.True(t, found, "should have warning for wrong component count")
	})
}

func TestValidateConfidenceMetadata_MissingCases(t *testing.T) {
	validator := NewValidator(false)

	t.Run("pattern_strength too low warning", func(t *testing.T) {
		rule := StaticRule{
			ID:              "test.low.strength",
			Protocol:        "http",
			Product:         "Test",
			Match:           "test",
			PatternStrength: 0.30, // Below 0.50 threshold
		}

		result := validator.Validate([]StaticRule{rule})
		require.True(t, result.IsValid())
		require.NotEmpty(t, result.Warnings)

		// Check for low pattern_strength warning
		found := false
		for _, warn := range result.Warnings {
			if warn.Field == "pattern_strength" && warn.RuleID == "test.low.strength" {
				found = true
				require.Contains(t, warn.Message, "below threshold")
				break
			}
		}
		require.True(t, found, "should have warning for low pattern_strength")
	})

	t.Run("pattern_strength not set warning", func(t *testing.T) {
		rule := StaticRule{
			ID:              "test.no.strength",
			Protocol:        "http",
			Product:         "Test",
			Match:           "test",
			PatternStrength: 0, // Not set (defaults to 0.80)
		}

		result := validator.Validate([]StaticRule{rule})
		require.True(t, result.IsValid())
		require.NotEmpty(t, result.Warnings)

		// Check for not set warning
		found := false
		for _, warn := range result.Warnings {
			if warn.Field == "pattern_strength" && warn.RuleID == "test.no.strength" {
				found = true
				require.Contains(t, warn.Message, "not set")
				require.Contains(t, warn.Message, "default to 0.80")
				break
			}
		}
		require.True(t, found, "should have warning for pattern_strength not set")
	})

	t.Run("valid pattern_strength in range", func(t *testing.T) {
		rule := StaticRule{
			ID:              "test.valid.strength",
			Protocol:        "http",
			Product:         "Test",
			Match:           "test",
			PatternStrength: 0.75, // Valid range
		}

		result := validator.Validate([]StaticRule{rule})
		require.True(t, result.IsValid())

		// Should not have pattern_strength warnings for valid value
		for _, warn := range result.Warnings {
			if warn.Field == "pattern_strength" {
				t.Errorf("should not have pattern_strength warning for valid value")
			}
		}
	})
}

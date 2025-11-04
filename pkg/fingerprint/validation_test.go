package fingerprint

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidationRunner(t *testing.T) {
	t.Run("load validation dataset", func(t *testing.T) {
		// Load fingerprint rules
		rules, err := LoadRulesFromFile("data/fingerprint_db.yaml")
		require.NoError(t, err)
		require.NotEmpty(t, rules)

		resolver := NewRuleBasedResolver(rules)

		// Create validation runner
		runner, err := NewValidationRunner(resolver, "testdata/validation_dataset.yaml")
		require.NoError(t, err)
		require.NotNil(t, runner)

		// Verify dataset loaded
		require.NotEmpty(t, runner.dataset.TruePositives)
		require.NotEmpty(t, runner.dataset.TrueNegatives)
	})

	t.Run("run validation and calculate metrics", func(t *testing.T) {
		// Load fingerprint rules
		rules, err := LoadRulesFromFile("data/fingerprint_db.yaml")
		require.NoError(t, err)

		resolver := NewRuleBasedResolver(rules)

		// Create validation runner
		runner, err := NewValidationRunner(resolver, "testdata/validation_dataset.yaml")
		require.NoError(t, err)

		// Run validation
		metrics, results, err := runner.Run(context.Background())
		require.NoError(t, err)
		require.NotNil(t, metrics)
		require.NotEmpty(t, results)

		// Verify metrics structure
		require.Greater(t, metrics.TotalTestCases, 0, "Should have test cases")
		require.GreaterOrEqual(t, metrics.TruePositivesCount, 0)
		require.GreaterOrEqual(t, metrics.TrueNegativesCount, 0)
		require.GreaterOrEqual(t, metrics.FalsePositivesCount, 0)
		require.GreaterOrEqual(t, metrics.FalseNegativesCount, 0)

		// Verify accuracy metrics are calculated
		require.GreaterOrEqual(t, metrics.FalsePositiveRate, 0.0)
		require.LessOrEqual(t, metrics.FalsePositiveRate, 1.0)
		require.GreaterOrEqual(t, metrics.TruePositiveRate, 0.0)
		require.LessOrEqual(t, metrics.TruePositiveRate, 1.0)
		require.GreaterOrEqual(t, metrics.Precision, 0.0)
		require.LessOrEqual(t, metrics.Precision, 1.0)

		// Verify coverage metrics
		require.Greater(t, metrics.ProtocolsCovered, 0, "Should cover multiple protocols")
		require.GreaterOrEqual(t, metrics.VersionExtractedCount, 0)
		require.GreaterOrEqual(t, metrics.VersionAttemptedCount, 0)

		// Verify performance metrics
		require.GreaterOrEqual(t, metrics.AvgDetectionTimeMicros, int64(0), "Should measure detection time")
		require.GreaterOrEqual(t, metrics.AvgDetectionTimeMs, 0.0)

		// Verify pass/fail flags are set
		require.NotNil(t, metrics.PassFPR)
		require.NotNil(t, metrics.PassTPR)
		require.NotNil(t, metrics.PassPrecision)

		// Verify metrics passed count
		require.GreaterOrEqual(t, metrics.MetricsPassed, 0)
		require.LessOrEqual(t, metrics.MetricsPassed, 10) // Max 10 metrics

		// Log metrics for inspection
		t.Logf("Validation Metrics:")
		t.Logf("  Total Test Cases: %d", metrics.TotalTestCases)
		t.Logf("  True Positives: %d", metrics.TruePositivesCount)
		t.Logf("  True Negatives: %d", metrics.TrueNegativesCount)
		t.Logf("  False Positives: %d", metrics.FalsePositivesCount)
		t.Logf("  False Negatives: %d", metrics.FalseNegativesCount)
		t.Logf("  FPR: %.2f%% (target: <%.0f%%)", metrics.FalsePositiveRate*100, metrics.TargetFPR*100)
		t.Logf("  TPR: %.2f%% (target: >%.0f%%)", metrics.TruePositiveRate*100, metrics.TargetTPR*100)
		t.Logf("  Precision: %.2f%% (target: >%.0f%%)", metrics.Precision*100, metrics.TargetPrecision*100)
		t.Logf("  F1 Score: %.4f (target: >%.2f)", metrics.F1Score, metrics.TargetF1)
		t.Logf("  Protocols Covered: %d (target: %d+)", metrics.ProtocolsCovered, metrics.TargetProtocols)
		t.Logf("  Version Extraction Rate: %.2f%% (target: >%.0f%%)", metrics.VersionExtractionRate*100, metrics.TargetVersionRate*100)
		t.Logf("  Avg Detection Time: %.2fms (target: <%.0fms)", metrics.AvgDetectionTimeMs, metrics.TargetPerfMs)
		t.Logf("  Metrics Passed: %d/10", metrics.MetricsPassed)
	})

	t.Run("verify true positive detection", func(t *testing.T) {
		// Test with a simple rule
		rules := []StaticRule{
			{
				ID:              "test.http.apache",
				Protocol:        "http",
				Product:         "Apache",
				Vendor:          "Apache",
				Match:           "apache",
				PatternStrength: 0.90,
			},
		}

		resolver := NewRuleBasedResolver(rules)

		// Create simple test case
		tc := ValidationTestCase{
			Protocol:        "http",
			Port:            80,
			Banner:          "Server: Apache/2.4.41",
			ExpectedProduct: "Apache",
			ExpectedVendor:  "Apache",
			ExpectedVersion: "2.4.41",
		}

		runner := &ValidationRunner{resolver: resolver}
		result := runner.runTestCase(context.Background(), tc, true)

		require.True(t, result.Matched, "Should match Apache")
		require.True(t, result.IsCorrect, "Should be correct match")
		require.Equal(t, "Apache", result.ActualProduct)
		require.Greater(t, result.ActualConfidence, 0.0)
		require.Greater(t, result.DurationMicros, int64(0))
	})

	t.Run("verify true negative detection", func(t *testing.T) {
		// Test with HTTP rule
		rules := []StaticRule{
			{
				ID:              "test.http.apache",
				Protocol:        "http",
				Product:         "Apache",
				Vendor:          "Apache",
				Match:           "apache",
				PatternStrength: 0.90,
			},
		}

		resolver := NewRuleBasedResolver(rules)

		// Test case with wrong banner (should NOT match)
		tc := ValidationTestCase{
			Protocol:      "http",
			Port:          80,
			Banner:        "Server: nginx/1.18.0",
			ExpectedMatch: boolPtr(false),
		}

		runner := &ValidationRunner{resolver: resolver}
		result := runner.runTestCase(context.Background(), tc, false)

		// Should not match (nginx banner on Apache rule)
		require.False(t, result.Matched || result.ActualProduct == "Apache", "Should not match Apache")
		require.True(t, result.IsCorrect, "Should be correct non-match")
	})

	t.Run("verify version extraction tracking", func(t *testing.T) {
		rules := []StaticRule{
			{
				ID:                "test.http.apache",
				Protocol:          "http",
				Product:           "Apache",
				Vendor:            "Apache",
				Match:             "apache",
				VersionExtraction: `apache/(\d+\.\d+\.\d+)`,
				PatternStrength:   0.90,
			},
		}

		resolver := NewRuleBasedResolver(rules)

		// Test case with version
		tc := ValidationTestCase{
			Protocol:        "http",
			Port:            80,
			Banner:          "Server: Apache/2.4.41",
			ExpectedProduct: "Apache",
			ExpectedVersion: "2.4.41",
		}

		runner := &ValidationRunner{resolver: resolver}
		result := runner.runTestCase(context.Background(), tc, true)

		require.True(t, result.Matched)
		require.True(t, result.VersionExtracted, "Should extract version")
		require.Equal(t, "2.4.41", result.ActualVersion)
	})
}

func TestValidationMetricsCalculation(t *testing.T) {
	t.Run("calculate metrics with perfect accuracy", func(t *testing.T) {
		runner := &ValidationRunner{}
		results := []ValidationResult{
			// 5 true positives
			{Matched: true, IsCorrect: true, ActualConfidence: 0.95, DurationMicros: 100, TestCase: ValidationTestCase{Protocol: "http", ExpectedVersion: "1.0"}, VersionExtracted: true},
			{Matched: true, IsCorrect: true, ActualConfidence: 0.90, DurationMicros: 150, TestCase: ValidationTestCase{Protocol: "ssh", ExpectedVersion: "2.0"}, VersionExtracted: true},
			{Matched: true, IsCorrect: true, ActualConfidence: 0.85, DurationMicros: 120, TestCase: ValidationTestCase{Protocol: "ftp"}},
			{Matched: true, IsCorrect: true, ActualConfidence: 0.92, DurationMicros: 110, TestCase: ValidationTestCase{Protocol: "mysql"}},
			{Matched: true, IsCorrect: true, ActualConfidence: 0.88, DurationMicros: 130, TestCase: ValidationTestCase{Protocol: "smtp"}},
			// 3 true negatives
			{Matched: false, IsCorrect: true, DurationMicros: 50, TestCase: ValidationTestCase{Protocol: "http", ExpectedMatch: boolPtr(false)}},
			{Matched: false, IsCorrect: true, DurationMicros: 60, TestCase: ValidationTestCase{Protocol: "ssh", ExpectedMatch: boolPtr(false)}},
			{Matched: false, IsCorrect: true, DurationMicros: 55, TestCase: ValidationTestCase{Protocol: "ftp", ExpectedMatch: boolPtr(false)}},
		}

		metrics := runner.calculateMetrics(results)

		// Perfect accuracy: 5 TP, 3 TN, 0 FP, 0 FN
		require.Equal(t, 8, metrics.TotalTestCases)
		require.Equal(t, 5, metrics.TruePositivesCount)
		require.Equal(t, 3, metrics.TrueNegativesCount)
		require.Equal(t, 0, metrics.FalsePositivesCount)
		require.Equal(t, 0, metrics.FalseNegativesCount)

		// Metrics should be perfect
		require.Equal(t, 0.0, metrics.FalsePositiveRate) // 0 / (0 + 3)
		require.Equal(t, 1.0, metrics.TruePositiveRate)  // 5 / (5 + 0)
		require.Equal(t, 1.0, metrics.Precision)         // 5 / (5 + 0)
		require.Equal(t, 1.0, metrics.F1Score)           // Perfect F1

		// Version extraction: 2 extracted / 2 attempted
		require.Equal(t, 2, metrics.VersionExtractedCount)
		require.Equal(t, 2, metrics.VersionAttemptedCount)
		require.Equal(t, 1.0, metrics.VersionExtractionRate)

		// Protocols: 5 unique
		require.Equal(t, 5, metrics.ProtocolsCovered)

		// Performance: avg of (100+150+120+110+130+50+60+55) / 8 = 96.875 µs
		require.InDelta(t, 96.875, float64(metrics.AvgDetectionTimeMicros), 1.0)
	})

	t.Run("calculate metrics with some errors", func(t *testing.T) {
		runner := &ValidationRunner{}
		results := []ValidationResult{
			// 3 true positives
			{Matched: true, IsCorrect: true, ActualConfidence: 0.95, DurationMicros: 100, TestCase: ValidationTestCase{Protocol: "http"}},
			{Matched: true, IsCorrect: true, ActualConfidence: 0.90, DurationMicros: 120, TestCase: ValidationTestCase{Protocol: "ssh"}},
			{Matched: true, IsCorrect: true, ActualConfidence: 0.85, DurationMicros: 110, TestCase: ValidationTestCase{Protocol: "ftp"}},
			// 1 false positive (matched but shouldn't have)
			{Matched: true, IsCorrect: false, ActualConfidence: 0.70, DurationMicros: 150, TestCase: ValidationTestCase{Protocol: "http", ExpectedMatch: boolPtr(false)}},
			// 1 false negative (should match but didn't)
			{Matched: false, IsCorrect: false, DurationMicros: 80, TestCase: ValidationTestCase{Protocol: "mysql"}},
			// 2 true negatives
			{Matched: false, IsCorrect: true, DurationMicros: 50, TestCase: ValidationTestCase{Protocol: "ssh", ExpectedMatch: boolPtr(false)}},
			{Matched: false, IsCorrect: true, DurationMicros: 60, TestCase: ValidationTestCase{Protocol: "ftp", ExpectedMatch: boolPtr(false)}},
		}

		metrics := runner.calculateMetrics(results)

		// Counts: 3 TP, 2 TN, 1 FP, 1 FN
		require.Equal(t, 7, metrics.TotalTestCases)
		require.Equal(t, 3, metrics.TruePositivesCount)
		require.Equal(t, 2, metrics.TrueNegativesCount)
		require.Equal(t, 1, metrics.FalsePositivesCount)
		require.Equal(t, 1, metrics.FalseNegativesCount)

		// FPR = 1 / (1 + 2) = 0.333
		require.InDelta(t, 0.333, metrics.FalsePositiveRate, 0.01)

		// TPR = 3 / (3 + 1) = 0.75
		require.InDelta(t, 0.75, metrics.TruePositiveRate, 0.01)

		// Precision = 3 / (3 + 1) = 0.75
		require.InDelta(t, 0.75, metrics.Precision, 0.01)

		// F1 = 2 * (0.75 * 0.75) / (0.75 + 0.75) = 0.75
		require.InDelta(t, 0.75, metrics.F1Score, 0.01)

		// Protocols: 4 unique (http, ssh, ftp, mysql)
		require.Equal(t, 4, metrics.ProtocolsCovered)
	})
}

func boolPtr(b bool) *bool {
	return &b
}

func TestValidationMetricsEdgeCases(t *testing.T) {
	t.Run("all true positives (no negatives)", func(t *testing.T) {
		runner := &ValidationRunner{}
		results := []ValidationResult{
			{Matched: true, IsCorrect: true, ActualConfidence: 0.95, DurationMicros: 100, TestCase: ValidationTestCase{Protocol: "http"}},
			{Matched: true, IsCorrect: true, ActualConfidence: 0.90, DurationMicros: 150, TestCase: ValidationTestCase{Protocol: "ssh"}},
			{Matched: true, IsCorrect: true, ActualConfidence: 0.85, DurationMicros: 120, TestCase: ValidationTestCase{Protocol: "ftp"}},
		}

		metrics := runner.calculateMetrics(results)

		// Only TPs, no FP/TN/FN
		require.Equal(t, 3, metrics.TotalTestCases)
		require.Equal(t, 3, metrics.TruePositivesCount)
		require.Equal(t, 0, metrics.TrueNegativesCount)
		require.Equal(t, 0, metrics.FalsePositivesCount)
		require.Equal(t, 0, metrics.FalseNegativesCount)

		// FPR undefined (no negatives), should be 0
		require.Equal(t, 0.0, metrics.FalsePositiveRate)

		// TPR = 3 / (3 + 0) = 1.0
		require.Equal(t, 1.0, metrics.TruePositiveRate)

		// Precision = 3 / (3 + 0) = 1.0
		require.Equal(t, 1.0, metrics.Precision)

		// F1 = 1.0
		require.Equal(t, 1.0, metrics.F1Score)
	})

	t.Run("all false positives (all matches wrong)", func(t *testing.T) {
		runner := &ValidationRunner{}
		results := []ValidationResult{
			// All matched but wrong product
			{Matched: true, IsCorrect: false, ActualConfidence: 0.70, DurationMicros: 100, TestCase: ValidationTestCase{Protocol: "http"}},
			{Matched: true, IsCorrect: false, ActualConfidence: 0.65, DurationMicros: 150, TestCase: ValidationTestCase{Protocol: "ssh"}},
			{Matched: true, IsCorrect: false, ActualConfidence: 0.60, DurationMicros: 120, TestCase: ValidationTestCase{Protocol: "ftp"}},
		}

		metrics := runner.calculateMetrics(results)

		// All FP
		require.Equal(t, 3, metrics.TotalTestCases)
		require.Equal(t, 0, metrics.TruePositivesCount)
		require.Equal(t, 0, metrics.FalseNegativesCount)
		require.Equal(t, 3, metrics.FalsePositivesCount)

		// TPR = 0 / (0 + 0) = 0 (no positives expected)
		require.Equal(t, 0.0, metrics.TruePositiveRate)

		// Precision = 0 / (0 + 3) = 0
		require.Equal(t, 0.0, metrics.Precision)

		// F1 = 0 (when P or R is 0)
		require.Equal(t, 0.0, metrics.F1Score)
	})

	t.Run("no predictions (all no-match)", func(t *testing.T) {
		runner := &ValidationRunner{}
		results := []ValidationResult{
			// Expected to match but didn't
			{Matched: false, IsCorrect: false, DurationMicros: 50, TestCase: ValidationTestCase{Protocol: "http"}},
			{Matched: false, IsCorrect: false, DurationMicros: 60, TestCase: ValidationTestCase{Protocol: "ssh"}},
			{Matched: false, IsCorrect: false, DurationMicros: 55, TestCase: ValidationTestCase{Protocol: "ftp"}},
		}

		metrics := runner.calculateMetrics(results)

		// All FN (expected to match but didn't)
		require.Equal(t, 3, metrics.TotalTestCases)
		require.Equal(t, 0, metrics.TruePositivesCount)
		require.Equal(t, 3, metrics.FalseNegativesCount)
		require.Equal(t, 0, metrics.FalsePositivesCount)

		// TPR = 0 / (0 + 3) = 0
		require.Equal(t, 0.0, metrics.TruePositiveRate)

		// Precision undefined (no predictions), should be 0
		require.Equal(t, 0.0, metrics.Precision)

		// F1 = 0
		require.Equal(t, 0.0, metrics.F1Score)
	})

	t.Run("only true negatives", func(t *testing.T) {
		runner := &ValidationRunner{}
		results := []ValidationResult{
			// All correctly rejected (should NOT match)
			{Matched: false, IsCorrect: true, DurationMicros: 50, TestCase: ValidationTestCase{Protocol: "http", ExpectedMatch: boolPtr(false)}},
			{Matched: false, IsCorrect: true, DurationMicros: 60, TestCase: ValidationTestCase{Protocol: "ssh", ExpectedMatch: boolPtr(false)}},
			{Matched: false, IsCorrect: true, DurationMicros: 55, TestCase: ValidationTestCase{Protocol: "ftp", ExpectedMatch: boolPtr(false)}},
		}

		metrics := runner.calculateMetrics(results)

		// Only TNs
		require.Equal(t, 3, metrics.TotalTestCases)
		require.Equal(t, 0, metrics.TruePositivesCount)
		require.Equal(t, 3, metrics.TrueNegativesCount)
		require.Equal(t, 0, metrics.FalsePositivesCount)
		require.Equal(t, 0, metrics.FalseNegativesCount)

		// FPR = 0 / (0 + 3) = 0
		require.Equal(t, 0.0, metrics.FalsePositiveRate)

		// TPR undefined (no positives expected), should be 0
		require.Equal(t, 0.0, metrics.TruePositiveRate)

		// Precision undefined (no predictions), should be 0
		require.Equal(t, 0.0, metrics.Precision)

		// F1 = 0
		require.Equal(t, 0.0, metrics.F1Score)
	})

	t.Run("mixed with zero version extraction", func(t *testing.T) {
		runner := &ValidationRunner{}
		results := []ValidationResult{
			// TPs without version extraction
			{Matched: true, IsCorrect: true, ActualConfidence: 0.95, DurationMicros: 100, TestCase: ValidationTestCase{Protocol: "http"}, VersionExtracted: false},
			{Matched: true, IsCorrect: true, ActualConfidence: 0.90, DurationMicros: 150, TestCase: ValidationTestCase{Protocol: "ssh", ExpectedVersion: "8.2"}, VersionExtracted: false},
			// TNs
			{Matched: false, IsCorrect: true, DurationMicros: 50, TestCase: ValidationTestCase{Protocol: "mysql", ExpectedMatch: boolPtr(false)}},
		}

		metrics := runner.calculateMetrics(results)

		// Version extraction: 0 extracted / 1 attempted = 0%
		require.Equal(t, 0, metrics.VersionExtractedCount)
		require.Equal(t, 1, metrics.VersionAttemptedCount)
		require.Equal(t, 0.0, metrics.VersionExtractionRate)

		// Regular metrics
		require.Equal(t, 2, metrics.TruePositivesCount)
		require.Equal(t, 1, metrics.TrueNegativesCount)
	})

	t.Run("empty results", func(t *testing.T) {
		runner := &ValidationRunner{}
		results := []ValidationResult{}

		metrics := runner.calculateMetrics(results)

		// All zeros
		require.Equal(t, 0, metrics.TotalTestCases)
		require.Equal(t, 0, metrics.TruePositivesCount)
		require.Equal(t, 0, metrics.TrueNegativesCount)
		require.Equal(t, 0, metrics.FalsePositivesCount)
		require.Equal(t, 0, metrics.FalseNegativesCount)

		// All rates should be 0
		require.Equal(t, 0.0, metrics.FalsePositiveRate)
		require.Equal(t, 0.0, metrics.TruePositiveRate)
		require.Equal(t, 0.0, metrics.Precision)
		require.Equal(t, 0.0, metrics.F1Score)
		require.Equal(t, 0.0, metrics.VersionExtractionRate)

		// No protocols covered
		require.Equal(t, 0, metrics.ProtocolsCovered)
	})
}

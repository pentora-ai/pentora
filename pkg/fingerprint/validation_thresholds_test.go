package fingerprint

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultThresholds(t *testing.T) {
	thresholds := DefaultThresholds()

	require.Equal(t, 0.10, thresholds.TargetFPR)
	require.Equal(t, 0.80, thresholds.TargetTPR)
	require.Equal(t, 0.85, thresholds.TargetPrecision)
	require.Equal(t, 0.82, thresholds.TargetF1)
	require.Equal(t, 20, thresholds.TargetProtocols)
	require.Equal(t, 0.70, thresholds.TargetVersionRate)
	require.Equal(t, 50.0, thresholds.TargetPerfMs)
}

func TestStrictThresholds(t *testing.T) {
	thresholds := StrictThresholds()

	// Strict profile should have more demanding thresholds
	require.Less(t, thresholds.TargetFPR, DefaultThresholds().TargetFPR, "Strict FPR should be lower")
	require.Greater(t, thresholds.TargetTPR, DefaultThresholds().TargetTPR, "Strict TPR should be higher")
	require.Greater(t, thresholds.TargetPrecision, DefaultThresholds().TargetPrecision, "Strict Precision should be higher")
	require.Greater(t, thresholds.TargetF1, DefaultThresholds().TargetF1, "Strict F1 should be higher")
	require.Greater(t, thresholds.TargetProtocols, DefaultThresholds().TargetProtocols, "Strict protocols should be higher")
	require.Greater(t, thresholds.TargetVersionRate, DefaultThresholds().TargetVersionRate, "Strict version rate should be higher")
	require.Less(t, thresholds.TargetPerfMs, DefaultThresholds().TargetPerfMs, "Strict performance should be faster")
}

func TestRelaxedThresholds(t *testing.T) {
	thresholds := RelaxedThresholds()

	// Relaxed profile should have more permissive thresholds
	require.Greater(t, thresholds.TargetFPR, DefaultThresholds().TargetFPR, "Relaxed FPR should be higher")
	require.Less(t, thresholds.TargetTPR, DefaultThresholds().TargetTPR, "Relaxed TPR should be lower")
	require.Less(t, thresholds.TargetPrecision, DefaultThresholds().TargetPrecision, "Relaxed Precision should be lower")
	require.Less(t, thresholds.TargetF1, DefaultThresholds().TargetF1, "Relaxed F1 should be lower")
	require.Less(t, thresholds.TargetProtocols, DefaultThresholds().TargetProtocols, "Relaxed protocols should be lower")
	require.Less(t, thresholds.TargetVersionRate, DefaultThresholds().TargetVersionRate, "Relaxed version rate should be lower")
	require.Greater(t, thresholds.TargetPerfMs, DefaultThresholds().TargetPerfMs, "Relaxed performance can be slower")
}

func TestLoadThresholdsFromEnv(t *testing.T) {
	t.Run("default when no env vars", func(t *testing.T) {
		// Clear all env vars
		os.Unsetenv("PENTORA_VALIDATION_TARGET_FPR")
		os.Unsetenv("PENTORA_VALIDATION_TARGET_TPR")
		os.Unsetenv("PENTORA_VALIDATION_TARGET_PRECISION")
		os.Unsetenv("PENTORA_VALIDATION_TARGET_F1")
		os.Unsetenv("PENTORA_VALIDATION_TARGET_PROTOCOLS")
		os.Unsetenv("PENTORA_VALIDATION_TARGET_VERSION_RATE")
		os.Unsetenv("PENTORA_VALIDATION_TARGET_PERF_MS")

		thresholds := LoadThresholdsFromEnv()
		defaults := DefaultThresholds()

		require.Equal(t, defaults, thresholds)
	})

	t.Run("override individual thresholds", func(t *testing.T) {
		// Set custom env vars
		os.Setenv("PENTORA_VALIDATION_TARGET_FPR", "0.05")
		os.Setenv("PENTORA_VALIDATION_TARGET_TPR", "0.90")
		os.Setenv("PENTORA_VALIDATION_TARGET_PROTOCOLS", "25")
		defer func() {
			os.Unsetenv("PENTORA_VALIDATION_TARGET_FPR")
			os.Unsetenv("PENTORA_VALIDATION_TARGET_TPR")
			os.Unsetenv("PENTORA_VALIDATION_TARGET_PROTOCOLS")
		}()

		thresholds := LoadThresholdsFromEnv()

		require.Equal(t, 0.05, thresholds.TargetFPR)
		require.Equal(t, 0.90, thresholds.TargetTPR)
		require.Equal(t, 25, thresholds.TargetProtocols)
		// Other thresholds should be default
		require.Equal(t, DefaultThresholds().TargetPrecision, thresholds.TargetPrecision)
		require.Equal(t, DefaultThresholds().TargetF1, thresholds.TargetF1)
	})

	t.Run("ignore invalid values", func(t *testing.T) {
		// Set invalid env vars
		os.Setenv("PENTORA_VALIDATION_TARGET_FPR", "invalid")
		os.Setenv("PENTORA_VALIDATION_TARGET_TPR", "-1.5")
		os.Setenv("PENTORA_VALIDATION_TARGET_PROTOCOLS", "not_a_number")
		defer func() {
			os.Unsetenv("PENTORA_VALIDATION_TARGET_FPR")
			os.Unsetenv("PENTORA_VALIDATION_TARGET_TPR")
			os.Unsetenv("PENTORA_VALIDATION_TARGET_PROTOCOLS")
		}()

		thresholds := LoadThresholdsFromEnv()

		// Should fall back to defaults when values are invalid
		require.Equal(t, DefaultThresholds().TargetFPR, thresholds.TargetFPR)
		require.Equal(t, DefaultThresholds().TargetTPR, thresholds.TargetTPR)
		require.Equal(t, DefaultThresholds().TargetProtocols, thresholds.TargetProtocols)
	})

	t.Run("validate range constraints", func(t *testing.T) {
		// FPR > 1.0 should be ignored
		os.Setenv("PENTORA_VALIDATION_TARGET_FPR", "1.5")
		defer os.Unsetenv("PENTORA_VALIDATION_TARGET_FPR")

		thresholds := LoadThresholdsFromEnv()
		require.Equal(t, DefaultThresholds().TargetFPR, thresholds.TargetFPR, "Should ignore out-of-range value")
	})
}

func TestThresholdsJSONExport(t *testing.T) {
	t.Run("export default thresholds", func(t *testing.T) {
		thresholds := DefaultThresholds()

		jsonData, err := thresholds.ToJSON()
		require.NoError(t, err)
		require.NotEmpty(t, jsonData)

		// Verify it's valid JSON
		var parsed ValidationThresholds
		err = json.Unmarshal(jsonData, &parsed)
		require.NoError(t, err)
		require.Equal(t, thresholds, parsed)
	})

	t.Run("export strict thresholds", func(t *testing.T) {
		thresholds := StrictThresholds()

		jsonData, err := thresholds.ToJSON()
		require.NoError(t, err)

		// Verify structure
		var parsed ValidationThresholds
		err = json.Unmarshal(jsonData, &parsed)
		require.NoError(t, err)
		require.Equal(t, 0.05, parsed.TargetFPR)
		require.Equal(t, 0.90, parsed.TargetTPR)
	})
}

func TestValidationMetricsJSONExport(t *testing.T) {
	t.Run("export metrics with all fields", func(t *testing.T) {
		metrics := &ValidationMetrics{
			TotalTestCases:      130,
			TruePositivesCount:  47,
			TrueNegativesCount:  29,
			FalsePositivesCount: 9,
			FalseNegativesCount: 45,
			FalsePositiveRate:   0.2368,
			TruePositiveRate:    0.5109,
			Precision:           0.8393,
			F1Score:             0.6351,
			ProtocolsCovered:    16,
			TargetFPR:           0.10,
			TargetTPR:           0.80,
			PassFPR:             false,
			PassTPR:             false,
			MetricsPassed:       1,
		}

		jsonData, err := metrics.ToJSON()
		require.NoError(t, err)
		require.NotEmpty(t, jsonData)

		// Verify it's valid JSON
		var parsed ValidationMetrics
		err = json.Unmarshal(jsonData, &parsed)
		require.NoError(t, err)

		// Check key fields
		require.Equal(t, metrics.TotalTestCases, parsed.TotalTestCases)
		require.Equal(t, metrics.FalsePositiveRate, parsed.FalsePositiveRate)
		require.Equal(t, metrics.TruePositiveRate, parsed.TruePositiveRate)
		require.Equal(t, metrics.PassFPR, parsed.PassFPR)
		require.Equal(t, metrics.MetricsPassed, parsed.MetricsPassed)
	})

	t.Run("export empty metrics", func(t *testing.T) {
		metrics := &ValidationMetrics{}

		jsonData, err := metrics.ToJSON()
		require.NoError(t, err)
		require.NotEmpty(t, jsonData)

		// Should still be valid JSON
		var parsed ValidationMetrics
		err = json.Unmarshal(jsonData, &parsed)
		require.NoError(t, err)
	})
}

func TestValidationRunnerWithCustomThresholds(t *testing.T) {
	t.Run("use default thresholds", func(t *testing.T) {
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
		runner, err := NewValidationRunner(resolver, "testdata/validation_dataset.yaml")
		require.NoError(t, err)
		require.NotNil(t, runner)

		// Verify default thresholds are used
		require.Equal(t, DefaultThresholds(), runner.Thresholds())
	})

	t.Run("use strict thresholds", func(t *testing.T) {
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
		runner, err := NewValidationRunnerWithThresholds(resolver, "testdata/validation_dataset.yaml", StrictThresholds())
		require.NoError(t, err)
		require.NotNil(t, runner)

		// Verify strict thresholds are used
		require.Equal(t, StrictThresholds(), runner.Thresholds())
	})

	t.Run("metrics use custom thresholds", func(t *testing.T) {
		rules, err := LoadRulesFromFile("data/fingerprint_db.yaml")
		require.NoError(t, err)

		resolver := NewRuleBasedResolver(rules)

		// Use strict thresholds
		runner, err := NewValidationRunnerWithThresholds(resolver, "testdata/validation_dataset.yaml", StrictThresholds())
		require.NoError(t, err)

		metrics, _, err := runner.Run(context.Background())
		require.NoError(t, err)

		// Verify metrics use strict thresholds
		require.Equal(t, 0.05, metrics.TargetFPR, "Should use strict FPR threshold")
		require.Equal(t, 0.90, metrics.TargetTPR, "Should use strict TPR threshold")
		require.Equal(t, 0.92, metrics.TargetPrecision, "Should use strict Precision threshold")
	})

	t.Run("env variables override defaults", func(t *testing.T) {
		// Set custom thresholds via env
		os.Setenv("PENTORA_VALIDATION_TARGET_FPR", "0.05")
		os.Setenv("PENTORA_VALIDATION_TARGET_TPR", "0.90")
		defer func() {
			os.Unsetenv("PENTORA_VALIDATION_TARGET_FPR")
			os.Unsetenv("PENTORA_VALIDATION_TARGET_TPR")
		}()

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
		thresholds := LoadThresholdsFromEnv()
		runner, err := NewValidationRunnerWithThresholds(resolver, "testdata/validation_dataset.yaml", thresholds)
		require.NoError(t, err)

		// Verify env thresholds are used
		require.Equal(t, 0.05, runner.thresholds.TargetFPR)
		require.Equal(t, 0.90, runner.thresholds.TargetTPR)
	})
}

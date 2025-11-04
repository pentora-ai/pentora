package fingerprint

import (
	"context"
	"time"
)

// ValidationDataset represents the structure of the validation_dataset.yaml file.
// moved to validation_dataset.go

// ValidationResult represents the outcome of validating a single test case.
type ValidationResult struct {
	TestCase         ValidationTestCase
	ActualProduct    string
	ActualVendor     string
	ActualVersion    string
	ActualConfidence float64
	Matched          bool
	Error            error
	IsCorrect        bool // True if result matches expectation
	VersionExtracted bool // True if version was successfully extracted
	DurationMicros   int64
}

// ValidationMetrics represents aggregated metrics from validation run.
type ValidationMetrics struct {
	// Overall counts
	TotalTestCases      int `json:"total_test_cases"`
	TruePositivesCount  int `json:"true_positives_count"`
	TrueNegativesCount  int `json:"true_negatives_count"`
	FalsePositivesCount int `json:"false_positives_count"`
	FalseNegativesCount int `json:"false_negatives_count"`
	EdgeCasesCount      int `json:"edge_cases_count"`

	// Accuracy metrics
	FalsePositiveRate float64 `json:"false_positive_rate"` // FP / (FP + TN)
	TruePositiveRate  float64 `json:"true_positive_rate"`  // TP / (TP + FN)
	Precision         float64 `json:"precision"`           // TP / (TP + FP)
	Recall            float64 `json:"recall"`              // Same as TPR
	F1Score           float64 `json:"f1_score"`            // 2 * (P * R) / (P + R)

	// Coverage metrics
	ProtocolsCovered      int     `json:"protocols_covered"`
	VersionExtractedCount int     `json:"version_extracted_count"`
	VersionAttemptedCount int     `json:"version_attempted_count"`
	VersionExtractionRate float64 `json:"version_extraction_rate"` // Extracted / Attempted

	// Confidence metrics
	ConfidenceMean   float64 `json:"confidence_mean"`
	ConfidenceMedian float64 `json:"confidence_median"`
	ConfidenceMin    float64 `json:"confidence_min"`
	ConfidenceMax    float64 `json:"confidence_max"`

	// Performance metrics
	AvgDetectionTimeMicros int64   `json:"avg_detection_time_micros"`
	AvgDetectionTimeMs     float64 `json:"avg_detection_time_ms"`

	// Pass/fail for each target
	TargetFPR         float64 `json:"target_fpr"`          // <10%
	TargetTPR         float64 `json:"target_tpr"`          // >80%
	TargetPrecision   float64 `json:"target_precision"`    // >85%
	TargetF1          float64 `json:"target_f1"`           // >0.82
	TargetProtocols   int     `json:"target_protocols"`    // 20+
	TargetVersionRate float64 `json:"target_version_rate"` // >70%
	TargetPerfMs      float64 `json:"target_perf_ms"`      // <50ms

	PassFPR         bool `json:"pass_fpr"`
	PassTPR         bool `json:"pass_tpr"`
	PassPrecision   bool `json:"pass_precision"`
	PassF1          bool `json:"pass_f1"`
	PassProtocols   bool `json:"pass_protocols"`
	PassVersionRate bool `json:"pass_version_rate"`
	PassPerformance bool `json:"pass_performance"`

	MetricsPassed int `json:"metrics_passed"` // Count of passed metrics (out of 10)
}

// ValidationRunner executes validation tests and computes metrics.
type ValidationRunner struct {
	resolver   Resolver
	dataset    *ValidationDataset
	thresholds ValidationThresholds
}

// calculateMetrics is deprecated; kept for test compatibility. It forwards to
// CalculateMetrics in validation_metrics.go with default targets.
func (vr *ValidationRunner) calculateMetrics(results []ValidationResult) *ValidationMetrics {
	targets := ValidationMetrics{
		TargetFPR:         0.10,
		TargetTPR:         0.80,
		TargetPrecision:   0.85,
		TargetF1:          0.82,
		TargetProtocols:   20,
		TargetVersionRate: 0.70,
		TargetPerfMs:      50.0,
	}
	return CalculateMetrics(results, targets)
}

// NewValidationRunner creates a new validation runner with the given resolver and default thresholds.
// For custom thresholds, use NewValidationRunnerWithThresholds.
func NewValidationRunner(resolver Resolver, datasetPath string) (*ValidationRunner, error) {
	return NewValidationRunnerWithThresholds(resolver, datasetPath, DefaultThresholds())
}

// NewValidationRunnerWithThresholds creates a new validation runner with custom thresholds.
// This allows fine-tuning validation criteria per use case (e.g., strict vs relaxed profiles).
func NewValidationRunnerWithThresholds(resolver Resolver, datasetPath string, thresholds ValidationThresholds) (*ValidationRunner, error) {
	dataset, err := LoadValidationDataset(datasetPath)
	if err != nil {
		return nil, err
	}
	return &ValidationRunner{
		resolver:   resolver,
		dataset:    dataset,
		thresholds: thresholds,
	}, nil
}

// Run executes all validation tests and returns aggregated metrics.
func (vr *ValidationRunner) Run(ctx context.Context) (*ValidationMetrics, []ValidationResult, error) {
	var results []ValidationResult

	// Run true positive tests
	for _, tc := range vr.dataset.TruePositives {
		result := vr.runTestCase(ctx, tc, true)
		results = append(results, result)
	}

	// Run true negative tests
	for _, tc := range vr.dataset.TrueNegatives {
		result := vr.runTestCase(ctx, tc, false)
		results = append(results, result)
	}

	// Run edge case tests
	for _, tc := range vr.dataset.EdgeCases {
		result := vr.runTestCase(ctx, tc, true) // Edge cases should match
		results = append(results, result)
	}

	// Calculate metrics using configured thresholds
	targets := ValidationMetrics{
		TargetFPR:         vr.thresholds.TargetFPR,
		TargetTPR:         vr.thresholds.TargetTPR,
		TargetPrecision:   vr.thresholds.TargetPrecision,
		TargetF1:          vr.thresholds.TargetF1,
		TargetProtocols:   vr.thresholds.TargetProtocols,
		TargetVersionRate: vr.thresholds.TargetVersionRate,
		TargetPerfMs:      vr.thresholds.TargetPerfMs,
	}
	metrics := CalculateMetrics(results, targets)

	return metrics, results, nil
}

// runTestCase executes a single test case and returns the result.
func (vr *ValidationRunner) runTestCase(ctx context.Context, tc ValidationTestCase, shouldMatch bool) ValidationResult {
	result := ValidationResult{
		TestCase: tc,
	}

	// Measure detection time
	start := time.Now()

	// Run resolver
	input := Input{
		Port:     tc.Port,
		Protocol: tc.Protocol,
		Banner:   tc.Banner,
	}

	resolverResult, err := vr.resolver.Resolve(ctx, input)
	duration := time.Since(start)
	result.DurationMicros = duration.Microseconds()

	if err != nil {
		// No match
		result.Matched = false
		result.Error = err
		result.IsCorrect = !shouldMatch // Correct if we expected no match
	} else {
		// Match found
		result.Matched = true
		result.ActualProduct = resolverResult.Product
		result.ActualVendor = resolverResult.Vendor
		result.ActualVersion = resolverResult.Version
		result.ActualConfidence = resolverResult.Confidence

		// Check if result is correct
		if shouldMatch {
			// True positive: check if product matches
			result.IsCorrect = (result.ActualProduct == tc.ExpectedProduct)

			// Version extraction check
			if tc.ExpectedVersion != "" {
				result.VersionExtracted = (result.ActualVersion != "")
			}
		} else {
			// True negative: match found but shouldn't have matched
			result.IsCorrect = false
		}
	}

	// For true negatives with explicit expected_match: false
	if tc.ExpectedMatch != nil && !*tc.ExpectedMatch {
		result.IsCorrect = !result.Matched
	}

	return result
}

// calculateMetrics computes all validation metrics from test results.
//
//nolint:funlen,gocyclo // Metric calculation is inherently complex
// calculateMetrics moved to validation_metrics.go (CalculateMetrics)

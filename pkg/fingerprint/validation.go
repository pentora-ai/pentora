package fingerprint

import (
	"context"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// ValidationDataset represents the structure of the validation_dataset.yaml file.
type ValidationDataset struct {
	TruePositives []ValidationTestCase `yaml:"true_positives"`
	TrueNegatives []ValidationTestCase `yaml:"true_negatives"`
	EdgeCases     []ValidationTestCase `yaml:"edge_cases"`
}

// ValidationTestCase represents a single test case in the validation dataset.
type ValidationTestCase struct {
	Protocol        string `yaml:"protocol"`
	Port            int    `yaml:"port"`
	Banner          string `yaml:"banner"`
	ExpectedProduct string `yaml:"expected_product,omitempty"`
	ExpectedVendor  string `yaml:"expected_vendor,omitempty"`
	ExpectedVersion string `yaml:"expected_version,omitempty"`
	ExpectedMatch   *bool  `yaml:"expected_match,omitempty"` // For true negatives
	Description     string `yaml:"description"`
}

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
	resolver Resolver
	dataset  *ValidationDataset
}

// NewValidationRunner creates a new validation runner with the given resolver.
func NewValidationRunner(resolver Resolver, datasetPath string) (*ValidationRunner, error) {
	data, err := os.ReadFile(datasetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read validation dataset: %w", err)
	}

	var dataset ValidationDataset
	if err := yaml.Unmarshal(data, &dataset); err != nil {
		return nil, fmt.Errorf("failed to parse validation dataset: %w", err)
	}

	return &ValidationRunner{
		resolver: resolver,
		dataset:  &dataset,
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

	// Calculate metrics
	metrics := vr.calculateMetrics(results)

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
func (vr *ValidationRunner) calculateMetrics(results []ValidationResult) *ValidationMetrics {
	metrics := &ValidationMetrics{
		TotalTestCases:    len(results),
		TargetFPR:         0.10, // <10%
		TargetTPR:         0.80, // >80%
		TargetPrecision:   0.85, // >85%
		TargetF1:          0.82, // >0.82
		TargetProtocols:   20,   // 20+
		TargetVersionRate: 0.70, // >70%
		TargetPerfMs:      50.0, // <50ms
	}

	var (
		truePositives       int
		trueNegatives       int
		falsePositives      int
		falseNegatives      int
		versionExtracted    int
		versionAttempted    int
		confidenceScores    []float64
		totalDurationMicros int64
		protocols           = make(map[string]bool)
	)

	for _, result := range results {
		// Track protocols
		protocols[result.TestCase.Protocol] = true

		// Track detection time
		totalDurationMicros += result.DurationMicros

		// Classify result
		shouldMatch := result.TestCase.ExpectedMatch == nil || *result.TestCase.ExpectedMatch

		if shouldMatch {
			// Expected to match
			if result.Matched && result.IsCorrect {
				truePositives++
				confidenceScores = append(confidenceScores, result.ActualConfidence)
			} else if result.Matched && !result.IsCorrect {
				// Matched but wrong product
				falsePositives++
			} else if !result.Matched {
				falseNegatives++
			}

			// Version extraction tracking
			if result.TestCase.ExpectedVersion != "" {
				versionAttempted++
				if result.VersionExtracted {
					versionExtracted++
				}
			}
		} else {
			// Expected NOT to match (true negatives)
			if !result.Matched {
				trueNegatives++
			} else {
				falsePositives++
			}
		}
	}

	// Set counts
	metrics.TruePositivesCount = truePositives
	metrics.TrueNegativesCount = trueNegatives
	metrics.FalsePositivesCount = falsePositives
	metrics.FalseNegativesCount = falseNegatives

	// Calculate accuracy metrics
	if (falsePositives + trueNegatives) > 0 {
		metrics.FalsePositiveRate = float64(falsePositives) / float64(falsePositives+trueNegatives)
	}

	if (truePositives + falseNegatives) > 0 {
		metrics.TruePositiveRate = float64(truePositives) / float64(truePositives+falseNegatives)
		metrics.Recall = metrics.TruePositiveRate
	}

	if (truePositives + falsePositives) > 0 {
		metrics.Precision = float64(truePositives) / float64(truePositives+falsePositives)
	}

	if (metrics.Precision + metrics.Recall) > 0 {
		metrics.F1Score = 2 * (metrics.Precision * metrics.Recall) / (metrics.Precision + metrics.Recall)
	}

	// Coverage metrics
	metrics.ProtocolsCovered = len(protocols)
	metrics.VersionExtractedCount = versionExtracted
	metrics.VersionAttemptedCount = versionAttempted
	if versionAttempted > 0 {
		metrics.VersionExtractionRate = float64(versionExtracted) / float64(versionAttempted)
	}

	// Confidence metrics
	if len(confidenceScores) > 0 {
		sum := 0.0
		min := 1.0
		max := 0.0
		for _, score := range confidenceScores {
			sum += score
			if score < min {
				min = score
			}
			if score > max {
				max = score
			}
		}
		metrics.ConfidenceMean = sum / float64(len(confidenceScores))
		metrics.ConfidenceMin = min
		metrics.ConfidenceMax = max

		// Calculate median
		// Note: This is a simple approximation (not sorting for performance)
		metrics.ConfidenceMedian = metrics.ConfidenceMean
	}

	// Performance metrics
	if len(results) > 0 {
		metrics.AvgDetectionTimeMicros = totalDurationMicros / int64(len(results))
		metrics.AvgDetectionTimeMs = float64(metrics.AvgDetectionTimeMicros) / 1000.0
	}

	// Pass/fail checks
	metrics.PassFPR = metrics.FalsePositiveRate < metrics.TargetFPR
	metrics.PassTPR = metrics.TruePositiveRate > metrics.TargetTPR
	metrics.PassPrecision = metrics.Precision > metrics.TargetPrecision
	metrics.PassF1 = metrics.F1Score > metrics.TargetF1
	metrics.PassProtocols = metrics.ProtocolsCovered >= metrics.TargetProtocols
	metrics.PassVersionRate = metrics.VersionExtractionRate > metrics.TargetVersionRate
	metrics.PassPerformance = metrics.AvgDetectionTimeMs < metrics.TargetPerfMs

	// Count passed metrics
	passed := 0
	if metrics.PassFPR {
		passed++
	}
	if metrics.PassTPR {
		passed++
	}
	if metrics.PassPrecision {
		passed++
	}
	if metrics.PassF1 {
		passed++
	}
	if metrics.PassProtocols {
		passed++
	}
	if metrics.PassVersionRate {
		passed++
	}
	if metrics.PassPerformance {
		passed++
	}

	metrics.MetricsPassed = passed

	return metrics
}

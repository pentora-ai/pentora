package fingerprint

import (
	"context"
	"sync"
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

	// Per-protocol breakdown
	PerProtocol map[string]ProtocolMetrics `json:"per_protocol"`
}

// ProtocolMetrics represents metrics for a single protocol
type ProtocolMetrics struct {
	Protocol          string  `json:"protocol"`
	TruePositives     int     `json:"true_positives"`
	FalsePositives    int     `json:"false_positives"`
	FalseNegatives    int     `json:"false_negatives"`
	TrueNegatives     int     `json:"true_negatives"`
	FalsePositiveRate float64 `json:"false_positive_rate"`
	TruePositiveRate  float64 `json:"true_positive_rate"`
	Precision         float64 `json:"precision"`
	F1Score           float64 `json:"f1_score"`
	TestCases         int     `json:"test_cases"`
	AvgConfidence     float64 `json:"avg_confidence"`
	AvgDetectTimeUs   int64   `json:"avg_detection_time_us"`
}

// ValidationRunner executes validation tests and computes metrics.
type ValidationRunner struct {
	resolver Resolver
	dataset  *ValidationDataset
	// configurable options
	// thresholds holds target values; expose as ValidationThresholds via getter
	thresholds       ValidationMetrics
	parallelism      int
	timeout          time.Duration
	progressCallback func(float64)
	verbose          bool
}

// ValidationOption configures ValidationRunner behavior.
type ValidationOption func(*ValidationRunner)

// WithThresholds sets custom validation targets used in metrics pass/fail.
func WithThresholds(t ValidationMetrics) ValidationOption {
	return func(vr *ValidationRunner) { vr.thresholds = t }
}

// Thresholds returns the configured thresholds as ValidationThresholds shape
// for compatibility with tests expecting this exact type.
func (vr *ValidationRunner) Thresholds() ValidationThresholds {
	return ValidationThresholds{
		TargetFPR:         vr.thresholds.TargetFPR,
		TargetTPR:         vr.thresholds.TargetTPR,
		TargetPrecision:   vr.thresholds.TargetPrecision,
		TargetF1:          vr.thresholds.TargetF1,
		TargetProtocols:   vr.thresholds.TargetProtocols,
		TargetVersionRate: vr.thresholds.TargetVersionRate,
		TargetPerfMs:      vr.thresholds.TargetPerfMs,
	}
}

// WithParallelism sets number of concurrent test executions.
func WithParallelism(n int) ValidationOption {
	return func(vr *ValidationRunner) {
		if n > 0 {
			vr.parallelism = n
		}
	}
}

// WithTimeout sets maximum duration for entire validation run.
func WithTimeout(d time.Duration) ValidationOption {
	return func(vr *ValidationRunner) { vr.timeout = d }
}

// WithProgressCallback sets callback for progress updates in [0,1].
func WithProgressCallback(cb func(float64)) ValidationOption {
	return func(vr *ValidationRunner) { vr.progressCallback = cb }
}

// WithVerbose enables verbose behavior (reserved for future logs).
func WithVerbose(v bool) ValidationOption {
	return func(vr *ValidationRunner) { vr.verbose = v }
}

// internal item type to represent a test work unit.
type item struct {
	tc          ValidationTestCase
	shouldMatch bool
}

// runParallel executes items with a bounded worker pool and updates results.
func (vr *ValidationRunner) runParallel(ctx context.Context, items []item, results []ValidationResult, total int) {
	sem := make(chan struct{}, vr.parallelism)
	var wg sync.WaitGroup
	for idx, it := range items {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, it item) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				results[i] = ValidationResult{TestCase: it.tc, Matched: false, Error: ctx.Err(), IsCorrect: !it.shouldMatch}
			default:
				results[i] = vr.runTestCase(ctx, it.tc, it.shouldMatch)
			}
			<-sem
			if vr.progressCallback != nil && total > 0 {
				// approximate progress based on filled results slots
				done := 0
				for _, r := range results {
					if r.TestCase.Protocol != "" || r.Error != nil || r.DurationMicros > 0 {
						done++
					}
				}
				vr.progressCallback(float64(done) / float64(total))
			}
		}(idx, it)
	}
	wg.Wait()
}

// calculateMetrics is deprecated; kept for test compatibility. It forwards to
// CalculateMetrics in validation_metrics.go with default targets.
func (vr *ValidationRunner) calculateMetrics(results []ValidationResult) *ValidationMetrics {
	targets := vr.defaultThresholds()
	return CalculateMetrics(results, targets)
}

// NewValidationRunner creates a new validation runner with the given resolver.
func NewValidationRunner(resolver Resolver, datasetPath string, opts ...ValidationOption) (*ValidationRunner, error) {
	dataset, err := LoadValidationDataset(datasetPath)
	if err != nil {
		return nil, err
	}
	// default thresholds to upstream defaults unless overridden by options
	defs := DefaultThresholds()
	vr := &ValidationRunner{
		resolver: resolver,
		dataset:  dataset,
		thresholds: ValidationMetrics{
			TargetFPR:         defs.TargetFPR,
			TargetTPR:         defs.TargetTPR,
			TargetPrecision:   defs.TargetPrecision,
			TargetF1:          defs.TargetF1,
			TargetProtocols:   defs.TargetProtocols,
			TargetVersionRate: defs.TargetVersionRate,
			TargetPerfMs:      defs.TargetPerfMs,
		},
		parallelism: 1,
		timeout:     0,
		verbose:     false,
	}
	for _, opt := range opts {
		opt(vr)
	}
	return vr, nil
}

// NewValidationRunnerWithThresholds preserves upstream API by constructing a runner
// with provided thresholds. Options like parallelism/timeout can still be adjusted later
// via dedicated With* options on a new runner if required.
func NewValidationRunnerWithThresholds(resolver Resolver, datasetPath string, thresholds ValidationThresholds) (*ValidationRunner, error) {
	// Map ValidationThresholds into our ValidationMetrics targets
	// If ValidationThresholds matches ValidationMetrics fields, adapt accordingly.
	vr, err := NewValidationRunner(resolver, datasetPath)
	if err != nil {
		return nil, err
	}
	// Best-effort mapping; if names match, set them.
	vr.thresholds = ValidationMetrics{
		TargetFPR:         thresholds.TargetFPR,
		TargetTPR:         thresholds.TargetTPR,
		TargetPrecision:   thresholds.TargetPrecision,
		TargetF1:          thresholds.TargetF1,
		TargetProtocols:   thresholds.TargetProtocols,
		TargetVersionRate: thresholds.TargetVersionRate,
		TargetPerfMs:      thresholds.TargetPerfMs,
	}
	return vr, nil
}

// Run executes all validation tests and returns aggregated metrics.
func (vr *ValidationRunner) Run(ctx context.Context) (*ValidationMetrics, []ValidationResult, error) {
	// Apply timeout if configured
	if vr.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, vr.timeout)
		defer cancel()
	}

	// Flatten all test cases with shouldMatch flag
	items := make([]item, 0, len(vr.dataset.TruePositives)+len(vr.dataset.TrueNegatives)+len(vr.dataset.EdgeCases))
	for _, tc := range vr.dataset.TruePositives {
		items = append(items, item{tc, true})
	}
	for _, tc := range vr.dataset.TrueNegatives {
		items = append(items, item{tc, false})
	}
	for _, tc := range vr.dataset.EdgeCases {
		items = append(items, item{tc, true})
	}

	total := len(items)
	results := make([]ValidationResult, total)

	if vr.parallelism <= 1 {
		for i, it := range items {
			results[i] = vr.runTestCase(ctx, it.tc, it.shouldMatch)
			if vr.progressCallback != nil && total > 0 {
				vr.progressCallback(float64(i+1) / float64(total))
			}
		}
	} else {
		vr.runParallel(ctx, items, results, total)
	}

	if vr.progressCallback != nil {
		vr.progressCallback(1.0)
	}

	metrics := CalculateMetrics(results, vr.defaultThresholds())
	return metrics, results, nil
}

// runTestCase executes a single test case and returns the result.
func (vr *ValidationRunner) runTestCase(ctx context.Context, tc ValidationTestCase, shouldMatch bool) ValidationResult {
	result := ValidationResult{
		TestCase: tc,
	}

	// Measure detection time (ensure non-zero by flooring to at least 1µs)
	start := time.Now()

	// Run resolver
	input := Input{
		Port:     tc.Port,
		Protocol: tc.Protocol,
		Banner:   tc.Banner,
	}

	resolverResult, err := vr.resolver.Resolve(ctx, input)
	duration := time.Since(start)
	micros := duration.Microseconds()
	if micros == 0 {
		micros = 1
	}
	result.DurationMicros = micros

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

package fingerprint

import (
	"testing"
)

func TestCalculateMetrics_BasicCountsAndRates(t *testing.T) {
	// Construct results: 3 TP, 1 FP, 1 FN, 2 TN
	// Also include version attempts/extractions and confidence/time
	True := func(b bool) *bool { return &b }(true)
	False := func(b bool) *bool { return &b }(false)

	results := []ValidationResult{
		// Should match (positive cases)
		{TestCase: ValidationTestCase{Protocol: "ssh", ExpectedMatch: True, ExpectedVersion: "8.2"}, Matched: true, IsCorrect: true, VersionExtracted: true, ActualConfidence: 0.9, DurationMicros: 1000}, // TP + verEx
		{TestCase: ValidationTestCase{Protocol: "ssh", ExpectedMatch: True}, Matched: true, IsCorrect: true, ActualConfidence: 0.8, DurationMicros: 2000},                                                 // TP
		{TestCase: ValidationTestCase{Protocol: "http", ExpectedMatch: True}, Matched: false, IsCorrect: false, DurationMicros: 1500},                                                                     // FN
		{TestCase: ValidationTestCase{Protocol: "mysql", ExpectedMatch: True}, Matched: true, IsCorrect: false, DurationMicros: 1200},                                                                     // FP

		// Should not match (negative cases)
		{TestCase: ValidationTestCase{Protocol: "ssh", ExpectedMatch: False}, Matched: false, IsCorrect: true, DurationMicros: 800},                       // TN
		{TestCase: ValidationTestCase{Protocol: "http", ExpectedMatch: False}, Matched: false, IsCorrect: true, DurationMicros: 900},                      // TN
		{TestCase: ValidationTestCase{Protocol: "ssh", ExpectedMatch: True}, Matched: true, IsCorrect: true, ActualConfidence: 0.7, DurationMicros: 1100}, // TP
	}

	targets := ValidationMetrics{ // loose targets so pass flags likely true
		TargetFPR:         0.5,
		TargetTPR:         0.5,
		TargetPrecision:   0.5,
		TargetF1:          0.5,
		TargetProtocols:   2,
		TargetVersionRate: 0.5,
		TargetPerfMs:      200.0,
	}

	m := CalculateMetrics(results, targets)

	if m.TotalTestCases != len(results) {
		t.Fatalf("TotalTestCases mismatch: %d", m.TotalTestCases)
	}
	// Counts: TP=3, FP=1, FN=1, TN=2
	if m.TruePositivesCount != 3 || m.FalsePositivesCount != 1 || m.FalseNegativesCount != 1 || m.TrueNegativesCount != 2 {
		t.Fatalf("counts mismatch tp=%d fp=%d fn=%d tn=%d", m.TruePositivesCount, m.FalsePositivesCount, m.FalseNegativesCount, m.TrueNegativesCount)
	}
	// Rates
	// FPR = FP/(FP+TN) = 1/3 ≈ 0.333
	if m.FalsePositiveRate < 0.33 || m.FalsePositiveRate > 0.34 {
		t.Fatalf("FPR out of expected range: %f", m.FalsePositiveRate)
	}
	// TPR = TP/(TP+FN) = 3/4 = 0.75
	if m.TruePositiveRate < 0.74 || m.TruePositiveRate > 0.76 {
		t.Fatalf("TPR out of expected range: %f", m.TruePositiveRate)
	}
	// Precision = TP/(TP+FP) = 3/4 = 0.75
	if m.Precision < 0.74 || m.Precision > 0.76 {
		t.Fatalf("Precision out of expected range: %f", m.Precision)
	}
	// F1 ~ 0.75 (prec=0.75, recall=0.75)
	if m.F1Score < 0.74 || m.F1Score > 0.76 {
		t.Fatalf("F1 out of expected range: %f", m.F1Score)
	}
	// Version extraction rate: 1/1 = 1.0
	if m.VersionAttemptedCount != 1 || m.VersionExtractedCount != 1 || m.VersionExtractionRate != 1.0 {
		t.Fatalf("version stats mismatch: att=%d ex=%d rate=%f", m.VersionAttemptedCount, m.VersionExtractedCount, m.VersionExtractionRate)
	}
	// Protocols covered: ssh, http, mysql => 3
	if m.ProtocolsCovered != 3 {
		t.Fatalf("ProtocolsCovered mismatch: %d", m.ProtocolsCovered)
	}
	// Performance average in ms should be < TargetPerfMs
	if !m.PassPerformance {
		t.Fatalf("expected performance to pass threshold")
	}
	// Per-protocol metrics present
	if len(m.PerProtocol) < 2 {
		t.Fatalf("expected per-protocol breakdown")
	}
	// Pass flags (given loose targets)
	if !m.PassFPR || !m.PassTPR || !m.PassPrecision || !m.PassF1 || !m.PassProtocols || !m.PassVersionRate || !m.PassPerformance {
		t.Fatalf("expected all pass flags true, got: %+v", m)
	}
}

package fingerprint

import "testing"

func TestCalculateFPR(t *testing.T) {
	tests := []struct {
		fp, tn int
		want   float64
	}{
		{0, 0, 0}, {1, 0, 1}, {0, 4, 0}, {1, 3, 0.25},
	}
	for _, tt := range tests {
		if got := CalculateFPR(tt.fp, tt.tn); (got-tt.want) > 1e-9 || (tt.want-got) > 1e-9 {
			t.Fatalf("FPR(%d,%d)=%v want %v", tt.fp, tt.tn, got, tt.want)
		}
	}
}

func TestCalculateTPR(t *testing.T) {
	tests := []struct {
		tp, fn int
		want   float64
	}{
		{0, 0, 0}, {1, 0, 1}, {0, 4, 0}, {2, 2, 0.5},
	}
	for _, tt := range tests {
		if got := CalculateTPR(tt.tp, tt.fn); (got-tt.want) > 1e-9 || (tt.want-got) > 1e-9 {
			t.Fatalf("TPR(%d,%d)=%v want %v", tt.tp, tt.fn, got, tt.want)
		}
	}
}

func TestCalculatePrecision(t *testing.T) {
	tests := []struct {
		tp, fp int
		want   float64
	}{
		{0, 0, 0}, {1, 0, 1}, {0, 4, 0}, {2, 2, 0.5},
	}
	for _, tt := range tests {
		if got := CalculatePrecision(tt.tp, tt.fp); (got-tt.want) > 1e-9 || (tt.want-got) > 1e-9 {
			t.Fatalf("Precision(%d,%d)=%v want %v", tt.tp, tt.fp, got, tt.want)
		}
	}
}

func TestCalculateF1Score(t *testing.T) {
	tests := []struct {
		p, r float64
		want float64
	}{
		{0, 0, 0}, {1, 1, 1}, {1, 0, 0}, {0.5, 0.5, 0.5},
	}
	for _, tt := range tests {
		if got := CalculateF1Score(tt.p, tt.r); (got-tt.want) > 1e-9 || (tt.want-got) > 1e-9 {
			t.Fatalf("F1(%v,%v)=%v want %v", tt.p, tt.r, got, tt.want)
		}
	}
}

func TestCalculateVersionExtractionRate(t *testing.T) {
	tests := []struct {
		ex, at int
		want   float64
	}{
		{0, 0, 0}, {1, 1, 1}, {1, 2, 0.5},
	}
	for _, tt := range tests {
		if got := CalculateVersionExtractionRate(tt.ex, tt.at); (got-tt.want) > 1e-9 || (tt.want-got) > 1e-9 {
			t.Fatalf("VersionRate(%d,%d)=%v want %v", tt.ex, tt.at, got, tt.want)
		}
	}
}

// helper to quickly build a ValidationResult
func vr(protocol string, matched, correct bool, conf float64, micros int64, expVersion bool) ValidationResult {
	r := ValidationResult{
		TestCase:         ValidationTestCase{Protocol: protocol},
		Matched:          matched,
		IsCorrect:        correct,
		ActualConfidence: conf,
		DurationMicros:   micros,
	}
	if expVersion {
		r.TestCase.ExpectedVersion = "1.2.3"
		r.VersionExtracted = matched // pretend version extracted on match
	}
	return r
}

func TestCalculateMetrics_Empty(t *testing.T) {
	// Set stringent targets so only performance can pass with empty input
	targets := ValidationMetrics{TargetFPR: 0.0, TargetTPR: 1.0, TargetPrecision: 1.0, TargetF1: 1.0, TargetProtocols: 1, TargetVersionRate: 1.0, TargetPerfMs: 50}
	m := CalculateMetrics(nil, targets)
	if m.TotalTestCases != 0 || m.ProtocolsCovered != 0 {
		t.Fatalf("expected zeros for empty metrics, got %+v", m)
	}
	if !m.PassPerformance || m.MetricsPassed != 1 {
		t.Fatalf("expected only performance to pass, got %+v", m)
	}
}

func TestCalculateMetrics_MixedCountsAndConfidence(t *testing.T) {
	// Build results: 2 TP (with confidence), 1 FP, 1 FN, 1 TN; versions attempted on two TPs
	res := []ValidationResult{
		vr("http", true, true, 0.9, 900, true),   // TP
		vr("http", true, true, 0.7, 1100, true),  // TP
		vr("ssh", true, false, 0.6, 800, false),  // FP
		vr("ssh", false, false, 0.0, 700, false), // FN (no match)
		{ // TN: expected_match=false, no match
			TestCase:       ValidationTestCase{Protocol: "ftp", ExpectedMatch: func() *bool { b := false; return &b }()},
			Matched:        false,
			IsCorrect:      true,
			DurationMicros: 500,
		},
	}
	targets := ValidationMetrics{TargetFPR: 0.1, TargetTPR: 0.8, TargetPrecision: 0.85, TargetF1: 0.82, TargetProtocols: 2, TargetVersionRate: 0.5, TargetPerfMs: 50}
	m := CalculateMetrics(res, targets)

	if m.TruePositivesCount != 2 || m.FalsePositivesCount != 1 || m.FalseNegativesCount != 1 || m.TrueNegativesCount != 1 {
		t.Fatalf("unexpected counts: %+v", m)
	}
	if m.ProtocolsCovered != 3 { // http, ssh, ftp
		t.Fatalf("expected 3 protocols, got %d", m.ProtocolsCovered)
	}
	// FPR = FP/(FP+TN) = 1/2 = 0.5
	if diff := m.FalsePositiveRate - 0.5; diff > 1e-9 || diff < -1e-9 {
		t.Fatalf("FPR got %v want 0.5", m.FalsePositiveRate)
	}
	// TPR = TP/(TP+FN) = 2/3 ≈ 0.6667
	if m.TruePositiveRate < 0.66 || m.TruePositiveRate > 0.67 {
		t.Fatalf("TPR got %v want ~0.6667", m.TruePositiveRate)
	}
	// Precision = TP/(TP+FP) = 2/3 ≈ 0.6667
	if m.Precision < 0.66 || m.Precision > 0.67 {
		t.Fatalf("Precision got %v want ~0.6667", m.Precision)
	}
	// F1 around 0.6667 as P≈R
	if m.F1Score < 0.66 || m.F1Score > 0.67 {
		t.Fatalf("F1 got %v want ~0.6667", m.F1Score)
	}
	// Version extraction: attempted=2, extracted=2 => 1.0
	if m.VersionAttemptedCount != 2 || m.VersionExtractedCount != 2 || m.VersionExtractionRate != 1.0 {
		t.Fatalf("version metrics unexpected: %+v", m)
	}
	// Confidence: mean of {0.9, 0.7} = 0.8, min=0.7 max=0.9, median≈mean
	if m.ConfidenceMean < 0.79 || m.ConfidenceMean > 0.81 || m.ConfidenceMin != 0.7 || m.ConfidenceMax != 0.9 {
		t.Fatalf("confidence stats unexpected: %+v", m)
	}
	// Avg detection time (integer microseconds average of 900,1100,800,700,500) = 800
	if m.AvgDetectionTimeMicros != 800 {
		t.Fatalf("avg micros got %d want 800", m.AvgDetectionTimeMicros)
	}
	// Targets: with protocols>=2 and versionRate>=0.5 we pass 2/7 checks here, perf <50ms will pass too
	if !m.PassProtocols || !m.PassVersionRate || !m.PassPerformance {
		t.Fatalf("expected protocols, version rate and perf to pass: %+v", m)
	}
}

package fingerprint

// Pure metric helper functions for validation. Keep these side-effect free
// and unit-testable. Formulas follow conventional definitions.

// CalculateFPR computes False Positive Rate: FP / (FP + TN)
func CalculateFPR(fp, tn int) float64 {
	denom := fp + tn
	if denom == 0 {
		return 0.0
	}
	return float64(fp) / float64(denom)
}

// CalculateTPR computes True Positive Rate (Recall): TP / (TP + FN)
func CalculateTPR(tp, fn int) float64 {
	denom := tp + fn
	if denom == 0 {
		return 0.0
	}
	return float64(tp) / float64(denom)
}

// CalculatePrecision computes Precision: TP / (TP + FP)
func CalculatePrecision(tp, fp int) float64 {
	denom := tp + fp
	if denom == 0 {
		return 0.0
	}
	return float64(tp) / float64(denom)
}

// CalculateF1Score computes F1: 2 * (P * R) / (P + R)
func CalculateF1Score(precision, recall float64) float64 {
	sum := precision + recall
	if sum == 0 {
		return 0.0
	}
	return 2 * (precision * recall) / sum
}

// CalculateVersionExtractionRate computes extracted/attempted ratio.
func CalculateVersionExtractionRate(extracted, attempted int) float64 {
	if attempted == 0 {
		return 0.0
	}
	return float64(extracted) / float64(attempted)
}

// CalculateMetrics computes aggregated validation metrics using pure helpers.
func CalculateMetrics(results []ValidationResult, targets ValidationMetrics) *ValidationMetrics {
	metrics := &ValidationMetrics{
		TotalTestCases:    len(results),
		TargetFPR:         targets.TargetFPR,
		TargetTPR:         targets.TargetTPR,
		TargetPrecision:   targets.TargetPrecision,
		TargetF1:          targets.TargetF1,
		TargetProtocols:   targets.TargetProtocols,
		TargetVersionRate: targets.TargetVersionRate,
		TargetPerfMs:      targets.TargetPerfMs,
	}
	// 1) Aggregate basic counts and collections
	tp, tn, fp, fn, verEx, verAttempt, confidence, totalMicros, protocols := aggregate(results)
	// 2) Compute scalar metrics
	assignCounts(metrics, tp, tn, fp, fn)
	computeRates(metrics, tp, tn, fp, fn, verEx, verAttempt)
	metrics.ProtocolsCovered = len(protocols)
	computeConfidence(metrics, confidence)
	computePerformance(metrics, totalMicros, len(results))
	// 3) Evaluate pass/fail
	evaluateTargets(metrics)
	return metrics
}

func aggregate(results []ValidationResult) (tp, tn, fp, fn, verEx, verAttempt int, confidence []float64, totalMicros int64, protocols map[string]bool) {
	protocols = make(map[string]bool)
	for _, r := range results {
		protocols[r.TestCase.Protocol] = true
		totalMicros += r.DurationMicros
		shouldMatch := r.TestCase.ExpectedMatch == nil || *r.TestCase.ExpectedMatch
		if shouldMatch {
			switch {
			case r.Matched && r.IsCorrect:
				tp++
				confidence = append(confidence, r.ActualConfidence)
			case r.Matched && !r.IsCorrect:
				fp++
			case !r.Matched:
				fn++
			}
			if r.TestCase.ExpectedVersion != "" {
				verAttempt++
				if r.VersionExtracted {
					verEx++
				}
			}
		} else {
			if !r.Matched {
				tn++
			} else {
				fp++
			}
		}
	}
	return tp, tn, fp, fn, verEx, verAttempt, confidence, totalMicros, protocols
}

func assignCounts(m *ValidationMetrics, tp, tn, fp, fn int) {
	m.TruePositivesCount = tp
	m.TrueNegativesCount = tn
	m.FalsePositivesCount = fp
	m.FalseNegativesCount = fn
}

func computeRates(m *ValidationMetrics, tp, tn, fp, fn, verEx, verAttempt int) {
	m.FalsePositiveRate = CalculateFPR(fp, tn)
	m.TruePositiveRate = CalculateTPR(tp, fn)
	m.Recall = m.TruePositiveRate
	m.Precision = CalculatePrecision(tp, fp)
	m.F1Score = CalculateF1Score(m.Precision, m.Recall)
	// ProtocolsCovered is set after aggregation.
	m.VersionExtractedCount = verEx
	m.VersionAttemptedCount = verAttempt
	m.VersionExtractionRate = CalculateVersionExtractionRate(verEx, verAttempt)
}

func computeConfidence(m *ValidationMetrics, scores []float64) {
	if n := len(scores); n > 0 {
		sum, min, max := 0.0, 1.0, 0.0
		for _, c := range scores {
			sum += c
			if c < min {
				min = c
			}
			if c > max {
				max = c
			}
		}
		m.ConfidenceMean = sum / float64(n)
		m.ConfidenceMin = min
		m.ConfidenceMax = max
		m.ConfidenceMedian = m.ConfidenceMean
	}
}

func computePerformance(m *ValidationMetrics, totalMicros int64, total int) {
	if total > 0 {
		m.AvgDetectionTimeMicros = totalMicros / int64(total)
		m.AvgDetectionTimeMs = float64(m.AvgDetectionTimeMicros) / 1000.0
	}
}

func evaluateTargets(m *ValidationMetrics) {
	// ProtocolsCovered is set in CalculateMetrics from aggregated protocols
	pass := 0
	if m.FalsePositiveRate < m.TargetFPR {
		pass++
	}
	if m.TruePositiveRate > m.TargetTPR {
		pass++
	}
	if m.Precision > m.TargetPrecision {
		pass++
	}
	if m.F1Score > m.TargetF1 {
		pass++
	}
	if m.ProtocolsCovered >= m.TargetProtocols {
		pass++
	}
	if m.VersionExtractionRate > m.TargetVersionRate {
		pass++
	}
	if m.AvgDetectionTimeMs < m.TargetPerfMs {
		pass++
	}
	m.PassFPR = m.FalsePositiveRate < m.TargetFPR
	m.PassTPR = m.TruePositiveRate > m.TargetTPR
	m.PassPrecision = m.Precision > m.TargetPrecision
	m.PassF1 = m.F1Score > m.TargetF1
	m.PassProtocols = m.ProtocolsCovered >= m.TargetProtocols
	m.PassVersionRate = m.VersionExtractionRate > m.TargetVersionRate
	m.PassPerformance = m.AvgDetectionTimeMs < m.TargetPerfMs
	m.MetricsPassed = pass
}

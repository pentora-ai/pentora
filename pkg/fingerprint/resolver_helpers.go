package fingerprint

import (
	"regexp"
	"strings"
)

// isHardRejected returns true if any exclude pattern matches the banner.
func isHardRejected(banner string, exclude []*regexp.Regexp) bool {
	if len(exclude) == 0 {
		return false
	}
	for _, rx := range exclude {
		if rx.MatchString(banner) {
			return true
		}
	}
	return false
}

// softExcludePenalty returns cumulative penalty based on soft-exclude matches.
// Each match applies the provided perMatchPenalty (e.g., 0.20).
func softExcludePenalty(banner string, soft []*regexp.Regexp, perMatchPenalty float64) float64 {
	if len(soft) == 0 || perMatchPenalty <= 0 {
		return 0
	}
	penalty := 0.0
	for _, rx := range soft {
		if rx.MatchString(banner) {
			penalty += perMatchPenalty
		}
	}
	if penalty < 0 {
		return 0
	}
	if penalty > 1 {
		return 1
	}
	return penalty
}

// calculateConfidence computes a confidence score based on pattern strength,
// soft-exclude penalties, and optional port bonuses.
func calculateConfidence(base, softPenalty, portBonus float64) float64 {
	conf := base - softPenalty + portBonus
	if conf < 0 {
		conf = 0
	}
	if conf > 1 {
		conf = 1
	}
	return conf
}

// sigmoid maps a value to (0,1) range; can be used to smooth scores.
// sigmoid is currently unused; keep it for Phase 2 when scoring smoothing is introduced.
// Temporarily comment out to satisfy lint until used.
// func sigmoid(x float64) float64 {
//     return x
// }

// normalizeVersion trims and lowercases a version-like string.
func normalizeVersion(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

// containsInt checks if a target port is present in a slice.
func containsPort(ports []int, p int) bool {
	for _, v := range ports {
		if v == p {
			return true
		}
	}
	return false
}

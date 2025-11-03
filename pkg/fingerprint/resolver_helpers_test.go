package fingerprint

import (
	"regexp"
	"testing"
)

func TestIsHardRejected(t *testing.T) {
	banner := "server: myapp version 1.2.3 (test build)"
	rx := []*regexp.Regexp{regexp.MustCompile(`test build`)}
	if !isHardRejected(banner, rx) {
		t.Fatalf("expected hard reject when pattern matches")
	}
	if isHardRejected(banner, []*regexp.Regexp{regexp.MustCompile(`nope`)}) {
		t.Fatalf("did not expect hard reject for non-matching pattern")
	}
}

func TestSoftExcludePenalty(t *testing.T) {
	banner := "server: foo test build; debug mode"
	soft := []*regexp.Regexp{
		regexp.MustCompile(`test build`),
		regexp.MustCompile(`debug`),
		regexp.MustCompile(`nope`),
	}
	p := softExcludePenalty(banner, soft, 0.20)
	if p <= 0 {
		t.Fatalf("expected positive penalty, got %v", p)
	}
	if p < 0.39 || p > 0.41 { // two matches ~ 0.40
		t.Fatalf("expected ~0.40 penalty, got %v", p)
	}
}

func TestCalculateConfidence(t *testing.T) {
	// base 0.8 - 0.4 + 0.1 = 0.5
	if got := calculateConfidence(0.8, 0.4, 0.1); got < 0.49 || got > 0.51 {
		t.Fatalf("expected ~0.5, got %v", got)
	}
	// clamp low
	if got := calculateConfidence(0.1, 0.5, 0.0); got != 0 {
		t.Fatalf("expected clamp to 0, got %v", got)
	}
	// clamp high
	if got := calculateConfidence(0.9, 0.0, 0.2); got != 1 {
		t.Fatalf("expected clamp to 1, got %v", got)
	}
}

func TestNormalizeVersion(t *testing.T) {
	if got := normalizeVersion("  V1.2.3 "); got != "v1.2.3" {
		t.Fatalf("expected 'v1.2.3', got %q", got)
	}
	if got := normalizeVersion("\t 1.0.0-RC \n"); got != "1.0.0-rc" {
		t.Fatalf("expected '1.0.0-rc', got %q", got)
	}
}

func TestContainsPort(t *testing.T) {
	ports := []int{22, 80, 443}
	if !containsPort(ports, 80) {
		t.Fatalf("expected to contain 80")
	}
	if containsPort(ports, 21) {
		t.Fatalf("did not expect to contain 21")
	}
}

func TestSoftExcludePenalty_NoPatternsOrZeroPenalty(t *testing.T) {
	if p := softExcludePenalty("anything", nil, 0.20); p != 0 {
		t.Fatalf("expected 0 with nil patterns, got %v", p)
	}
	rx := []*regexp.Regexp{regexp.MustCompile(`match`)}
	if p := softExcludePenalty("match", rx, 0.0); p != 0 {
		t.Fatalf("expected 0 with zero perMatchPenalty, got %v", p)
	}
}

func TestCalculateConfidence_ClampEdges(t *testing.T) {
	if got := calculateConfidence(-0.5, 0.0, 0.0); got != 0 {
		t.Fatalf("expected clamp to 0, got %v", got)
	}
	if got := calculateConfidence(2.0, 0.0, 0.5); got != 1 {
		t.Fatalf("expected clamp to 1, got %v", got)
	}
}

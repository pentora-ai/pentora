package fingerprint

import (
    "context"
    "sync"
    "testing"
)

// helper to reset global registry between tests
func resetFingerprinterRegistry() {
	fingerprinterMu.Lock()
	defer fingerprinterMu.Unlock()
	fingerprinterSet = make([]Fingerprinter, 0)
}

type simpleFP struct{ id string }

func (s *simpleFP) ID() string                   { return s.id }
func (s *simpleFP) SupportedProtocols() []string { return nil }
func (s *simpleFP) AnalyzePassive(ctx context.Context, obs PassiveObservation) (*ServiceCandidate, bool, error) {
	return nil, false, nil
}
func (s *simpleFP) ActiveProbes() []Probe { return nil }
func (s *simpleFP) Verify(ctx context.Context, probe Probe, response []byte) (*ServiceCandidate, bool, error) {
	return nil, false, nil
}

func TestFingerprinterRegistry_RegisterAndList(t *testing.T) {
	resetFingerprinterRegistry()
	a := &simpleFP{id: "a"}
	b := &simpleFP{id: "b"}

	RegisterFingerprinter(a)
	RegisterFingerprinter(b)

	got := ListFingerprinters()
	if len(got) != 2 {
		t.Fatalf("expected 2, got %d", len(got))
	}

	// Ensure returned slice is a copy and not alias of internal slice
	got[0] = nil
	again := ListFingerprinters()
	if again[0] == nil {
		t.Fatalf("expected snapshot copy, not alias")
	}
}

func TestFingerprinterRegistry_NewDefaultCoordinator(t *testing.T) {
	resetFingerprinterRegistry()
	RegisterFingerprinter(&simpleFP{id: "x"})
	RegisterFingerprinter(&simpleFP{id: "y"})

	c := NewDefaultCoordinator()
	if c == nil {
		t.Fatalf("expected coordinator")
	}
	if len(c.fingerprinters) != 2 {
		t.Fatalf("expected 2, got %d", len(c.fingerprinters))
	}
}

func TestFingerprinterRegistry_ConcurrentRegisterAndList(t *testing.T) {
	resetFingerprinterRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			RegisterFingerprinter(&simpleFP{id: "fp"})
			_ = ListFingerprinters()
		}(i)
	}
	wg.Wait()
	if len(ListFingerprinters()) == 0 {
		t.Fatalf("expected some fingerprinters registered")
	}
}

package fingerprint

import (
	"context"
	"errors"
	"testing"
)

type stubFingerprinter struct {
	id         string
	passive    func(ctx context.Context, obs PassiveObservation) (*ServiceCandidate, bool, error)
	probes     []Probe
	verifyFunc func(ctx context.Context, probe Probe, resp []byte) (*ServiceCandidate, bool, error)
}

func (s *stubFingerprinter) ID() string { return s.id }

func (s *stubFingerprinter) SupportedProtocols() []string { return []string{"tcp"} }

func (s *stubFingerprinter) AnalyzePassive(ctx context.Context, obs PassiveObservation) (*ServiceCandidate, bool, error) {
	if s.passive == nil {
		return nil, false, nil
	}
	return s.passive(ctx, obs)
}

func (s *stubFingerprinter) ActiveProbes() []Probe { return s.probes }

func (s *stubFingerprinter) Verify(ctx context.Context, probe Probe, resp []byte) (*ServiceCandidate, bool, error) {
	if s.verifyFunc == nil {
		return nil, false, nil
	}
	return s.verifyFunc(ctx, probe, resp)
}

type stubExecutor struct {
	responses map[string][]byte
	failProbe string
}

//nolint:revive
func (e *stubExecutor) Execute(ctx context.Context, probe Probe) ([]byte, error) {
	if probe.ID == e.failProbe {
		return nil, errors.New("probe failed")
	}
	if resp, ok := e.responses[probe.ID]; ok {
		return resp, nil
	}
	return nil, nil
}

//nolint:revive
func TestCoordinatorPassivePriority(t *testing.T) {
	ctx := context.Background()
	obs := PassiveObservation{Banner: []byte("SSH-2.0-OpenSSH"), Port: 2222}

	sshFP := &stubFingerprinter{
		id: "ssh",
		passive: func(ctx context.Context, obs PassiveObservation) (*ServiceCandidate, bool, error) {
			if string(obs.Banner) != "SSH-2.0-OpenSSH" {
				return nil, false, nil
			}
			return &ServiceCandidate{Protocol: "ssh", Confidence: 0.95, Metadata: map[string]string{"banner": string(obs.Banner)}}, true, nil
		},
	}

	httpFP := &stubFingerprinter{
		id: "http",
		passive: func(ctx context.Context, obs PassiveObservation) (*ServiceCandidate, bool, error) {
			return &ServiceCandidate{Protocol: "http", Confidence: 0.40}, false, nil
		},
	}

	coord := NewCoordinator(sshFP, httpFP)
	cand, err := coord.Identify(ctx, obs, nil)
	if err != nil {
		t.Fatalf("Identify error: %v", err)
	}
	if cand == nil || cand.Protocol != "ssh" {
		t.Fatalf("expected ssh candidate, got %#v", cand)
	}
	if cand.Source != "ssh" {
		t.Fatalf("expected source ssh, got %s", cand.Source)
	}
}

//nolint:revive
func TestCoordinatorActiveProbe(t *testing.T) {
	ctx := context.Background()
	obs := PassiveObservation{Port: 80}

	httpFP := &stubFingerprinter{
		id:     "http",
		probes: []Probe{{ID: "http-get", Payload: []byte("GET / HTTP/1.0\r\n\r\n")}},
		verifyFunc: func(ctx context.Context, probe Probe, resp []byte) (*ServiceCandidate, bool, error) {
			if probe.ID != "http-get" {
				return nil, false, nil
			}
			if len(resp) == 0 {
				return nil, false, nil
			}
			return &ServiceCandidate{Protocol: "http", Confidence: 0.88, Metadata: map[string]string{"status_line": string(resp)}}, true, nil
		},
	}

	exec := &stubExecutor{responses: map[string][]byte{"http-get": []byte("HTTP/1.1 200 OK")}}

	coord := NewCoordinator(httpFP)
	cand, err := coord.Identify(ctx, obs, exec)
	if err != nil {
		t.Fatalf("Identify error: %v", err)
	}
	if cand == nil || cand.Protocol != "http" {
		t.Fatalf("expected http candidate, got %#v", cand)
	}
	if cand.MatchedProbe != "http-get" {
		t.Fatalf("expected matched probe 'http-get', got %q", cand.MatchedProbe)
	}
}

func TestCoordinatorProbeFailure(t *testing.T) {
	ctx := context.Background()
	fp := &stubFingerprinter{
		id:     "ftp",
		probes: []Probe{{ID: "ftp-banner"}},
	}

	exec := &stubExecutor{failProbe: "ftp-banner"}

	coord := NewCoordinator(fp)
	_, err := coord.Identify(ctx, PassiveObservation{}, exec)
	if err == nil {
		t.Fatalf("expected error when probe fails")
	}
}

func TestRegistryHelpers(t *testing.T) {
	fingerprinterMu.Lock()
	fingerprinterSet = nil
	fingerprinterMu.Unlock()

	fp := &stubFingerprinter{id: "demo"}
	RegisterFingerprinter(fp)

	list := ListFingerprinters()
	if len(list) != 1 || list[0].ID() != "demo" {
		t.Fatalf("unexpected list result: %#v", list)
	}

	coord := NewDefaultCoordinator()
	cand, err := coord.Identify(context.Background(), PassiveObservation{}, nil)
	if err != nil {
		t.Fatalf("Identify error: %v", err)
	}
	if cand != nil {
		t.Fatalf("expected no candidate from empty fingerprinter, got %#v", cand)
	}
}

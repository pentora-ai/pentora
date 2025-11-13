package fingerprint

import (
	"context"
	"errors"
	"testing"
)

type mockExecutor func(ctx context.Context, probe Probe) ([]byte, error)

func (m mockExecutor) Execute(ctx context.Context, probe Probe) ([]byte, error) { return m(ctx, probe) }

type fpMock struct {
	id          string
	priority    Priority
	passiveCand *ServiceCandidate
	passiveDone bool
	passiveErr  error
	probes      []Probe
	verifySeq   []struct {
		resp []byte
		cand *ServiceCandidate
		done bool
		err  error
	}
	verifyIdx int
}

func (f *fpMock) ID() string                   { return f.id }
func (f *fpMock) Priority() Priority           { return f.priority }
func (f *fpMock) SupportedProtocols() []string { return nil }
func (f *fpMock) AnalyzePassive(ctx context.Context, obs PassiveObservation) (*ServiceCandidate, bool, error) {
	return f.passiveCand, f.passiveDone, f.passiveErr
}
func (f *fpMock) ActiveProbes() []Probe { return f.probes }
func (f *fpMock) Verify(ctx context.Context, probe Probe, response []byte) (*ServiceCandidate, bool, error) {
	if f.verifyIdx >= len(f.verifySeq) {
		return nil, false, nil
	}
	v := f.verifySeq[f.verifyIdx]
	f.verifyIdx++
	return v.cand, v.done, v.err
}

func TestCoordinator_RegisterAndIdentify_NoFingerprinters(t *testing.T) {
	c := NewCoordinator()
	got, err := c.Identify(context.Background(), PassiveObservation{}, nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}
}

func TestCoordinator_Register_Appends(t *testing.T) {
	c := NewCoordinator()
	c.Register(&fpMock{id: "a"})
	c.Register(&fpMock{id: "b"})
	if len(c.fingerprinters) != 2 {
		t.Fatalf("expected 2 fingerprinters, got %d", len(c.fingerprinters))
	}
}

func TestCoordinator_Identify_PassiveFinalizedWins(t *testing.T) {
	fp := &fpMock{id: "fp1", passiveCand: &ServiceCandidate{Protocol: "http", Confidence: 0.9}, passiveDone: true}
	c := NewCoordinator(fp)
	got, err := c.Identify(context.Background(), PassiveObservation{}, nil)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got == nil || got.Source != "fp1" || got.Confidence != 0.9 {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestCoordinator_Identify_ActiveProbesAndBestConfidence(t *testing.T) {
	fp1 := &fpMock{id: "fp1", probes: []Probe{{ID: "p1"}}, verifySeq: []struct {
		resp []byte
		cand *ServiceCandidate
		done bool
		err  error
	}{
		{nil, &ServiceCandidate{Protocol: "ssh", Confidence: 0.6}, true, nil},
	}}
	fp2 := &fpMock{id: "fp2", probes: []Probe{{ID: "p2"}}, verifySeq: []struct {
		resp []byte
		cand *ServiceCandidate
		done bool
		err  error
	}{
		{nil, &ServiceCandidate{Protocol: "http", Confidence: 0.8}, true, nil},
	}}
	exec := mockExecutor(func(ctx context.Context, probe Probe) ([]byte, error) { return []byte("ok"), nil })
	c := NewCoordinator(fp1, fp2)
	got, err := c.Identify(context.Background(), PassiveObservation{}, exec)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got == nil || got.Source != "fp2" || got.MatchedProbe != "p2" || got.Confidence != 0.8 {
		t.Fatalf("unexpected result: %#v", got)
	}
}

func TestCoordinator_Identify_ErrorPaths(t *testing.T) {
	// passive error
	fpE := &fpMock{id: "err", passiveErr: errors.New("boom")}
	if _, err := NewCoordinator(fpE).Identify(context.Background(), PassiveObservation{}, nil); err == nil {
		t.Fatalf("expected error from passive")
	}

	// probe error
	fpP := &fpMock{id: "fp", probes: []Probe{{ID: "q"}}}
	execErr := mockExecutor(func(ctx context.Context, probe Probe) ([]byte, error) { return nil, errors.New("x") })
	if _, err := NewCoordinator(fpP).Identify(context.Background(), PassiveObservation{}, execErr); err == nil {
		t.Fatalf("expected probe error")
	}

	// verify error
	fpV := &fpMock{id: "fp", probes: []Probe{{ID: "q"}}, verifySeq: []struct {
		resp []byte
		cand *ServiceCandidate
		done bool
		err  error
	}{
		{nil, nil, false, errors.New("v")},
	}}
	execOK := mockExecutor(func(ctx context.Context, probe Probe) ([]byte, error) { return []byte("ok"), nil })
	if _, err := NewCoordinator(fpV).Identify(context.Background(), PassiveObservation{}, execOK); err == nil {
		t.Fatalf("expected verify error")
	}
}

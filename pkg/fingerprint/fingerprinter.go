package fingerprint

import (
	"context"
	"fmt"
	"sort"
)

// ServiceCandidate represents a potential identification for a network service.
type ServiceCandidate struct {
	Protocol     string            // Normalized protocol identifier, e.g. "http", "ssh"
	Confidence   float64           // 0.0-1.0 confidence score assigned by the fingerprinter
	Metadata     map[string]string // Optional key/value metadata (banner, product, notes)
	Evidence     []byte            // Raw bytes that led to this conclusion (probe response, banner, etc.)
	MatchedProbe string            // ID of the probe that produced evidence (if any)
	Source       string            // ID of the fingerprinter contributing this candidate
}

// PassiveObservation contains information gathered before active probing occurs.
type PassiveObservation struct {
	Port          int               // Target port
	Banner        []byte            // Raw banner or greeting captured during connection setup
	ProtocolHints []string          // Initial guesses (e.g. from port mapping, previous runs)
	Attributes    map[string]string // Additional facts (TLS SNI, ALPN, cert CN, reverse DNS, etc.)
}

// Probe describes an active interaction attempt used to refine identification.
type Probe struct {
	ID          string // Unique identifier for telemetry & verification
	Description string
	Payload     []byte // Raw request payload to send
	ContentType string // Optional hint describing payload semantics ("http", "ssh", ...)
}

// ProbeExecutor executes a probe and returns the raw response bytes.
type ProbeExecutor interface {
	Execute(ctx context.Context, probe Probe) ([]byte, error)
}

// Fingerprinter defines the behaviour required for active/passive service identification modules.
type Fingerprinter interface {
	ID() string
	SupportedProtocols() []string

	// AnalyzePassive inspects the passive observation and may return a candidate. The boolean indicates
	// whether the fingerprinter considers the candidate conclusive (true) or if further probing is suggested (false).
	AnalyzePassive(ctx context.Context, obs PassiveObservation) (*ServiceCandidate, bool, error)

	// ActiveProbes returns the probes this fingerprinter wants to run if passive analysis was inconclusive.
	ActiveProbes() []Probe

	// Verify inspects the probe response and may return a refined candidate. The boolean indicates whether
	// the fingerprinter is satisfied with the derived candidate. Returning nil candidate means "no match".
	Verify(ctx context.Context, probe Probe, response []byte) (*ServiceCandidate, bool, error)
}

// Coordinator manages multiple fingerprinters and orchestrates passive and active analysis.
type Coordinator struct {
	fingerprinters []Fingerprinter
}

// NewCoordinator builds a Coordinator with zero or more fingerprinters.
func NewCoordinator(fps ...Fingerprinter) *Coordinator {
	return &Coordinator{fingerprinters: append([]Fingerprinter(nil), fps...)}
}

// Register adds a fingerprinter to the coordinator.
func (c *Coordinator) Register(fp Fingerprinter) {
	c.fingerprinters = append(c.fingerprinters, fp)
}

// Identify runs passive analysis and, if required, active probes using the provided executor. It returns the
// highest confidence candidate across all fingerprinters. If no identification is possible, nil is returned.
func (c *Coordinator) Identify(ctx context.Context, obs PassiveObservation, exec ProbeExecutor) (*ServiceCandidate, error) {
	if len(c.fingerprinters) == 0 {
		return nil, nil
	}

	type candidate struct {
		data      *ServiceCandidate
		finalized bool
	}

	var candidates []candidate

	for _, fp := range c.fingerprinters {
		cand, done, err := fp.AnalyzePassive(ctx, obs)
		if err != nil {
			return nil, fmt.Errorf("fingerprinter %s passive analysis failed: %w", fp.ID(), err)
		}
		if cand != nil {
			copied := *cand
			copied.Source = fp.ID()
			candidates = append(candidates, candidate{data: &copied, finalized: done})
			if done {
				continue
			}
		}

		if exec == nil {
			continue
		}

		for _, probe := range fp.ActiveProbes() {
			resp, err := exec.Execute(ctx, probe)
			if err != nil {
				return nil, fmt.Errorf("fingerprinter %s probe %s failed: %w", fp.ID(), probe.ID, err)
			}

			cand, done, err := fp.Verify(ctx, probe, resp)
			if err != nil {
				return nil, fmt.Errorf("fingerprinter %s verify for probe %s failed: %w", fp.ID(), probe.ID, err)
			}
			if cand == nil {
				continue
			}
			copied := *cand
			copied.Source = fp.ID()
			copied.MatchedProbe = probe.ID
			candidates = append(candidates, candidate{data: &copied, finalized: done})
			if done {
				break
			}
		}
	}

	if len(candidates) == 0 {
		return nil, nil
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].data.Confidence > candidates[j].data.Confidence
	})

	return candidates[0].data, nil
}

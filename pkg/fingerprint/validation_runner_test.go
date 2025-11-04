package fingerprint

import (
	"context"
	"fmt"
	"testing"
)

// simple stub resolver to exercise runTestCase and Run aggregation
type stubResolver struct {
	res      Result
	err      error
	errProto string
}

func (s stubResolver) Resolve(_ context.Context, in Input) (Result, error) {
	if s.err != nil && (s.errProto == "" || s.errProto == in.Protocol) {
		return Result{}, s.err
	}
	return s.res, nil
}

func TestRunTestCase_Branches(t *testing.T) {
	vr := &ValidationRunner{resolver: stubResolver{res: Result{Product: "nginx", Vendor: "nginx", Version: "1.21", Confidence: 0.9}}}

	// TP: shouldMatch, correct product, expected version -> VersionExtracted true
	tp := vr.runTestCase(context.Background(), ValidationTestCase{Protocol: "http", Banner: "Server: nginx", ExpectedProduct: "nginx", ExpectedVersion: "1.21"}, true)
	if !tp.Matched || !tp.IsCorrect || !tp.VersionExtracted {
		t.Fatalf("expected TP with version extracted, got %+v", tp)
	}

	// FP: shouldMatch, wrong product
	fp := vr.runTestCase(context.Background(), ValidationTestCase{Protocol: "http", Banner: "Server: nginx", ExpectedProduct: "apache"}, true)
	if !fp.Matched || fp.IsCorrect {
		t.Fatalf("expected FP (matched but incorrect), got %+v", fp)
	}

	// TN: expected_match=false and no match (error path)
	vr.resolver = stubResolver{err: fmt.Errorf("no matching rule found")}
	b := false
	tn := vr.runTestCase(context.Background(), ValidationTestCase{Protocol: "ssh", Banner: "HTTP/1.1 200 OK", ExpectedMatch: &b}, false)
	if tn.Matched || !tn.IsCorrect {
		t.Fatalf("expected TN (no match, correct), got %+v", tn)
	}

	// FN: shouldMatch but no match
	fn := vr.runTestCase(context.Background(), ValidationTestCase{Protocol: "ssh", Banner: "SSH-2.0-OpenSSH_8.9", ExpectedProduct: "OpenSSH"}, true)
	if fn.Matched || fn.IsCorrect {
		t.Fatalf("expected FN (no match, incorrect), got %+v", fn)
	}
}

func TestRun_AggregatesAndTargets(t *testing.T) {
	// Two cases: one TP (resolver returns match), one TN (resolver returns error)
	// Return match for http, error for ssh to produce TP and TN
	r := &ValidationRunner{resolver: stubResolver{res: Result{Product: "nginx", Vendor: "nginx", Version: "", Confidence: 0.8}, err: fmt.Errorf("no matching rule found"), errProto: "ssh"}}
	r.dataset = &ValidationDataset{
		TruePositives: []ValidationTestCase{{Protocol: "http", Banner: "Server: nginx", ExpectedProduct: "nginx"}},
		TrueNegatives: []ValidationTestCase{{Protocol: "ssh", Banner: "HTTP/1.1 200 OK", ExpectedMatch: func() *bool { b := false; return &b }()}},
	}
	m, results, err := r.Run(context.Background())
	if err != nil || len(results) != 2 {
		t.Fatalf("unexpected run outcome: m=%+v results=%d err=%v", m, len(results), err)
	}
	if m.TotalTestCases != 2 || m.TruePositivesCount != 1 || m.TrueNegativesCount != 1 {
		t.Fatalf("unexpected metrics: %+v", m)
	}
	if m.TargetPerfMs != 50.0 || m.TargetFPR != 0.10 {
		t.Fatalf("targets not set as expected: %+v", m)
	}
}

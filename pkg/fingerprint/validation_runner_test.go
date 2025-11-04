package fingerprint

import (
	"context"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// simpleResolver is a tiny test resolver that simulates work and returns
// deterministic results based on banner content.
type simpleResolver struct{ delay time.Duration }

func (s simpleResolver) Resolve(ctx context.Context, in Input) (Result, error) {
	if s.delay > 0 {
		select {
		case <-time.After(s.delay):
		case <-ctx.Done():
			return Result{}, ctx.Err()
		}
	}
	// very basic: if banner contains "apache" return Apache; otherwise no match
	if in.Banner != "" && (in.Protocol == "http" || in.Protocol == "https") && containsFold(in.Banner, "apache") {
		return Result{Product: "Apache", Vendor: "Apache", Version: "2.4", Confidence: 0.9}, nil
	}
	return Result{}, context.DeadlineExceeded
}

func containsFold(s, sub string) bool { return len(s) >= len(sub) && (stringIndexFold(s, sub) >= 0) }

// stringIndexFold: naive case-insensitive contains for tests only.
func stringIndexFold(s, sub string) int {
	ls, lsub := len(s), len(sub)
	if lsub == 0 {
		return 0
	}
	for i := 0; i+lsub <= ls; i++ {
		ok := true
		for j := 0; j < lsub; j++ {
			a, b := s[i+j], sub[j]
			if 'A' <= a && a <= 'Z' {
				a += 'a' - 'A'
			}
			if 'A' <= b && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

func TestValidationRunner_WithThresholdsOverrides(t *testing.T) {
	resolver := simpleResolver{}
	// dataset with one TP and one TN
	ds := &ValidationDataset{
		TruePositives: []ValidationTestCase{{Protocol: "http", Port: 80, Banner: "Server: Apache/2.4", ExpectedProduct: "Apache"}},
		TrueNegatives: []ValidationTestCase{{Protocol: "http", Port: 80, Banner: "Server: nginx/1.18", ExpectedMatch: boolPtr(false)}},
	}
	// write dataset to temp file via helper
	path := writeTempDataset(t, ds)

	// strict thresholds that will likely fail
	strict := ValidationMetrics{TargetTPR: 1.0, TargetPrecision: 1.0, TargetF1: 1.0, TargetFPR: 0.0, TargetProtocols: 3, TargetVersionRate: 1.0, TargetPerfMs: 0.1}
	runner, err := NewValidationRunner(resolver, path, WithThresholds(strict))
	require.NoError(t, err)

	metrics, results, err := runner.Run(context.Background())
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, strict.TargetTPR, metrics.TargetTPR)
}

func TestValidationRunner_ParallelismAndProgress(t *testing.T) {
	resolver := simpleResolver{delay: 5 * time.Millisecond}
	ds := &ValidationDataset{
		TruePositives: []ValidationTestCase{
			{Protocol: "http", Port: 80, Banner: "apache", ExpectedProduct: "Apache"},
			{Protocol: "http", Port: 80, Banner: "apache", ExpectedProduct: "Apache"},
			{Protocol: "http", Port: 80, Banner: "apache", ExpectedProduct: "Apache"},
			{Protocol: "http", Port: 80, Banner: "apache", ExpectedProduct: "Apache"},
		},
		TrueNegatives: []ValidationTestCase{{Protocol: "http", Port: 80, Banner: "nginx", ExpectedMatch: boolPtr(false)}},
	}
	path := writeTempDataset(t, ds)

	var progressCalls int32
	runner, err := NewValidationRunner(resolver, path, WithParallelism(4), WithProgressCallback(func(p float64) { atomic.AddInt32(&progressCalls, 1) }))
	require.NoError(t, err)

	metrics, results, err := runner.Run(context.Background())
	require.NoError(t, err)
	require.NotNil(t, metrics)
	require.Len(t, results, 5)
	// progress should be reported more than once including completion
	require.Greater(t, atomic.LoadInt32(&progressCalls), int32(1))
}

func TestValidationRunner_TimeoutCancelsRun(t *testing.T) {
	resolver := simpleResolver{delay: 50 * time.Millisecond}
	ds := &ValidationDataset{
		TruePositives: []ValidationTestCase{{Protocol: "http", Port: 80, Banner: "apache", ExpectedProduct: "Apache"}},
		TrueNegatives: []ValidationTestCase{{Protocol: "http", Port: 80, Banner: "nginx", ExpectedMatch: boolPtr(false)}},
	}
	path := writeTempDataset(t, ds)

	runner, err := NewValidationRunner(resolver, path, WithParallelism(2), WithTimeout(1*time.Millisecond))
	require.NoError(t, err)

	_, results, _ := runner.Run(context.Background())
	// At least one result should carry a context error due to timeout
	var hasCtxErr bool
	for _, r := range results {
		if r.Error != nil {
			hasCtxErr = true
			break
		}
	}
	require.True(t, hasCtxErr, "expected at least one context error due to timeout")
}

// writeTempDataset writes the dataset to a temp file that LoadValidationDataset can read via the same YAML codec.
func writeTempDataset(t *testing.T, ds *ValidationDataset) string {
	t.Helper()
	// Serialize minimal YAML by hand to avoid external deps.
	// Only fields we use in tests are included.
	// true_negatives require expected_match: false to count as TNs.
	content := "true_positives:\n"
	for _, tc := range ds.TruePositives {
		content += "  - protocol: " + tc.Protocol + "\n"
		content += "    port: " + itoa(tc.Port) + "\n"
		content += "    banner: \"" + tc.Banner + "\"\n"
		if tc.ExpectedProduct != "" {
			content += "    expected_product: \"" + tc.ExpectedProduct + "\"\n"
		}
	}
	content += "true_negatives:\n"
	for _, tc := range ds.TrueNegatives {
		content += "  - protocol: " + tc.Protocol + "\n"
		content += "    port: " + itoa(tc.Port) + "\n"
		content += "    banner: \"" + tc.Banner + "\"\n"
		content += "    expected_match: false\n"
	}

	f := t.TempDir() + "/dataset.yaml"
	writeFile(t, f, []byte(content))
	return f
}

func writeFile(t *testing.T, p string, b []byte) {
	t.Helper()
	require.NoError(t, osWriteFile(p, b, 0o600))
}

// indirections to avoid importing extra packages in this test file
var osWriteFile = writeFileImpl

func writeFileImpl(name string, data []byte, perm uint32) error {
	return osWriteFileReal(name, data, perm)
}

// real os wrappers (replaced below for build)
// Use the real os.WriteFile for tests
var osWriteFileReal = func(name string, data []byte, perm uint32) error { return os.WriteFile(name, data, 0o600) }

// minimal integer to string helper for ports
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	buf := [20]byte{}
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

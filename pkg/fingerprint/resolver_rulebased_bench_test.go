package fingerprint

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"gopkg.in/yaml.v3"
)

// BenchmarkResolverSingleMatch benchmarks resolver performance with a single rule match.
func BenchmarkResolverSingleMatch(b *testing.B) {
	rules := []StaticRule{
		{
			ID:              "bench.http.apache",
			Protocol:        "http",
			Product:         "Apache",
			Vendor:          "Apache",
			Match:           "apache",
			PatternStrength: 0.90,
		},
	}

	resolver := NewRuleBasedResolver(rules)
	input := Input{
		Port:     80,
		Protocol: "http",
		Banner:   "Server: Apache/2.4.41 (Ubuntu)",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolver.Resolve(context.Background(), input)
	}
}

// BenchmarkResolverMultipleRules benchmarks resolver with multiple rules (realistic scenario).
func BenchmarkResolverMultipleRules(b *testing.B) {
	// Load all rules from database
	rules, err := LoadRulesFromFile("data/fingerprint_db.yaml")
	if err != nil {
		b.Fatalf("failed to load rules: %v", err)
	}

	resolver := NewRuleBasedResolver(rules)
	input := Input{
		Port:     80,
		Protocol: "http",
		Banner:   "Server: Apache/2.4.41 (Ubuntu)",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolver.Resolve(context.Background(), input)
	}
}

// BenchmarkResolverNoMatch benchmarks resolver when no rules match.
func BenchmarkResolverNoMatch(b *testing.B) {
	rules, err := LoadRulesFromFile("data/fingerprint_db.yaml")
	if err != nil {
		b.Fatalf("failed to load rules: %v", err)
	}

	resolver := NewRuleBasedResolver(rules)
	input := Input{
		Port:     9999,
		Protocol: "unknown",
		Banner:   "JUNK DATA NO MATCH",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolver.Resolve(context.Background(), input)
	}
}

// BenchmarkResolverVersionExtraction benchmarks version extraction performance.
func BenchmarkResolverVersionExtraction(b *testing.B) {
	rules := []StaticRule{
		{
			ID:                "bench.ssh.openssh",
			Protocol:          "ssh",
			Product:           "OpenSSH",
			Vendor:            "OpenBSD",
			Match:             "openssh",
			VersionExtraction: `openssh[_/](\d+\.\d+(?:p\d+)?)`,
			PatternStrength:   0.95,
		},
	}

	resolver := NewRuleBasedResolver(rules)
	input := Input{
		Port:     22,
		Protocol: "ssh",
		Banner:   "SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.5",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolver.Resolve(context.Background(), input)
	}
}

// BenchmarkResolverWithAntiPatterns benchmarks resolver with anti-pattern checks.
func BenchmarkResolverWithAntiPatterns(b *testing.B) {
	rules := []StaticRule{
		{
			ID:                  "bench.http.apache",
			Protocol:            "http",
			Product:             "Apache",
			Vendor:              "Apache",
			Match:               "apache",
			ExcludePatterns:     []string{"nginx", "iis"},
			SoftExcludePatterns: []string{"error", "test"},
			PatternStrength:     0.90,
		},
	}

	resolver := NewRuleBasedResolver(rules)
	input := Input{
		Port:     80,
		Protocol: "http",
		Banner:   "Server: Apache/2.4.41 (Ubuntu)",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolver.Resolve(context.Background(), input)
	}
}

// BenchmarkResolverWithTelemetry benchmarks resolver with telemetry enabled.
func BenchmarkResolverWithTelemetry(b *testing.B) {
	rules := []StaticRule{
		{
			ID:              "bench.http.apache",
			Protocol:        "http",
			Product:         "Apache",
			Vendor:          "Apache",
			Match:           "apache",
			PatternStrength: 0.90,
		},
	}

	resolver := NewRuleBasedResolver(rules)

	// Create temp telemetry file
	tmpFile := b.TempDir() + "/bench-telemetry.jsonl"
	telemetry, err := NewTelemetryWriter(tmpFile)
	if err != nil {
		b.Fatalf("failed to create telemetry: %v", err)
	}
	defer telemetry.Close()

	resolver.SetTelemetry(telemetry)

	input := Input{
		Port:     80,
		Protocol: "http",
		Banner:   "Server: Apache/2.4.41 (Ubuntu)",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolver.Resolve(context.Background(), input)
	}
}

// BenchmarkResolverConcurrent benchmarks resolver with concurrent requests.
func BenchmarkResolverConcurrent(b *testing.B) {
	rules, err := LoadRulesFromFile("data/fingerprint_db.yaml")
	if err != nil {
		b.Fatalf("failed to load rules: %v", err)
	}

	resolver := NewRuleBasedResolver(rules)
	input := Input{
		Port:     80,
		Protocol: "http",
		Banner:   "Server: Apache/2.4.41 (Ubuntu)",
	}

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = resolver.Resolve(context.Background(), input)
		}
	})
}

// BenchmarkValidationRunner benchmarks full validation suite performance.
func BenchmarkValidationRunner(b *testing.B) {
	rules, err := LoadRulesFromFile("data/fingerprint_db.yaml")
	if err != nil {
		b.Fatalf("failed to load rules: %v", err)
	}

	resolver := NewRuleBasedResolver(rules)
	runner, err := NewValidationRunner(resolver, "testdata/validation_dataset.yaml")
	if err != nil {
		b.Fatalf("failed to create runner: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = runner.Run(context.Background())
	}
}

// BenchmarkValidationMetricsCalculation benchmarks metrics calculation only.
func BenchmarkValidationMetricsCalculation(b *testing.B) {
	rules, err := LoadRulesFromFile("data/fingerprint_db.yaml")
	if err != nil {
		b.Fatalf("failed to load rules: %v", err)
	}

	resolver := NewRuleBasedResolver(rules)
	runner, err := NewValidationRunner(resolver, "testdata/validation_dataset.yaml")
	if err != nil {
		b.Fatalf("failed to create runner: %v", err)
	}

	// Run once to get results
	_, results, err := runner.Run(context.Background())
	if err != nil {
		b.Fatalf("failed to run validation: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = runner.calculateMetrics(results)
	}
}

// BenchmarkRulePreparation benchmarks rule compilation and preparation.
func BenchmarkRulePreparation(b *testing.B) {
	rules, err := LoadRulesFromFile("data/fingerprint_db.yaml")
	if err != nil {
		b.Fatalf("failed to load rules: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = prepareRules(rules)
	}
}

// --- Helpers for large dataset generation ---

// generateLargeDataset creates n synthetic validation cases that still parse via loader.
func generateLargeDataset(n int) *ValidationDataset {
	ds := &ValidationDataset{
		TruePositives: make([]ValidationTestCase, 0, n),
		TrueNegatives: nil,
		EdgeCases:     nil,
	}
	for i := 0; i < n; i++ {
		ds.TruePositives = append(ds.TruePositives, ValidationTestCase{
			Protocol:        "http",
			Port:            80,
			Banner:          "Server: Apache/2.4.41 (Ubuntu)",
			ExpectedProduct: "Apache",
			ExpectedVendor:  "Apache",
			ExpectedVersion: "2.4.41",
			Description:     "synthetic-apache-case-" + itoa(i),
		})
	}
	return ds
}

// saveDataset writes the dataset to a temporary YAML file path using yaml.Marshal schema from validation.go
func saveDataset(ds *ValidationDataset, path string) error {
	// Reuse the YAML structure by marshaling via yaml from validation.go by writing a small wrapper runner
	// Since validation.NewValidationRunner reads YAML directly, write the file with the same shape.
	// Implement minimal writer here to avoid test import cycles.
	type fileFmt struct {
		TruePositives []ValidationTestCase `yaml:"true_positives"`
		TrueNegatives []ValidationTestCase `yaml:"true_negatives"`
		EdgeCases     []ValidationTestCase `yaml:"edge_cases"`
	}
	data, err := yamlMarshal(fileFmt{TruePositives: ds.TruePositives, TrueNegatives: ds.TrueNegatives, EdgeCases: ds.EdgeCases})
	if err != nil {
		return err
	}
	return osWriteFile(path, data, 0o644)
}

// itoa is a tiny int->string helper to avoid fmt import in benchmarks.
func itoa(i int) string { return strconv.Itoa(i) }

// ptr helper
// lightweight wrappers to avoid importing packages directly in benchmarks
var (
	yamlMarshal = yaml.Marshal
	osWriteFile = os.WriteFile
)

// BenchmarkValidationRunnerLargeDataset tests worst-case performance on large synthetic datasets.
func BenchmarkValidationRunnerLargeDataset(b *testing.B) {
	rules, err := LoadRulesFromFile("data/fingerprint_db.yaml")
	if err != nil {
		b.Fatalf("failed to load rules: %v", err)
	}

	resolver := NewRuleBasedResolver(rules)

	// Generate and persist large dataset (default 1000 cases)
	ds := generateLargeDataset(1000)
	dir := b.TempDir()
	tmpFile := filepath.Join(dir, "large_dataset.yaml")
	if err := saveDataset(ds, tmpFile); err != nil {
		b.Fatalf("failed to save dataset: %v", err)
	}

	runner, err := NewValidationRunner(resolver, tmpFile)
	if err != nil {
		b.Fatalf("failed to create runner: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = runner.Run(context.Background())
	}
}

// BenchmarkValidationRunnerLargeDataset10k for extreme scaling checks (skipped in short mode)
func BenchmarkValidationRunnerLargeDataset10k(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping 10k dataset in short mode")
	}

	rules, err := LoadRulesFromFile("data/fingerprint_db.yaml")
	if err != nil {
		b.Fatalf("failed to load rules: %v", err)
	}
	resolver := NewRuleBasedResolver(rules)

	ds := generateLargeDataset(10000)
	dir := b.TempDir()
	tmpFile := filepath.Join(dir, "large_dataset_10k.yaml")
	if err := saveDataset(ds, tmpFile); err != nil {
		b.Fatalf("failed to save dataset: %v", err)
	}

	runner, err := NewValidationRunner(resolver, tmpFile)
	if err != nil {
		b.Fatalf("failed to create runner: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = runner.Run(context.Background())
	}
}

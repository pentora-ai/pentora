package fingerprint

import (
	"context"
	"os"
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = prepareRules(rules)
	}
}

// Memory-focused benchmarks
func BenchmarkResolverMemory(b *testing.B) {
	b.ReportAllocs()
	rules, err := LoadRulesFromFile("data/fingerprint_db.yaml")
	if err != nil {
		b.Fatalf("failed to load rules: %v", err)
	}
	resolver := NewRuleBasedResolver(rules)
	input := Input{Port: 80, Protocol: "http", Banner: "Server: Apache/2.4.41 (Ubuntu)"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = resolver.Resolve(context.Background(), input)
	}
}

func BenchmarkValidationRunnerMemory(b *testing.B) {
	b.ReportAllocs()
	rules, err := LoadRulesFromFile("data/fingerprint_db.yaml")
	if err != nil {
		b.Fatalf("failed to load rules: %v", err)
	}
	resolver := NewRuleBasedResolver(rules)
	runner, err := NewValidationRunner(resolver, "testdata/validation_dataset.yaml")
	if err != nil {
		b.Fatalf("failed to create runner: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = runner.Run(context.Background())
	}
}

// Large dataset helpers and benchmarks
func generateLargeDataset(n int) []ValidationTestCase {
	cases := make([]ValidationTestCase, 0, n)
	protos := []struct{ proto, banner string }{
		{"http", "Server: Apache/2.4.41 (Ubuntu)"},
		{"http", "Server: nginx/1.21.6"},
		{"ssh", "SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.5"},
		{"mysql", "5.7.31-log"},
		{"redis", "+PONG"},
	}
	for i := 0; i < n; i++ {
		p := protos[i%len(protos)]
		cases = append(cases, ValidationTestCase{
			Protocol:        p.proto,
			Port:            80,
			Banner:          p.banner,
			ExpectedProduct: "",
			Description:     "auto-" + itoa(i),
		})
	}
	return cases
}

func BenchmarkValidationRunnerLargeDataset(b *testing.B) {
	b.ReportAllocs()
	rules, err := LoadRulesFromFile("data/fingerprint_db.yaml")
	if err != nil {
		b.Fatalf("failed to load rules: %v", err)
	}
	resolver := NewRuleBasedResolver(rules)
	ds := generateLargeDataset(1000)
	tmp := b.TempDir() + "/large_ds_1k.yaml"
	if err := saveValidationDatasetCompat(tmp, ds); err != nil {
		b.Fatalf("save dataset: %v", err)
	}
	runner, err := NewValidationRunner(resolver, tmp)
	if err != nil {
		b.Fatalf("runner: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = runner.Run(context.Background())
	}
}

func BenchmarkValidationRunnerLargeDataset5k(b *testing.B) {
	b.ReportAllocs()
	rules, err := LoadRulesFromFile("data/fingerprint_db.yaml")
	if err != nil {
		b.Fatalf("failed to load rules: %v", err)
	}
	resolver := NewRuleBasedResolver(rules)
	ds := generateLargeDataset(5000)
	tmp := b.TempDir() + "/large_ds_5k.yaml"
	if err := saveValidationDatasetCompat(tmp, ds); err != nil {
		b.Fatalf("save dataset: %v", err)
	}
	runner, err := NewValidationRunner(resolver, tmp)
	if err != nil {
		b.Fatalf("runner: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = runner.Run(context.Background())
	}
}

func BenchmarkValidationRunnerLargeDataset10k(b *testing.B) {
	b.ReportAllocs()
	rules, err := LoadRulesFromFile("data/fingerprint_db.yaml")
	if err != nil {
		b.Fatalf("failed to load rules: %v", err)
	}
	resolver := NewRuleBasedResolver(rules)
	ds := generateLargeDataset(10000)
	tmp := b.TempDir() + "/large_ds_10k.yaml"
	if err := saveValidationDatasetCompat(tmp, ds); err != nil {
		b.Fatalf("save dataset: %v", err)
	}
	runner, err := NewValidationRunner(resolver, tmp)
	if err != nil {
		b.Fatalf("runner: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = runner.Run(context.Background())
	}
}

// saveValidationDatasetCompat writes a minimal dataset YAML with true_positives only.
func saveValidationDatasetCompat(path string, cases []ValidationTestCase) error {
	// Minimal YAML serialization using gopkg.in/yaml.v3
	type ds struct {
		TruePositives []ValidationTestCase `yaml:"true_positives"`
	}
	d := ds{TruePositives: cases}
	data, err := yaml.Marshal(&d)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Small int to string helper without fmt
// reuse itoa from validation_runner_test.go to avoid redeclaration

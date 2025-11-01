package engine

import (
	"context"
	"testing"
)

// helper to register a minimal fake module with given meta
func fakeFactory(meta ModuleMetadata) ModuleFactory {
	return func() Module {
		return &fakeModule{meta: meta}
	}
}

type fakeModule struct{ meta ModuleMetadata }

func (f *fakeModule) Metadata() ModuleMetadata                  { return f.meta }
func (f *fakeModule) Init(string, map[string]interface{}) error { return nil }
func (f *fakeModule) Execute(_ context.Context, _ map[string]interface{}, _ chan<- ModuleOutput) error {
	return nil
}

// Test PlanDAG basic path with default intent and module selection
func TestPlanner_PlanDAG_DefaultProfile_SelectsAndConfigures(t *testing.T) {
	// discovery depends on targets only (implicit), scan consumes discovery output, reporter no deps
	discoveryMeta := ModuleMetadata{
		Name: "tcp-port-discovery", Type: DiscoveryModuleType,
		Consumes:     nil,
		Produces:     []DataContractEntry{{Key: "discovery.open_tcp_ports"}},
		ConfigSchema: map[string]ParameterDefinition{"timeout": {Default: "1s"}},
	}
	scanMeta := ModuleMetadata{
		Name: "banner-grabber", Type: ScanModuleType,
		Consumes: []DataContractEntry{{Key: "discovery.open_tcp_ports"}},
		Produces: []DataContractEntry{{Key: "service.banner.tcp"}},
		ConfigSchema: map[string]ParameterDefinition{
			"read_timeout":    {Default: "3s"},
			"connect_timeout": {Default: "2s"},
		},
		Tags: []string{"scan"},
	}
	parseMeta := ModuleMetadata{
		Name: "http-parser", Type: ParseModuleType,
		Consumes:     []DataContractEntry{{Key: "service.banner.tcp", IsOptional: true}},
		Produces:     []DataContractEntry{{Key: "service.http.details"}},
		ConfigSchema: map[string]ParameterDefinition{},
		Tags:         []string{"parse"},
	}
	reporterMeta := ModuleMetadata{
		Name: "json-reporter", Type: ReportingModuleType,
		ConfigSchema: map[string]ParameterDefinition{},
		Tags:         []string{"report"},
	}

	registry := map[string]ModuleFactory{
		discoveryMeta.Name: fakeFactory(discoveryMeta),
		scanMeta.Name:      fakeFactory(scanMeta),
		parseMeta.Name:     fakeFactory(parseMeta),
		reporterMeta.Name:  fakeFactory(reporterMeta),
	}

	planner, err := NewDAGPlanner(registry)
	if err != nil {
		t.Fatalf("NewDAGPlanner error: %v", err)
	}

	intent := ScanIntent{Targets: []string{"127.0.0.1"}, CustomTimeout: "10s"}
	dag, err := planner.PlanDAG(intent)
	if err != nil {
		t.Fatalf("PlanDAG error: %v", err)
	}
	if dag == nil || len(dag.Nodes) == 0 {
		t.Fatalf("expected nodes in DAG, got %+v", dag)
	}

	// Verify unique instance IDs and configs applied
	names := map[string]bool{}
	hasDiscovery, hasScan, hasParse, hasReporter := false, false, false, false
	var scanCfg map[string]interface{}
	for _, n := range dag.Nodes {
		if names[n.InstanceID] {
			t.Fatalf("duplicate instance id: %s", n.InstanceID)
		}
		names[n.InstanceID] = true
		switch n.ModuleType {
		case discoveryMeta.Name:
			hasDiscovery = true
		case scanMeta.Name:
			hasScan = true
			scanCfg = n.Config
		case parseMeta.Name:
			hasParse = true
		case reporterMeta.Name:
			hasReporter = true
		}
	}
	if !hasDiscovery || !hasScan || !hasParse || !hasReporter {
		t.Fatalf("expected discovery, scan, parse, reporter: got D=%v S=%v P=%v R=%v", hasDiscovery, hasScan, hasParse, hasReporter)
	}
	// From planner change: when CustomTimeout set, banner-grabber gets read/connect timeouts
	if scanCfg == nil {
		t.Fatalf("scan node config missing")
	}
	if scanCfg["read_timeout"] != "10s" || scanCfg["connect_timeout"] != "10s" {
		t.Fatalf("expected scan timeouts to be 10s, got read=%v connect=%v", scanCfg["read_timeout"], scanCfg["connect_timeout"])
	}
}

func TestPlanner_configureModule_AppliesCustoms(t *testing.T) {
	planner, _ := NewDAGPlanner(nil)
	// tcp-port-discovery gets ports and timeout from intent
	meta := ModuleMetadata{Name: "tcp-port-discovery", ConfigSchema: map[string]ParameterDefinition{"ports": {Default: nil}, "timeout": {Default: nil}}}
	cfg := planner.configureModule(meta, ScanIntent{CustomPortConfig: "80,443", CustomTimeout: "5s"})
	if cfg["timeout"] != "5s" {
		t.Fatalf("expected discovery timeout 5s, got %v", cfg["timeout"])
	}

	// banner-grabber gets propagated timeouts
	scanMeta := ModuleMetadata{Name: "banner-grabber", ConfigSchema: map[string]ParameterDefinition{"read_timeout": {Default: "3s"}, "connect_timeout": {Default: "2s"}}}
	sc := planner.configureModule(scanMeta, ScanIntent{CustomTimeout: "7s"})
	if sc["read_timeout"] != "7s" || sc["connect_timeout"] != "7s" {
		t.Fatalf("expected scan timeouts 7s, got read=%v connect=%v", sc["read_timeout"], sc["connect_timeout"])
	}
}

func TestPlanner_generateInstanceID_Unique(t *testing.T) {
	planner, _ := NewDAGPlanner(nil)
	existing := map[string]DAGNodeConfig{"banner_grabber": {InstanceID: "banner_grabber"}}
	id := planner.generateInstanceID("banner-grabber", existing)
	if id == "banner_grabber" {
		t.Fatalf("expected unique id not equal to existing, got %s", id)
	}
}

// Test filterHostDiscoveryModules filters ICMP but preserves port scanners
func TestPlanner_filterHostDiscoveryModules(t *testing.T) {
	planner, _ := NewDAGPlanner(nil)

	icmpMeta := ModuleMetadata{Name: "icmp-ping-discovery", Type: DiscoveryModuleType}
	tcpPortMeta := ModuleMetadata{Name: "tcp-port-discovery", Type: DiscoveryModuleType}
	udpPortMeta := ModuleMetadata{Name: "udp-port-discovery", Type: DiscoveryModuleType}
	scanMeta := ModuleMetadata{Name: "banner-grabber", Type: ScanModuleType}

	factories := []ModuleFactory{
		fakeFactory(icmpMeta),
		fakeFactory(tcpPortMeta),
		fakeFactory(udpPortMeta),
		fakeFactory(scanMeta),
	}

	filtered := planner.filterHostDiscoveryModules(factories)

	// Should have 3 modules: tcp-port-discovery, udp-port-discovery, banner-grabber
	// ICMP should be filtered out
	if len(filtered) != 3 {
		t.Fatalf("expected 3 modules after filtering, got %d", len(filtered))
	}

	hasICMP, hasTCPPort, hasUDPPort, hasScan := false, false, false, false
	for _, factory := range filtered {
		meta := factory().Metadata()
		switch meta.Name {
		case "icmp-ping-discovery":
			hasICMP = true
		case "tcp-port-discovery":
			hasTCPPort = true
		case "udp-port-discovery":
			hasUDPPort = true
		case "banner-grabber":
			hasScan = true
		}
	}

	if hasICMP {
		t.Fatal("ICMP ping should be filtered out")
	}
	if !hasTCPPort {
		t.Fatal("TCP port discovery should be preserved")
	}
	if !hasUDPPort {
		t.Fatal("UDP port discovery should be preserved")
	}
	if !hasScan {
		t.Fatal("Scanner module should be preserved")
	}
}

// Test selectModulesByProfile with SkipDiscovery
func TestPlanner_selectModulesByProfile_SkipDiscovery(t *testing.T) {
	icmpMeta := ModuleMetadata{Name: "icmp-ping-discovery", Type: DiscoveryModuleType}
	tcpPortMeta := ModuleMetadata{Name: "tcp-port-discovery", Type: DiscoveryModuleType}
	scanMeta := ModuleMetadata{Name: "banner-grabber", Type: ScanModuleType}
	parseMeta := ModuleMetadata{Name: "http-parser", Type: ParseModuleType}

	registry := map[string]ModuleFactory{
		icmpMeta.Name:    fakeFactory(icmpMeta),
		tcpPortMeta.Name: fakeFactory(tcpPortMeta),
		scanMeta.Name:    fakeFactory(scanMeta),
		parseMeta.Name:   fakeFactory(parseMeta),
	}

	planner, _ := NewDAGPlanner(registry)

	// Test with SkipDiscovery=false (normal)
	intent := ScanIntent{Targets: []string{"127.0.0.1"}, SkipDiscovery: false}
	selected := planner.selectModulesForIntent(intent)

	// Should include tcp-port-discovery and parse modules
	hasTCPPort := false
	for _, factory := range selected {
		if factory().Metadata().Name == "tcp-port-discovery" {
			hasTCPPort = true
			break
		}
	}
	if !hasTCPPort {
		t.Fatal("expected tcp-port-discovery when SkipDiscovery=false")
	}

	// Test with SkipDiscovery=true
	intentSkip := ScanIntent{Targets: []string{"127.0.0.1"}, SkipDiscovery: true}
	selectedSkip := planner.selectModulesForIntent(intentSkip)

	// Should still include tcp-port-discovery (port scanner, not host discovery)
	hasTCPPortSkip := false
	hasICMP := false
	for _, factory := range selectedSkip {
		meta := factory().Metadata()
		if meta.Name == "tcp-port-discovery" {
			hasTCPPortSkip = true
		}
		if meta.Name == "icmp-ping-discovery" {
			hasICMP = true
		}
	}

	if !hasTCPPortSkip {
		t.Fatal("tcp-port-discovery should be preserved with SkipDiscovery=true")
	}
	if hasICMP {
		t.Fatal("ICMP should be filtered out with SkipDiscovery=true")
	}
}

// Test initializeDataKeys injects discovery.live_hosts when SkipDiscovery=true
func TestPlanner_initializeDataKeys_SkipDiscovery(t *testing.T) {
	planner, _ := NewDAGPlanner(nil)

	// Without SkipDiscovery
	intent := ScanIntent{Targets: []string{"127.0.0.1"}, SkipDiscovery: false}
	keys := planner.initializeDataKeys(intent)

	if _, found := keys["config.targets"]; !found {
		t.Fatal("expected config.targets to be initialized")
	}
	if _, found := keys["discovery.live_hosts"]; found {
		t.Fatal("discovery.live_hosts should NOT be initialized when SkipDiscovery=false")
	}

	// With SkipDiscovery
	intentSkip := ScanIntent{Targets: []string{"127.0.0.1"}, SkipDiscovery: true}
	keysSkip := planner.initializeDataKeys(intentSkip)

	if _, found := keysSkip["config.targets"]; !found {
		t.Fatal("expected config.targets to be initialized")
	}
	if _, found := keysSkip["discovery.live_hosts"]; !found {
		t.Fatal("discovery.live_hosts should be initialized when SkipDiscovery=true")
	}
}

// Test different profiles select correct modules
func TestPlanner_selectModulesByProfile_Profiles(t *testing.T) {
	discoveryMeta := ModuleMetadata{Name: "icmp-ping", Type: DiscoveryModuleType}
	scanMeta := ModuleMetadata{Name: "scanner", Type: ScanModuleType, Tags: []string{"quick"}}
	evalMeta := ModuleMetadata{Name: "evaluator", Type: EvaluationModuleType}

	registry := map[string]ModuleFactory{
		discoveryMeta.Name: fakeFactory(discoveryMeta),
		scanMeta.Name:      fakeFactory(scanMeta),
		evalMeta.Name:      fakeFactory(evalMeta),
	}

	planner, _ := NewDAGPlanner(registry)

	// Test DiscoveryOnly
	intentDiscovery := ScanIntent{DiscoveryOnly: true}
	selected := planner.selectModulesByProfile(intentDiscovery)
	hasDiscovery := false
	for _, factory := range selected {
		if factory().Metadata().Type == DiscoveryModuleType {
			hasDiscovery = true
		}
	}
	if !hasDiscovery {
		t.Fatal("DiscoveryOnly should select discovery modules")
	}

	// Test quick_discovery profile
	intentQuick := ScanIntent{Profile: "quick_discovery"}
	selectedQuick := planner.selectModulesByProfile(intentQuick)
	if len(selectedQuick) == 0 {
		t.Fatal("quick_discovery should select modules")
	}

	// Test full_scan with EnableVulnChecks
	intentFull := ScanIntent{Profile: "full_scan", EnableVulnChecks: true}
	selectedFull := planner.selectModulesByProfile(intentFull)
	hasEval := false
	for _, factory := range selectedFull {
		if factory().Metadata().Type == EvaluationModuleType {
			hasEval = true
		}
	}
	if !hasEval {
		t.Fatal("full_scan with EnableVulnChecks should include evaluation modules")
	}
}

// Test matchesTags covers all scenarios
func TestPlanner_matchesTags(t *testing.T) {
	planner, _ := NewDAGPlanner(nil)

	tests := []struct {
		name        string
		moduleTags  []string
		includeTags []string
		excludeTags []string
		want        bool
	}{
		{
			name:        "no filters - should match",
			moduleTags:  []string{"tag1", "tag2"},
			includeTags: nil,
			excludeTags: nil,
			want:        true,
		},
		{
			name:        "exclude tag present - should not match",
			moduleTags:  []string{"tag1", "intrusive"},
			includeTags: nil,
			excludeTags: []string{"intrusive"},
			want:        false,
		},
		{
			name:        "exclude tag not present - should match",
			moduleTags:  []string{"tag1", "tag2"},
			includeTags: nil,
			excludeTags: []string{"intrusive"},
			want:        true,
		},
		{
			name:        "include tag present - should match",
			moduleTags:  []string{"tag1", "quick"},
			includeTags: []string{"quick"},
			excludeTags: nil,
			want:        true,
		},
		{
			name:        "include tag not present - should not match",
			moduleTags:  []string{"tag1", "tag2"},
			includeTags: []string{"quick"},
			excludeTags: nil,
			want:        false,
		},
		{
			name:        "both include and exclude, include present - should match",
			moduleTags:  []string{"tag1", "quick"},
			includeTags: []string{"quick"},
			excludeTags: []string{"intrusive"},
			want:        true,
		},
		{
			name:        "both include and exclude, exclude present - should not match",
			moduleTags:  []string{"tag1", "quick", "intrusive"},
			includeTags: []string{"quick"},
			excludeTags: []string{"intrusive"},
			want:        false,
		},
		{
			name:        "multiple include tags, one matches - should match",
			moduleTags:  []string{"tag1", "quick"},
			includeTags: []string{"fast", "quick", "speed"},
			excludeTags: nil,
			want:        true,
		},
		{
			name:        "multiple exclude tags, one matches - should not match",
			moduleTags:  []string{"tag1", "slow"},
			includeTags: nil,
			excludeTags: []string{"intrusive", "slow", "heavy"},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := planner.matchesTags(tt.moduleTags, tt.includeTags, tt.excludeTags)
			if got != tt.want {
				t.Errorf("matchesTags() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Test logUnprocessedModules logs unprocessed modules with unmet dependencies
func TestPlanner_logUnprocessedModules(t *testing.T) {
	planner, _ := NewDAGPlanner(nil)

	// Create modules with dependencies
	module1Meta := ModuleMetadata{
		Name: "module1",
		Consumes: []DataContractEntry{
			{Key: "missing.key1"},
			{Key: "missing.key2"},
		},
	}
	module2Meta := ModuleMetadata{
		Name: "module2",
		Consumes: []DataContractEntry{
			{Key: "available.key"},
		},
	}

	candidateModules := []ModuleFactory{
		fakeFactory(module1Meta),
		fakeFactory(module2Meta),
	}

	// Only module2 was added (module1 has unmet dependencies)
	moduleTypesAddedToDAG := map[string]bool{
		"module2": true,
	}

	availableDataKeys := map[string]string{
		"available.key": "some_module",
	}

	// This should log module1 with unmet dependencies (missing.key1, missing.key2)
	// Test passes if no panic occurs (function is mainly for logging)
	planner.logUnprocessedModules(candidateModules, moduleTypesAddedToDAG, availableDataKeys)

	// Test case where all modules were added (no logging)
	allAddedModules := map[string]bool{
		"module1": true,
		"module2": true,
	}
	planner.logUnprocessedModules(candidateModules, allAddedModules, availableDataKeys)
}

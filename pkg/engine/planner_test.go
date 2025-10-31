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

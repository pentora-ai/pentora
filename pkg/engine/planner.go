// pkg/engine/planner.go
package engine

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	// Module type names used throughout the planner
	moduleTypeTCPPortDiscovery  = "tcp-port-discovery"
	moduleTypeUDPPortDiscovery  = "udp-port-discovery"
	moduleTypeICMPPingDiscovery = "icmp-ping-discovery"
)

// ScanIntent represents the user's high-level goal for the scan.
type ScanIntent struct {
	Targets          []string
	Profile          string // e.g., "quick_discovery", "full_scan", "vuln_scan"
	Level            string // e.g., "light", "comprehensive", "intrusive"
	IncludeTags      []string
	ExcludeTags      []string
	EnableVulnChecks bool
	// ... other parameters like custom ports, timeouts from CLI/API
	CustomPortConfig string // Example: "80,443,1000-1024"
	CustomTimeout    string // Example: "1s"
	EnablePing       bool   // Whether to enable ICMP ping discovery
	PingCount        int    // Number of ICMP echo requests to send
	AllowLoopback    bool   // Whether to allow scanning loopback addresses
	Concurrency      int    // Number of concurrent modules to run
	DiscoveryOnly    bool
	SkipDiscovery    bool
}

// DAGPlanner is responsible for automatically constructing a DAGDefinition based on scan intent and module metadata.
type DAGPlanner struct {
	moduleRegistry map[string]ModuleFactory // Access to all registered module factories and their metadata
	logger         zerolog.Logger
}

// NewDAGPlanner creates a new DAGPlanner.
func NewDAGPlanner(registry map[string]ModuleFactory) (*DAGPlanner, error) {
	return &DAGPlanner{
		moduleRegistry: registry,
		logger:         log.With().Str("component", "DAGPlanner").Logger(),
	}, nil
}

// initializeDataKeys sets up initial data keys for DAG planning based on intent.
func (p *DAGPlanner) initializeDataKeys(intent ScanIntent) map[string]string {
	availableDataKeys := make(map[string]string)
	if len(intent.Targets) > 0 {
		availableDataKeys["config.targets"] = "initial_input"

		// If skipping discovery, treat all targets as live hosts
		if intent.SkipDiscovery {
			availableDataKeys["discovery.live_hosts"] = "initial_input"
			p.logger.Debug().Msg("SkipDiscovery enabled: treating all targets as live hosts")
		}

		p.logger.Debug().Interface("initial_keys", availableDataKeys).Msg("Initial available data keys")
	}
	return availableDataKeys
}

// checkModuleDependencies checks if all required dependencies for a module are met.
func (p *DAGPlanner) checkModuleDependencies(
	meta ModuleMetadata,
	availableDataKeys map[string]string,
) bool {
	if len(meta.Consumes) == 0 {
		return true
	}

	for _, consumedContract := range meta.Consumes {
		consumedKeyString := consumedContract.Key
		if _, keyIsAvailable := availableDataKeys[consumedKeyString]; !keyIsAvailable && !consumedContract.IsOptional {
			p.logger.Trace().Str("module", meta.Name).Str("missing_key", consumedKeyString).
				Msg("Dependency key not yet available for module")
			return false
		}
	}
	return true
}

// addModuleToDAG adds a module to the DAG and updates tracking structures.
func (p *DAGPlanner) addModuleToDAG(
	meta ModuleMetadata,
	intent ScanIntent,
	dagDef *DAGDefinition,
	dagNodeConfigs map[string]DAGNodeConfig,
	availableDataKeys map[string]string,
) {
	instanceID := p.generateInstanceID(meta.Name, dagNodeConfigs)

	nodeCfg := DAGNodeConfig{
		InstanceID: instanceID,
		ModuleType: meta.Name,
		Config:     p.configureModule(meta, intent),
	}

	dagDef.Nodes = append(dagDef.Nodes, nodeCfg)
	dagNodeConfigs[instanceID] = dagDef.Nodes[len(dagDef.Nodes)-1]

	p.logger.Debug().Str("module", meta.Name).Str("instance_id", instanceID).Msg("Added module to DAG")

	// Register produced data keys
	for _, producedContract := range meta.Produces {
		producedKey := producedContract.Key
		if existingProducer, found := availableDataKeys[producedKey]; found && existingProducer != "initial_input" {
			p.logger.Warn().Str("data_key", producedKey).Str("new_producer", instanceID).
				Str("existing_producer", existingProducer).
				Msg("DataKey already produced by another module. Overwriting producer.")
		}
		availableDataKeys[producedKey] = instanceID
		p.logger.Trace().Str("module_producer", meta.Name).Str("instance_id_producer", instanceID).
			Str("produced_key", producedKey).Msg("Marked key as available")
	}
}

// buildDAGIteratively builds the DAG by iteratively adding modules whose dependencies are met.
func (p *DAGPlanner) buildDAGIteratively(
	candidateModules []ModuleFactory,
	intent ScanIntent,
	dagDef *DAGDefinition,
	availableDataKeys map[string]string,
) map[string]bool {
	dagNodeConfigs := make(map[string]DAGNodeConfig)
	moduleTypesAddedToDAG := make(map[string]bool)

	for {
		addedInThisIteration := 0

		for _, modFactory := range candidateModules {
			tempMod := modFactory()
			meta := tempMod.Metadata()

			if moduleTypesAddedToDAG[meta.Name] {
				continue
			}

			if p.checkModuleDependencies(meta, availableDataKeys) {
				p.addModuleToDAG(meta, intent, dagDef, dagNodeConfigs, availableDataKeys)
				moduleTypesAddedToDAG[meta.Name] = true
				addedInThisIteration++
			}
		}

		if addedInThisIteration == 0 {
			p.logger.Debug().Int("total_dag_nodes", len(dagDef.Nodes)).
				Msg("No more modules added in this planning iteration. Loop will terminate.")
			break
		}
		p.logger.Debug().Int("added_this_iteration", addedInThisIteration).
			Int("total_dag_nodes", len(dagDef.Nodes)).
			Msg("Completed an iteration of DAG planning.")
	}

	return moduleTypesAddedToDAG
}

// logUnprocessedModules logs modules that couldn't be added due to unmet dependencies.
func (p *DAGPlanner) logUnprocessedModules(
	candidateModules []ModuleFactory,
	moduleTypesAddedToDAG map[string]bool,
	availableDataKeys map[string]string,
) {
	if len(moduleTypesAddedToDAG) >= len(candidateModules) {
		return
	}

	p.logger.Warn().Msg("Not all candidate modules selected by intent could be added to the DAG. Logging unprocessed modules and their potential unmet dependencies:")
	for _, modFactory := range candidateModules {
		meta := modFactory().Metadata()
		if !moduleTypesAddedToDAG[meta.Name] {
			unmetDependencies := []string{}
			for _, consumedContract := range meta.Consumes {
				consumedKey := consumedContract.Key
				if _, found := availableDataKeys[consumedKey]; !found {
					unmetDependencies = append(unmetDependencies, consumedKey)
				}
			}
			p.logger.Warn().Str("module", meta.Name).Strs("unmet_dependencies", unmetDependencies).
				Msg("Unprocessed candidate module")
		}
	}
}

// PlanDAG attempts to create a DAGDefinition based on the provided scan intent.
func (p *DAGPlanner) PlanDAG(intent ScanIntent) (*DAGDefinition, error) {
	p.logger.Info().Interface("intent", intent).Msg("Planning DAG based on scan intent")

	dagDef := &DAGDefinition{
		Name:        fmt.Sprintf("AutoPlannedDAG_%s", intent.Profile_or_Level_or_Default()),
		Description: fmt.Sprintf("Automatically planned DAG for intent: %s", intent.Profile_or_Level_or_Default()),
		Nodes:       []DAGNodeConfig{},
	}

	candidateModules := p.selectModulesForIntent(intent)
	if len(candidateModules) == 0 {
		p.logger.Error().Msg("No suitable modules found for the given scan intent")
		return nil, fmt.Errorf("no suitable modules found for the given scan intent")
	}
	p.logger.Debug().Int("count", len(candidateModules)).Msg("Candidate modules selected")

	// Initialize available data keys
	availableDataKeys := p.initializeDataKeys(intent)

	// Build DAG iteratively
	moduleTypesAddedToDAG := p.buildDAGIteratively(candidateModules, intent, dagDef, availableDataKeys)

	// Log unprocessed modules if any
	p.logUnprocessedModules(candidateModules, moduleTypesAddedToDAG, availableDataKeys)

	// Validate DAG is not empty
	if len(dagDef.Nodes) == 0 {
		if len(candidateModules) > 0 {
			p.logger.Error().Msg("Failed to plan any nodes for the DAG, though candidates were selected. Check dependencies or initial inputs.")
			return nil, fmt.Errorf("failed to plan any nodes for the DAG, though candidates were selected. Check dependencies or initial inputs")
		}
		p.logger.Error().Msg("No candidate modules selected and no DAG nodes planned")
		return nil, fmt.Errorf("no candidate modules selected and no DAG nodes planned")
	}

	p.logger.Info().Int("nodes_in_dag", len(dagDef.Nodes)).Msg("DAG planning complete")
	return dagDef, nil
}

// filterHostDiscoveryModules removes host discovery modules when SkipDiscovery=true.
// Only filters ICMP ping modules, preserves port scanners (tcp-port-discovery, udp-port-discovery).
func (p *DAGPlanner) filterHostDiscoveryModules(selected []ModuleFactory) []ModuleFactory {
	filtered := selected[:0]
	filteredCount := 0
	for _, factory := range selected {
		meta := factory().Metadata()
		// Only filter host discovery modules (ICMP ping), NOT port scanning modules
		if meta.Type == DiscoveryModuleType &&
			meta.Name != moduleTypeTCPPortDiscovery &&
			meta.Name != moduleTypeUDPPortDiscovery {
			filteredCount++
			continue
		}
		filtered = append(filtered, factory)
	}
	if filteredCount > 0 {
		p.logger.Debug().Int("filtered_modules", filteredCount).
			Msg("Filtered host discovery modules due to SkipDiscovery")
	}
	return filtered
}

// selectModulesByType filters modules by type and tags from the registry.
func (p *DAGPlanner) selectModulesByType(
	moduleTypes []ModuleType,
	intent ScanIntent,
	logMessage string,
) []ModuleFactory {
	var selected []ModuleFactory
	for name, factory := range p.moduleRegistry {
		meta := factory().Metadata()
		for _, mType := range moduleTypes {
			if meta.Type == mType && p.matchesTags(meta.Tags, intent.IncludeTags, intent.ExcludeTags) {
				selected = append(selected, factory)
				p.logger.Debug().Str("module", name).Msg(logMessage)
				break
			}
		}
	}
	return selected
}

// selectDiscoveryModules selects only discovery modules (for DiscoveryOnly mode).
func (p *DAGPlanner) selectDiscoveryModules(intent ScanIntent) []ModuleFactory {
	return p.selectModulesByType(
		[]ModuleType{DiscoveryModuleType},
		intent,
		"Selected module for discovery-only run",
	)
}

// selectQuickDiscoveryModules selects modules for quick_discovery/light profile.
func (p *DAGPlanner) selectQuickDiscoveryModules(intent ScanIntent) []ModuleFactory {
	var selected []ModuleFactory
	for name, factory := range p.moduleRegistry {
		meta := factory().Metadata()
		if (meta.Type == DiscoveryModuleType ||
			(containsTag(meta.Tags, "quick") && meta.Type == ScanModuleType)) &&
			p.matchesTags(meta.Tags, intent.IncludeTags, intent.ExcludeTags) {
			selected = append(selected, factory)
			p.logger.Debug().Str("module", name).Msg("Selected module for quick_discovery/light profile")
		}
	}
	return selected
}

// selectFullScanModules selects modules for full_scan/comprehensive profile.
func (p *DAGPlanner) selectFullScanModules(intent ScanIntent) []ModuleFactory {
	var selected []ModuleFactory
	for name, factory := range p.moduleRegistry {
		meta := factory().Metadata()
		// Include Discovery, Scan, Parse, Reporting, and optionally Evaluation
		includeModule := meta.Type == DiscoveryModuleType ||
			meta.Type == ScanModuleType ||
			meta.Type == ParseModuleType ||
			meta.Type == ReportingModuleType ||
			(intent.EnableVulnChecks && meta.Type == EvaluationModuleType)

		if includeModule && p.matchesTags(meta.Tags, intent.IncludeTags, intent.ExcludeTags) {
			selected = append(selected, factory)
			p.logger.Debug().Str("module", name).Msg("Selected module for full_scan/comprehensive profile")
		}
	}
	return selected
}

// selectDefaultModules selects modules for default profile (discovery + scan, optionally vuln).
func (p *DAGPlanner) selectDefaultModules(intent ScanIntent) []ModuleFactory {
	var selected []ModuleFactory

	// Select discovery and scan modules (non-intrusive)
	for name, factory := range p.moduleRegistry {
		meta := factory().Metadata()
		if (meta.Type == DiscoveryModuleType || meta.Type == ScanModuleType) &&
			!containsTag(meta.Tags, "intrusive") &&
			p.matchesTags(meta.Tags, intent.IncludeTags, intent.ExcludeTags) {
			selected = append(selected, factory)
			p.logger.Debug().Str("module", name).Msg("Selected module for default profile")
		}
	}

	// Add evaluation modules if vuln checks enabled
	if intent.EnableVulnChecks {
		for name, factory := range p.moduleRegistry {
			meta := factory().Metadata()
			if meta.Type == EvaluationModuleType &&
				p.matchesTags(meta.Tags, intent.IncludeTags, intent.ExcludeTags) {
				selected = append(selected, factory)
				p.logger.Debug().Str("module", name).Msg("Selected evaluation module for vuln-enabled default profile")
			}
		}
	}

	return selected
}

// selectModulesByProfile selects modules based on intent profile/level.
func (p *DAGPlanner) selectModulesByProfile(intent ScanIntent) []ModuleFactory {
	if intent.DiscoveryOnly {
		return p.selectDiscoveryModules(intent)
	}
	if intent.Profile == "quick_discovery" || intent.Level == "light" {
		return p.selectQuickDiscoveryModules(intent)
	}
	if intent.Profile == "full_scan" || intent.Level == "comprehensive" {
		return p.selectFullScanModules(intent)
	}
	return p.selectDefaultModules(intent)
}

// selectModulesForIntent filters moduleRegistry based on the scan intent.
func (p *DAGPlanner) selectModulesForIntent(intent ScanIntent) []ModuleFactory {
	// Select modules based on profile/level
	selected := p.selectModulesByProfile(intent)

	// Add parse modules
	selected = p.addParseModules(selected, p.moduleRegistry, intent)

	// Filter host discovery modules if needed
	if intent.SkipDiscovery {
		selected = p.filterHostDiscoveryModules(selected)
	}

	// Ensure reporter module exists
	return p.ensureReporter(selected, intent)
}

func (p *DAGPlanner) addParseModules(selected []ModuleFactory, all map[string]ModuleFactory, intent ScanIntent) []ModuleFactory {
	for name, factory := range all {
		meta := factory().Metadata()
		if meta.Type != ParseModuleType {
			continue
		}
		if !p.matchesTags(meta.Tags, intent.IncludeTags, intent.ExcludeTags) {
			continue
		}
		selected = append(selected, factory)
		p.logger.Debug().Str("module", name).Msg("Selected parser module")
	}
	return selected
}

func (p *DAGPlanner) ensureReporter(selected []ModuleFactory, intent ScanIntent) []ModuleFactory {
	if len(selected) == 0 {
		return selected
	}
	hasReporter := false
	for _, factory := range selected {
		if factory().Metadata().Type == ReportingModuleType {
			hasReporter = true
			break
		}
	}
	if hasReporter {
		return selected
	}
	for name, factory := range p.moduleRegistry {
		if factory().Metadata().Type != ReportingModuleType {
			continue
		}
		if !p.matchesTags(factory().Metadata().Tags, intent.IncludeTags, intent.ExcludeTags) {
			continue
		}
		selected = append(selected, factory)
		p.logger.Debug().Str("module", name).Msg("Added default reporting module")
		break
	}
	return selected
}

// matchesTags checks if a module's tags satisfy the include/exclude criteria.
func (p *DAGPlanner) matchesTags(moduleTags, includeTags, excludeTags []string) bool {
	if len(excludeTags) > 0 {
		for _, et := range excludeTags {
			if containsTag(moduleTags, et) {
				return false // Excluded by tag
			}
		}
	}
	if len(includeTags) > 0 {
		included := false
		for _, it := range includeTags {
			if containsTag(moduleTags, it) {
				included = true
				break
			}
		}
		if !included {
			return false // Does not have any of the required include tags
		}
	}
	return true
}

// configureModule creates a configuration map for a module instance based on its
// default schema and overrides from the scan intent.
func (p *DAGPlanner) configureModule(meta ModuleMetadata, intent ScanIntent) map[string]interface{} {
	cfg := make(map[string]interface{})

	// Start with module defaults from its schema
	for paramName, paramDef := range meta.ConfigSchema {
		if paramDef.Default != nil {
			cfg[paramName] = paramDef.Default
		}
	}

	// Apply intent-specific overrides (this part needs more detailed logic)
	// Example: if intent has specific port or timeout settings, apply them to relevant modules.
	if meta.Name == moduleTypeTCPPortDiscovery && intent.CustomPortConfig != "" {
		// Parse intent.CustomPortConfig and override "ports" in cfg
		// Example: cfg["ports"] = strings.Split(intent.CustomPortConfig, ",")
		// For simplicity, assume CustomPortConfig is already []string or parseable
		parsedPorts := strings.Split(intent.CustomPortConfig, ",")
		if len(parsedPorts) > 0 && (len(parsedPorts) > 1 || strings.TrimSpace(parsedPorts[0]) != "") {
			cfg["ports"] = parsedPorts
			p.logger.Debug().Str("module", meta.Name).Interface("ports", parsedPorts).Msg("Applied custom port config")
		}

	}
	if (meta.Name == moduleTypeTCPPortDiscovery || meta.Name == moduleTypeICMPPingDiscovery) && intent.CustomTimeout != "" {
		cfg["timeout"] = intent.CustomTimeout // Assuming modules can parse duration string
		p.logger.Debug().Str("module", meta.Name).Str("timeout", intent.CustomTimeout).Msg("Applied custom timeout config")
	}

	// Propagate global custom timeout to banner grabber as read/connect timeouts.
	// This ensures slow networks have enough time for banners.
	if meta.Name == "banner-grabber" && intent.CustomTimeout != "" {
		// Use full timeout for read and half for connect to stay responsive.
		cfg["read_timeout"] = intent.CustomTimeout
		// We cannot easily halve the duration string without parsing here;
		// keep connect equal to read for simplicity and reliability.
		cfg["connect_timeout"] = intent.CustomTimeout
		p.logger.Debug().Str("module", meta.Name).Str("read_timeout", intent.CustomTimeout).Str("connect_timeout", intent.CustomTimeout).Msg("Applied custom banner timeouts from intent")
	}

	// Pass global targets if module consumes "config.targets" and it's not set by a dependency yet
	// This is implicitly handled by the orchestrator when it resolves inputs from DataContext.
	// The config here is for module-specific parameters.

	return cfg
}

// generateInstanceID creates a unique instance ID for a module in the DAG.
// Appends a suffix if a module with the same base name already exists.
func (p *DAGPlanner) generateInstanceID(moduleName string, existingNodes map[string]DAGNodeConfig) string {
	baseID := strings.ReplaceAll(strings.ToLower(moduleName), "-", "_")
	id := baseID
	counter := 1
	for {
		if _, exists := existingNodes[id]; !exists {
			return id
		}
		id = fmt.Sprintf("%s_%d", baseID, counter)
		counter++
	}
}

// Helper to check if a slice contains a string.
func containsTag(tags []string, tagToFind string) bool {
	for _, t := range tags {
		if t == tagToFind {
			return true
		}
	}
	return false
}

// Helper to get a meaningful name for the DAG based on intent
func (intent ScanIntent) Profile_or_Level_or_Default() string {
	if intent.Profile != "" {
		return intent.Profile
	}
	if intent.Level != "" {
		return intent.Level
	}
	return "default_scan"
}

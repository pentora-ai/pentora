// pkg/engine/planner.go
package engine

import (
	"fmt"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

	availableDataKeys := make(map[string]string) // DataKey -> Producing InstanceID
	if len(intent.Targets) > 0 {
		availableDataKeys["config.targets"] = "initial_input" // Mark config.targets as initially available
		p.logger.Debug().Interface("initial_keys", availableDataKeys).Msg("Initial available data keys")
	}
	// Add other global/initial keys if necessary, e.g., from intent.CustomPortConfig if planner doesn't set it directly in module config
	// if intent.CustomPortConfig != "" {
	// 	availableDataKeys["config.ports"] = "initial_input"
	// }

	// Store node configs by instance ID to ensure uniqueness and for lookups
	dagNodeConfigs := make(map[string]DAGNodeConfig)
	// Track module types already added to the DAG to add each type at most once in this simple auto-plan
	moduleTypesAddedToDAG := make(map[string]bool)

	// Iteratively build the DAG layer by layer
	for { // Loop until no more modules can be added in a full pass
		addedInThisIteration := 0

		for _, modFactory := range candidateModules {
			tempMod := modFactory() // Create a temporary instance to get metadata
			meta := tempMod.Metadata()

			if moduleTypesAddedToDAG[meta.Name] { // If this module *type* has already been added
				continue
			}

			spew.Dump(meta.ID)

			//spew.Dump(meta.Consumes, availableDataKeys)

			// Check if all consumed keys for this module are currently available
			allConsumesMet := true
			if len(meta.Consumes) > 0 {
				for _, consumedContract := range meta.Consumes {
					consumedKeyString := consumedContract.Key // Use the string Key
					if _, keyIsAvailable := availableDataKeys[consumedKeyString]; !keyIsAvailable {
						allConsumesMet = false
						p.logger.Trace().Str("module", meta.Name).Str("missing_key", consumedKeyString).Msg("Dependency key not yet available for module")
						break
					}
				}
			} // Modules with no consumes (or all consumes met by initial_input) are considered for the first layer

			if allConsumesMet {
				// This module's dependencies are met, it can be added to the DAG
				instanceID := p.generateInstanceID(meta.Name, dagNodeConfigs) // Pass current DAG nodes to ensure unique ID

				nodeCfg := DAGNodeConfig{
					InstanceID: instanceID,
					ModuleType: meta.Name, // Use the registered module type name
					Config:     p.configureModule(meta, intent),
				}

				dagDef.Nodes = append(dagDef.Nodes, nodeCfg)
				dagNodeConfigs[instanceID] = dagDef.Nodes[len(dagDef.Nodes)-1] // Store pointer to the added node config
				moduleTypesAddedToDAG[meta.Name] = true                        // Mark this module TYPE as added

				p.logger.Debug().Str("module", meta.Name).Str("instance_id", instanceID).Msg("Added module to DAG")

				// Add its produced keys to availableDataKeys for subsequent modules in this or next iterations
				for _, producedContract := range meta.Produces {
					producedKey := producedContract.Key // Use the string Key
					if existingProducer, found := availableDataKeys[producedKey]; found && existingProducer != "initial_input" {
						p.logger.Warn().Str("data_key", producedKey).Str("new_producer", instanceID).Str("existing_producer", existingProducer).Msg("DataKey already produced by another module. Overwriting producer.")
					}
					availableDataKeys[producedKey] = instanceID // Mark key as available, produced by this new instance
					p.logger.Trace().Str("module_producer", meta.Name).Str("instance_id_producer", instanceID).Str("produced_key", producedKey).Msg("Marked key as available")
				}
				addedInThisIteration++
			}
		}

		if addedInThisIteration == 0 {
			// No new modules were added in this full pass over all candidates.
			// This means either all addable modules are in, or remaining ones have unmet dependencies.
			p.logger.Debug().Int("total_dag_nodes", len(dagDef.Nodes)).Msg("No more modules added in this planning iteration. Loop will terminate.")
			break
		}
		p.logger.Debug().Int("added_this_iteration", addedInThisIteration).Int("total_dag_nodes", len(dagDef.Nodes)).Msg("Completed an iteration of DAG planning.")
	} // End of main planning loop

	// After the loop, check if all selected candidate modules were actually added
	if len(moduleTypesAddedToDAG) < len(candidateModules) {
		p.logger.Warn().Msg("Not all candidate modules selected by intent could be added to the DAG. Logging unprocessed modules and their potential unmet dependencies:")
		for _, modFactory := range candidateModules {
			meta := modFactory().Metadata()
			if !moduleTypesAddedToDAG[meta.Name] {
				unmetDependencies := []string{}
				for _, consumedContract := range meta.Consumes {
					consumedKey := consumedContract.Key // Use the string Key
					if _, found := availableDataKeys[consumedKey]; !found {
						unmetDependencies = append(unmetDependencies, consumedKey)
					}
				}
				p.logger.Warn().Str("module", meta.Name).Strs("unmet_dependencies", unmetDependencies).Msg("Unprocessed candidate module")
			}
		}
		// Depending on strictness, this could be an error or just a warning.
		// If a core module for the intent couldn't be added, it might be an error.
	}

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

// selectModulesForIntent filters moduleRegistry based on the scan intent.
// This is a placeholder and needs to be implemented with more sophisticated logic.
func (p *DAGPlanner) selectModulesForIntent(intent ScanIntent) []ModuleFactory {
	var selected []ModuleFactory
	allModules := p.moduleRegistry // Assuming this holds ModuleName -> Factory

	// Basic filtering based on profile or level (example logic)
	if intent.Profile == "quick_discovery" || intent.Level == "light" {
		for name, factory := range allModules {
			meta := factory().Metadata()
			if meta.Type == DiscoveryModuleType || (containsTag(meta.Tags, "quick") && meta.Type == ScanModuleType) {
				// Further filter by IncludeTags/ExcludeTags
				if p.matchesTags(meta.Tags, intent.IncludeTags, intent.ExcludeTags) {
					selected = append(selected, factory)
					p.logger.Debug().Str("module", name).Msg("Selected module for quick_discovery/light profile")
				}
			}
		}
	} else if intent.Profile == "full_scan" || intent.Level == "comprehensive" {
		for name, factory := range allModules {
			meta := factory().Metadata()
			// Include Discovery, Scan, Parse. Conditionally Evaluation.
			if meta.Type == DiscoveryModuleType || meta.Type == ScanModuleType || meta.Type == ParseModuleType ||
				(intent.EnableVulnChecks && meta.Type == EvaluationModuleType) {
				if p.matchesTags(meta.Tags, intent.IncludeTags, intent.ExcludeTags) {
					selected = append(selected, factory)
					p.logger.Debug().Str("module", name).Msg("Selected module for full_scan/comprehensive profile")
				}
			}
		}
	} else { // Default: select all non-intrusive discovery and basic scan modules
		for name, factory := range allModules {
			meta := factory().Metadata()
			if (meta.Type == DiscoveryModuleType || meta.Type == ScanModuleType) && !containsTag(meta.Tags, "intrusive") {
				if p.matchesTags(meta.Tags, intent.IncludeTags, intent.ExcludeTags) {
					selected = append(selected, factory)
					p.logger.Debug().Str("module", name).Msg("Selected module for default profile")
				}
			}
		}
	}

	// Always try to add a reporting module if not already present in logic
	// This part can be more sophisticated
	hasReporter := false
	for _, factory := range selected {
		if factory().Metadata().Type == ReportingModuleType {
			hasReporter = true
			break
		}
	}
	if !hasReporter {
		for name, factory := range allModules { // Search all modules again for a reporter
			if factory().Metadata().Type == ReportingModuleType {
				// Add a default reporter if none selected and one exists
				if p.matchesTags(factory().Metadata().Tags, intent.IncludeTags, intent.ExcludeTags) {
					selected = append(selected, factory)
					p.logger.Debug().Str("module", name).Msg("Added default reporting module")
					break // Add one reporter for now
				}
			}
		}
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
	if meta.Name == "tcp-port-discovery" && intent.CustomPortConfig != "" {
		// Parse intent.CustomPortConfig and override "ports" in cfg
		// Example: cfg["ports"] = strings.Split(intent.CustomPortConfig, ",")
		// For simplicity, assume CustomPortConfig is already []string or parseable
		parsedPorts := strings.Split(intent.CustomPortConfig, ",")
		if len(parsedPorts) > 0 && (len(parsedPorts) > 1 || strings.TrimSpace(parsedPorts[0]) != "") {
			cfg["ports"] = parsedPorts
			p.logger.Debug().Str("module", meta.Name).Interface("ports", parsedPorts).Msg("Applied custom port config")
		}

	}
	if (meta.Name == "tcp-port-discovery" || meta.Name == "icmp-ping-discovery") && intent.CustomTimeout != "" {
		cfg["timeout"] = intent.CustomTimeout // Assuming modules can parse duration string
		p.logger.Debug().Str("module", meta.Name).Str("timeout", intent.CustomTimeout).Msg("Applied custom timeout config")
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

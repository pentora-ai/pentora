// pkg/cli/scan.go
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/pentora-ai/pentora/pkg/engine"
	"github.com/pentora-ai/pentora/pkg/modules/discovery" // For casting results
	_ "github.com/pentora-ai/pentora/pkg/modules/parse"   // Register parse modules if needed
	_ "github.com/pentora-ai/pentora/pkg/modules/scan"    // Register this module

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Flags for the scan command (remains the same)
var (
	scanTargets       []string
	scanPorts         string
	scanProfile       string
	scanLevel         string
	scanIncludeTags   []string
	scanExcludeTags   []string
	scanEnableVuln    bool
	scanOutputFormat  string
	scanCustomTimeout string
	scanConcurrency   int  // This might become a global config or part of ScanIntent for planner
	scanEnablePing    bool // This now feeds into ScanIntent
	scanPingCount     int  // This now feeds into ScanIntent
	scanAllowLoopback bool // This now feeds into ScanIntent
)

// ScanCmd defines the 'scan' command for comprehensive scanning.
var ScanCmd = &cobra.Command{
	Use:   "scan [targets...]",
	Short: "Perform a comprehensive scan on specified targets",
	Long: `Performs various scanning stages based on selected profile, level, or flags.
The command automatically plans the execution DAG using available modules.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scanTargetsFromCLI := args

		logger := log.With().Str("command", "scan").Logger()
		logger.Info().Strs("targets", scanTargetsFromCLI).Msg("Initializing scan command")

		ctxFromCmd := cmd.Context()
		appMgr, ok := ctxFromCmd.Value(engine.AppManagerKey).(*engine.AppManager)
		if !ok || appMgr == nil {
			logger.Error().Msg("AppManager not found in context.")
			fmt.Fprintln(os.Stderr, "[ERROR] Critical: AppManager not found in context.")
			return
		}
		orchestratorCtx := appMgr.Context()

		// 1. Create ScanIntent from CLI flags
		// More fields can be added to ScanIntent to guide the planner
		scanIntent := engine.ScanIntent{
			Targets:          scanTargetsFromCLI,
			Profile:          scanProfile, // e.g., "quick_discovery", "full_vuln_scan"
			Level:            scanLevel,   // e.g., "light", "comprehensive"
			IncludeTags:      scanIncludeTags,
			ExcludeTags:      scanExcludeTags,
			EnableVulnChecks: scanEnableVuln,
			CustomPortConfig: scanPorts,         // Planner will use this to configure port scanning modules
			CustomTimeout:    scanCustomTimeout, // Planner can apply this to relevant modules
			// Pass ping-related flags to the intent so planner can configure ICMP module if selected
			EnablePing:    scanEnablePing,
			PingCount:     scanPingCount,
			AllowLoopback: scanAllowLoopback,
			Concurrency:   scanConcurrency, // Global concurrency hint for planner
		}
		logger.Debug().Interface("scan_intent", scanIntent).Msg("Scan intent created")

		// 2. Initialize DAGPlanner
		// GetRegisteredModuleFactories provides access to all module metadata for the planner
		planner, err := engine.NewDAGPlanner(engine.GetRegisteredModuleFactories())
		if err != nil {
			logger.Error().Err(err).Msg("Failed to initialize DAGPlanner")
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to initialize DAGPlanner: %v\n", err)
			return
		}

		// 3. Plan the DAG based on the intent
		// The planner now decides which modules to use (icmp, tcp, etc.) and how to configure them.
		dagDefinition, err := planner.PlanDAG(scanIntent)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to plan DAG")
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to plan DAG: %v\n", err)
			return
		}
		if dagDefinition == nil || len(dagDefinition.Nodes) == 0 {
			logger.Error().Msg("DAG planning resulted in an empty DAG. No modules were selected for the scan intent.")
			fmt.Fprintln(os.Stderr, "[ERROR] DAG planning resulted in an empty DAG. Check scan parameters and available modules.")
			return
		}
		logger.Info().Str("dag_name", dagDefinition.Name).Int("node_count", len(dagDefinition.Nodes)).Msg("DAG planned successfully")
		logger.Debug().Interface("dag_definition", dagDefinition).Msg("Full automatically planned DAG Definition")

		// 4. Create Orchestrator with the automatically planned DAG
		orchestrator, err := engine.NewOrchestrator(dagDefinition) // NewOrchestrator uses the planned DAG
		if err != nil {
			logger.Error().Err(err).Msg("Failed to initialize orchestrator with planned DAG")
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to initialize orchestrator: %v\n", err)
			return
		}

		// 5. Prepare initial inputs for the Orchestrator (minimal, as planner handles most config)
		initialInputs := make(map[string]interface{})
		initialInputs["config.targets"] = scanTargetsFromCLI // The primary input
		// Other global settings that planner might not have put into individual module configs
		// but some modules might still consume globally.
		// For example, if a report module needs the original CLI target strings:
		initialInputs["config.original_cli_targets"] = scanTargetsFromCLI

		if scanOutputFormat == "text" {
			logger.Info().Msg("Starting scan execution with automatically planned DAG...")
		}

		// 6. Run the Orchestrator
		finalDataContext, dagErr := orchestrator.Run(orchestratorCtx, initialInputs)

		// 7. Process and Output Results (logic remains similar, but adapts to the planned DAG)
		outputData := processScanResults(finalDataContext, scanTargetsFromCLI, dagDefinition, dagErr, logger)

		switch strings.ToLower(scanOutputFormat) {
		case "json":
			jsonData, jsonErr := json.MarshalIndent(outputData, "", "  ")
			if jsonErr != nil {
				logger.Error().Err(jsonErr).Msg("Failed to marshal JSON output")
				fmt.Fprintf(os.Stderr, "[ERROR] Failed to marshal JSON output: %v\n", jsonErr)
				return
			}
			fmt.Println(string(jsonData))
		case "yaml":
			yamlData, yamlErr := yaml.Marshal(outputData)
			if yamlErr != nil {
				logger.Error().Err(yamlErr).Msg("Failed to marshal YAML output")
				fmt.Fprintf(os.Stderr, "[ERROR] Failed to marshal YAML output: %v\n", yamlErr)
				return
			}
			fmt.Println(string(yamlData))
		default:
			if scanOutputFormat != "" && !strings.EqualFold(scanOutputFormat, "text") {
				logger.Warn().Str("format_provided", scanOutputFormat).Msg("Unknown output format. Defaulting to text.")
			}
			printScanTextOutput(outputData, logger, scanIntent.EnablePing) // Pass ping status for context
		}
		logger.Info().Msg("Scan command finished.")
	},
}

// processScanResults extracts and structures data from the final DataContext.
// It needs the dagDefinition to know which modules ran and what their instanceIDs are.
func processScanResults(dataCtx map[string]interface{}, queriedTargets []string, dagDef *engine.DAGDefinition, dagErr error, logger zerolog.Logger) DiscoveryOutput {
	output := DiscoveryOutput{ // Using DiscoveryOutput for now, can be extended to a more generic ScanOutput
		Timestamp:      time.Now(),
		TargetsQueried: queriedTargets,
		LiveHosts:      []string{},
		OpenTCPPorts:   []discovery.TCPPortDiscoveryResult{},
		Errors:         []string{},
	}
	if dagErr != nil {
		output.Errors = append(output.Errors, fmt.Sprintf("DAG execution error: %v", dagErr))
	}

	// Iterate over the nodes that were actually planned and find their outputs
	for _, nodeCfg := range dagDef.Nodes {
		moduleType := nodeCfg.ModuleType // This is the registered name of the module

		// Temporary module instance to get metadata (Produces DataKey)
		// Ideally, DAGPlanner might also return a list of (InstanceID, ProducedDataKey) mappings
		// or the ModuleMetadata could be part of DAGNodeConfig after planning.
		// For now, we'll get metadata again, or assume standard DataKeys.

		// Example for ICMP Ping results
		if moduleType == "icmp-ping-discovery" {
			outputKey := "discovery.live_hosts"
			if rawData, found := dataCtx[outputKey]; found {
				if listData, ok := rawData.([]interface{}); ok && len(listData) > 0 {
					if result, castOk := listData[0].(discovery.ICMPPingDiscoveryResult); castOk {
						output.LiveHosts = append(output.LiveHosts, result.LiveHosts...)
					} else {
						output.Errors = append(output.Errors, fmt.Sprintf("Cast error for %s: expected %T, got %T", outputKey, discovery.ICMPPingDiscoveryResult{}, listData[0]))
					}
				} else if listData == nil && len(listData) == 0 {
					// It means key exists but with empty list value (e.g. no live hosts)
				} else if rawData != nil { // Not a list, but not nil
					output.Errors = append(output.Errors, fmt.Sprintf("Unexpected data type for %s: expected list, got %T", outputKey, rawData))
				}
			}
		}

		// Example for TCP Port Discovery results
		if moduleType == "tcp-port-discovery" {
			outputKey := "discovery.open_tcp_ports"
			if rawData, found := dataCtx[outputKey]; found {
				if listData, ok := rawData.([]interface{}); ok {
					for _, item := range listData {
						if result, castOk := item.(discovery.TCPPortDiscoveryResult); castOk {
							output.OpenTCPPorts = append(output.OpenTCPPorts, result)
						} else {
							output.Errors = append(output.Errors, fmt.Sprintf("Cast error for item in %s: expected %T, got %T", outputKey, discovery.TCPPortDiscoveryResult{}, item))
						}
					}
				} else if rawData != nil {
					output.Errors = append(output.Errors, fmt.Sprintf("Unexpected data type for %s: expected list, got %T", outputKey, rawData))
				}
			}
		}
		// Add more cases here for other module types and their expected DataKeys
		// e.g., "service-banner-scanner", "ssh-default-creds"
	}

	// Consolidate and sort
	if len(output.LiveHosts) > 0 {
		seen := make(map[string]struct{})
		uniqueLiveHosts := []string{}
		for _, h := range output.LiveHosts {
			if _, exists := seen[h]; !exists {
				seen[h] = struct{}{}
				uniqueLiveHosts = append(uniqueLiveHosts, h)
			}
		}
		sort.Strings(uniqueLiveHosts)
		output.LiveHosts = uniqueLiveHosts
	}
	if len(output.OpenTCPPorts) > 0 {
		sort.Slice(output.OpenTCPPorts, func(i, j int) bool {
			if output.OpenTCPPorts[i].Target == output.OpenTCPPorts[j].Target {
				// This case shouldn't happen if tcp-module sends one result per target.
				// If it does, sort by first port.
				if len(output.OpenTCPPorts[i].OpenPorts) > 0 && len(output.OpenTCPPorts[j].OpenPorts) > 0 {
					return output.OpenTCPPorts[i].OpenPorts[0] < output.OpenTCPPorts[j].OpenPorts[0]
				}
				return len(output.OpenTCPPorts[i].OpenPorts) < len(output.OpenTCPPorts[j].OpenPorts)
			}
			return output.OpenTCPPorts[i].Target < output.OpenTCPPorts[j].Target
		})
	}

	return output
}

// printScanTextOutput needs to be adjusted to use the `enablePing` from intent
func printScanTextOutput(data DiscoveryOutput, logger zerolog.Logger, pingEnabled bool) {
	// ... (Text output logic, using pingEnabled to determine if "No live hosts" vs "Ping disabled")
	logger.Info().Msg("Printing scan results in text format...")
	fmt.Println("\nScan Results (Text):")
	fmt.Printf("  Timestamp: %s\n", data.Timestamp.Format(time.RFC3339))
	fmt.Printf("  Targets Queried: %v\n", data.TargetsQueried)

	if len(data.LiveHosts) > 0 {
		fmt.Printf("  Live Hosts (ICMP):\n")
		for _, host := range data.LiveHosts { // Already sorted
			fmt.Printf("    - %s\n", host)
		}
	} else if pingEnabled {
		fmt.Println("  No live hosts found via ICMP ping.")
	} else {
		fmt.Println("  ICMP Ping was disabled for this scan.")
	}

	if len(data.OpenTCPPorts) > 0 {
		fmt.Printf("\n  Open TCP Ports:\n")
		for _, result := range data.OpenTCPPorts { // Already sorted by target
			fmt.Printf("    - Target %s: %v\n", result.Target, result.OpenPorts) // Ports are sorted within TCPPortDiscoveryResult
		}
	} else {
		fmt.Println("  No open TCP ports found (or no live hosts to scan ports on).")
	}

	if len(data.Errors) > 0 {
		fmt.Printf("\n  Errors/Warnings Encountered During Scan:\n")
		for _, errMsg := range data.Errors {
			fmt.Printf("    - %s\n", errMsg)
		}
	}
	fmt.Println("\nScan finished.")
}

func init() {
	// Flags for ScanCmd (ensure these are descriptive for the planner)
	ScanCmd.Flags().StringVarP(&scanPorts, "ports", "p", "", "Ports/port ranges for TCP scan (e.g., 'top-1000', '22,80,443', '1-65535')")
	ScanCmd.Flags().StringVar(&scanProfile, "profile", "", "Predefined scan profile (e.g., 'quick_discovery', 'full_vuln_scan')")
	ScanCmd.Flags().StringVar(&scanLevel, "level", "default", "Scan intensity level (e.g., 'light', 'default', 'comprehensive', 'intrusive')")
	ScanCmd.Flags().StringSliceVar(&scanIncludeTags, "tags", []string{}, "Only include modules with these tags (comma-separated)")
	ScanCmd.Flags().StringSliceVar(&scanExcludeTags, "exclude-tags", []string{}, "Exclude modules with these tags (comma-separated)")
	ScanCmd.Flags().BoolVar(&scanEnableVuln, "vuln", false, "Enable vulnerability assessment modules (shortcut for a common intent)")
	ScanCmd.Flags().StringVarP(&scanOutputFormat, "output", "o", "text", "Output format: text, json, yaml")
	ScanCmd.Flags().StringVar(&scanCustomTimeout, "timeout", "1s", "Default timeout for network operations like ping/port connect")
	ScanCmd.Flags().IntVar(&scanConcurrency, "concurrency", 50, "Default concurrency for parallel operations")

	// Ping specific flags - planner can use these if ICMP module is selected
	ScanCmd.Flags().BoolVar(&scanEnablePing, "ping", true, "Enable ICMP host discovery (default: true)")
	ScanCmd.Flags().IntVar(&scanPingCount, "ping-count", 1, "Number of ICMP pings per host")
	ScanCmd.Flags().BoolVar(&scanAllowLoopback, "allow-loopback", false, "Allow scanning loopback addresses")
}

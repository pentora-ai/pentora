// pkg/cli/discover.go
package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	// For ParsePortString and ParseAndExpandTargets
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/pentora-ai/pentora/pkg/engine"            // Engine interfaces
	"github.com/pentora-ai/pentora/pkg/modules/discovery" // To access discovery module results if needed directly
)

var (
	// Flag variables for the discover command
	discoverPortsFlag       string // Port list or range for TCP port discovery
	discoverTimeoutFlag     string // Timeout for discovery operations
	discoverPingFlag        bool   // Flag to enable/disable ICMP ping
	discoverPingCountFlag   int    // ICMP ping count
	discoverConcurrencyFlag int    // Concurrency for discovery operations
	discoverAllowLoopback   bool   // Allow discovery on loopback addresses
	outputFormatFlag        string // For selecting output format (text, json, yaml)

)

// DiscoveryOutput is a structured type for JSON/YAML output.
type DiscoveryOutput struct {
	Timestamp      time.Time                          `json:"timestamp" yaml:"timestamp"`
	TargetsQueried []string                           `json:"targets_queried" yaml:"targets_queried"`
	LiveHosts      []string                           `json:"live_hosts,omitempty" yaml:"live_hosts,omitempty"`
	OpenTCPPorts   []discovery.TCPPortDiscoveryResult `json:"open_tcp_ports,omitempty" yaml:"open_tcp_ports,omitempty"`
	Errors         []string                           `json:"errors,omitempty" yaml:"errors,omitempty"`
}

// DiscoverCmd defines the 'discover' command for host and port discovery.
var DiscoverCmd = &cobra.Command{
	Use:   "discover [targets...]",
	Short: "Discover live hosts and open ports on specified targets",
	Long: `Performs host discovery using ICMP ping (optional) and TCP port discovery on the specified targets.
Targets can be IPs, hostnames, CIDR notations, or IP ranges.
Ports can be specified as a comma-separated list or ranges (e.g., "80,443,1000-1024").`,
	Args: cobra.MinimumNArgs(1), // Require at least one target
	Run: func(cmd *cobra.Command, args []string) {
		// Create a root context for the orchestrator run
		// A real AppManager would provide this.
		ctx := cmd.Context()
		AppManager, ok := ctx.Value(engine.AppManagerKey).(*engine.AppManager)
		if !ok || AppManager == nil {
			fmt.Fprintln(os.Stderr, "[ERROR] AppManager not found in context. Initialization failed.")
			return
		}

		targets := args // Targets are taken directly from command arguments

		// Initialize a basic AppManager and Orchestrator for discovery
		// In a real application, this might be retrieved from a global instance
		// or created via a factory that sets up logging, config, etc.
		// For simplicity, we'll mock or create a minimal one here.

		// Create a minimal DAG definition for discovery
		// This DAG will run ICMP ping first (if enabled), then TCP port discovery
		dagNodes := []engine.DAGNodeConfig{}
		moduleInputs := make(map[string]interface{}) // Initial inputs for the DAG

		// Prepare global/initial config for the DAG context
		// These can be overridden by module-specific configs in DAGNodeConfig
		moduleInputs["config.targets"] = targets // Pass targets from CLI

		// 1. ICMP Ping Discovery Node (Optional)
		if discoverPingFlag {
			icmpConfig := map[string]interface{}{
				// "targets" will be consumed from inputs["config.targets"] by the module if not specified here
				"count":          discoverPingCountFlag,
				"timeout":        discoverTimeoutFlag, // Ping module uses this for its overall operation
				"packet_timeout": discoverTimeoutFlag, // And this for individual pinger.Timeout
				"concurrency":    discoverConcurrencyFlag,
				"allow_loopback": discoverAllowLoopback,
				// Privileged mode could be another flag or auto-detected
			}
			dagNodes = append(dagNodes, engine.DAGNodeConfig{
				InstanceID: "icmp_discovery_instance",
				ModuleType: "icmp-ping-discovery", // Must match registered factory name
				Config:     icmpConfig,
			})
		}

		// 2. TCP Port Discovery Node
		tcpPortsToScan := strings.Split(discoverPortsFlag, ",") // Allows multiple port strings like "80,443", "1000-1024"
		if discoverPortsFlag == "" {                            // If no ports specified, use a default common set
			tcpPortsToScan = []string{"1-1024"} // Default to scan first 1024 ports if --ports is empty
		}

		tcpConfig := map[string]interface{}{
			// "targets" can be consumed from "discovery.live_hosts" if ICMP ran, or from "config.targets"
			"ports":       tcpPortsToScan,
			"timeout":     discoverTimeoutFlag,
			"concurrency": discoverConcurrencyFlag,
			// "allow_loopback" could also be a config here if needed independently of ICMP
		}
		// If ICMP ping is enabled, TCP discovery should consume its output
		// The orchestrator and module's `Consumes` field will handle this data flow.
		// Ensure 'tcp-port-discovery' module is designed to look for 'discovery.live_hosts' in its inputs.
		dagNodes = append(dagNodes, engine.DAGNodeConfig{
			InstanceID: "tcp_discovery_instance",
			ModuleType: "tcp-port-discovery", // Must match registered factory name
			Config:     tcpConfig,
		})

		discoveryDAG := &engine.DAGDefinition{
			Name:        "CLI Discovery DAG",
			Description: "Performs host and port discovery based on CLI inputs.",
			Nodes:       dagNodes,
		}

		// --- Orchestrator Setup and Execution ---
		// This assumes module factories for "icmp-ping-discovery" and "tcp-port-discovery" are registered
		orchestrator, err := engine.NewOrchestrator(discoveryDAG)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] Failed to initialize orchestrator: %v\n", err)
			return
		}

		ctx = AppManager.Context()

		log.Info().Msg("Starting discovery...")
		finalDataContext, dagErr := orchestrator.Run(ctx, moduleInputs) // Pass the initial inputs
		// dagErr will be non-nil if a module returned an error or context was canceled.

		// Prepare structured output
		outputData := DiscoveryOutput{
			Timestamp:      time.Now(),
			TargetsQueried: targets, // Store the initial targets queried by the user
			LiveHosts:      []string{},
			OpenTCPPorts:   []discovery.TCPPortDiscoveryResult{},
			Errors:         []string{},
		}

		if dagErr != nil {
			errMsg := fmt.Sprintf("DAG execution encountered an error: %v", dagErr)
			outputData.Errors = append(outputData.Errors, errMsg)
			log.Error().Err(dagErr).Msg("Discovery DAG execution failed")
		}

		// Process ICMP Ping Results if the module was part of the DAG
		if discoverPingFlag {
			icmpOutputKey := "icmp_discovery_instance.discovery.live_hosts"
			log.Debug().Str("key", icmpOutputKey).Msg("Processing ICMP results")
			if rawOutput, found := finalDataContext[icmpOutputKey]; found {
				// DataContext.AddOrAppendToList stores outputs as []interface{}
				if outputList, listOk := rawOutput.([]interface{}); listOk {
					if len(outputList) > 0 {
						// ICMP module produces one ICMPPingDiscoveryResult
						if pingResult, castOk := outputList[0].(discovery.ICMPPingDiscoveryResult); castOk {
							outputData.LiveHosts = pingResult.LiveHosts
							sort.Strings(outputData.LiveHosts) // Sort for consistent output
							log.Debug().Strs("live_hosts", outputData.LiveHosts).Msg("Live hosts from ICMP")
						} else {
							errMsg := fmt.Sprintf("Could not cast ICMP result item: expected discovery.ICMPPingDiscoveryResult, got %T", outputList[0])
							outputData.Errors = append(outputData.Errors, errMsg)
							log.Warn().Str("key", icmpOutputKey).Type("actual_type", outputList[0]).Msg(errMsg)
						}
					} else {
						log.Info().Str("key", icmpOutputKey).Msg("ICMP discovery result list is empty")
					}
				} else if rawOutput != nil { // Should be a list if AddOrAppendToList was used
					errMsg := fmt.Sprintf("ICMP result for key '%s' is not a list as expected. Type: %T", icmpOutputKey, rawOutput)
					outputData.Errors = append(outputData.Errors, errMsg)
					log.Warn().Str("key", icmpOutputKey).Type("actual_type", rawOutput).Msg(errMsg)
				}
			} else {
				log.Info().Str("key", icmpOutputKey).Msg("ICMP discovery output key not found in results. Module might not have run or produced output.")
			}
		}

		// Process TCP Port Discovery Results
		tcpOutputKey := "tcp_discovery_instance.discovery.open_tcp_ports"
		log.Debug().Str("key", tcpOutputKey).Msg("Processing TCP port results")
		if rawOutput, found := finalDataContext[tcpOutputKey]; found {
			// TCP module sends one TCPPortDiscoveryResult per target, aggregated into a list by DataContext
			if outputList, listOk := rawOutput.([]interface{}); listOk {
				var tcpResults []discovery.TCPPortDiscoveryResult
				for i, item := range outputList {
					if portResult, castOk := item.(discovery.TCPPortDiscoveryResult); castOk {
						sort.Ints(portResult.OpenPorts) // Sort ports for each target
						tcpResults = append(tcpResults, portResult)
					} else {
						errMsg := fmt.Sprintf("Could not cast TCP port result item #%d: expected discovery.TCPPortDiscoveryResult, got %T", i, item)
						outputData.Errors = append(outputData.Errors, errMsg)
						log.Warn().Str("key", tcpOutputKey).Int("item_index", i).Type("actual_type", item).Msg(errMsg)
					}
				}
				// Sort results by target IP for consistent output
				sort.Slice(tcpResults, func(i, j int) bool {
					return tcpResults[i].Target < tcpResults[j].Target
				})
				outputData.OpenTCPPorts = tcpResults
				log.Debug().Int("num_targets_with_open_ports", len(outputData.OpenTCPPorts)).Msg("Open TCP ports processed")

			} else if rawOutput != nil {
				errMsg := fmt.Sprintf("TCP port result for key '%s' is not a list as expected. Type: %T", tcpOutputKey, rawOutput)
				outputData.Errors = append(outputData.Errors, errMsg)
				log.Warn().Str("key", tcpOutputKey).Type("actual_type", rawOutput).Msg(errMsg)
			}
		} else {
			log.Info().Str("key", tcpOutputKey).Msg("TCP port discovery output key not found in results. Module might not have run or no open ports found.")
		}

		// Collect explicit error outputs from modules (if any module uses DataKey like "error.*")
		for key, data := range finalDataContext {
			if strings.Contains(key, ".error.") { // e.g., "instance_id.error.input"
				if errorList, listOk := data.([]interface{}); listOk {
					for _, item := range errorList {
						if errItem, castOk := item.(error); castOk {
							outputData.Errors = append(outputData.Errors, fmt.Sprintf("%s: %s", key, errItem.Error()))
						} else if strItem, castOk := item.(string); castOk {
							outputData.Errors = append(outputData.Errors, fmt.Sprintf("%s: %s", key, strItem))
						}
					}
				} else if errItem, castOk := data.(error); castOk { // If not stored as list by DataContext for some reason
					outputData.Errors = append(outputData.Errors, fmt.Sprintf("%s: %s", key, errItem.Error()))
				} else if strItem, castOk := data.(string); castOk {
					outputData.Errors = append(outputData.Errors, fmt.Sprintf("%s: %s", key, strItem))
				}
			}
		}

		// Output results based on the selected format
		switch strings.ToLower(outputFormatFlag) {
		case "json":
			jsonData, jsonErr := json.MarshalIndent(outputData, "", "  ")
			if jsonErr != nil {
				log.Error().Err(jsonErr).Msg("Failed to marshal JSON output")
				fmt.Fprintf(os.Stderr, "[ERROR] Failed to marshal JSON output: %v\n", jsonErr)
				return
			}
			fmt.Println(string(jsonData))
		case "yaml":
			yamlData, yamlErr := yaml.Marshal(outputData)
			if yamlErr != nil {
				log.Error().Err(yamlErr).Msg("Failed to marshal YAML output")
				fmt.Fprintf(os.Stderr, "[ERROR] Failed to marshal YAML output: %v\n", yamlErr)
				return
			}
			fmt.Println(string(yamlData))
		default: // "text" or undefined
			if outputFormatFlag != "" && !strings.EqualFold(outputFormatFlag, "text") {
				log.Warn().Str("format_provided", outputFormatFlag).Msg("Unknown output format. Defaulting to text.")
				fmt.Fprintf(os.Stderr, "[WARN] Unknown output format '%s'. Defaulting to text.\n", outputFormatFlag)
			}
			printTextOutput(outputData, log.Logger) // Pass logger to text output function
		}
	},
}

// printTextOutput formats and prints the discovery results in a human-readable text format.
func printTextOutput(data DiscoveryOutput, logger zerolog.Logger) {
	logger.Info().Msg("Printing discovery results in text format...")

	fmt.Println("\nDiscovery Results (Text):")
	fmt.Printf("  Timestamp: %s\n", data.Timestamp.Format(time.RFC3339))
	fmt.Printf("  Targets Queried: %v\n", data.TargetsQueried)

	if len(data.LiveHosts) > 0 {
		fmt.Printf("  Live Hosts (ICMP):\n")
		for _, host := range data.LiveHosts { // Already sorted
			fmt.Printf("    - %s\n", host)
		}
	} else if discoverPingFlag { // Only print "no live hosts" if ping was attempted
		fmt.Println("  No live hosts found via ICMP ping.")
	} else {
		fmt.Println("  ICMP Ping was disabled.")
	}

	if len(data.OpenTCPPorts) > 0 {
		fmt.Printf("\n  Open TCP Ports:\n")
		for _, result := range data.OpenTCPPorts { // Already sorted by target, ports sorted within
			fmt.Printf("    - Target %s: %v\n", result.Target, result.OpenPorts)
		}
	} else {
		fmt.Println("  No open TCP ports found for the specified port list.")
	}

	if len(data.Errors) > 0 {
		fmt.Printf("\n  Errors Encountered During Discovery:\n")
		for _, errMsg := range data.Errors {
			fmt.Printf("    - %s\n", errMsg)
		}
	}
	fmt.Println("\nDiscovery finished.")
	fmt.Println()
}

// Function to register CLI flags for the discover command.
// This should be called from a central CLI setup function (e.g., in pkg/cli/root.go or cmd/pentora/pentora.go).
func init() {
	DiscoverCmd.Flags().StringVarP(&discoverPortsFlag, "ports", "p", "", "Comma-separated list or ranges of ports for TCP discovery (e.g., '22,80,443', '1-1024')")
	DiscoverCmd.Flags().StringVar(&discoverTimeoutFlag, "timeout", "1s", "Timeout for discovery operations (e.g., '500ms', '1s', '2s')")
	DiscoverCmd.Flags().BoolVar(&discoverPingFlag, "ping", true, "Enable/disable ICMP host discovery (default: true)")
	DiscoverCmd.Flags().IntVarP(&discoverPingCountFlag, "ping-count", "", 1, "Number of ICMP echo requests to send to each host")
	DiscoverCmd.Flags().IntVar(&discoverConcurrencyFlag, "concurrency", 50, "Number of concurrent discovery operations")
	DiscoverCmd.Flags().BoolVar(&discoverAllowLoopback, "allow-loopback", false, "Allow discovery on loopback addresses (e.g., 127.0.0.1)")

	// Output format flag
	DiscoverCmd.Flags().StringVarP(&outputFormatFlag, "output", "o", "text", "Output format: text, json, yaml")
}

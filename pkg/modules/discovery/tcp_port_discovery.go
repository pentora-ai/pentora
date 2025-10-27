// pkg/modules/discovery/tcp_port_discovery.go
package discovery

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	// Utilities like target and port parsing
	"github.com/rs/zerolog/log"
	"github.com/spf13/cast"

	"github.com/pentora-ai/pentora/pkg/engine" // Engine interfaces
	"github.com/pentora-ai/pentora/pkg/netutil"
)

// TCPPortDiscoveryResult stores the outcome of the TCP port discovery for a single target.
type TCPPortDiscoveryResult struct {
	Target    string `json:"target"`
	OpenPorts []int  `json:"open_ports"`
}

// TCPPortDiscoveryConfig holds configuration for the TCP port discovery module.
type TCPPortDiscoveryConfig struct {
	Targets     []string      `json:"targets"`
	Ports       []string      `json:"ports"`   // Port ranges and lists (e.g., "1-1024", "80,443,8080")
	Timeout     time.Duration `json:"timeout"` // Connection timeout for each port
	Concurrency int           `json:"concurrency"`
}

// TCPPortDiscoveryModule implements the engine.Module interface for TCP port discovery.
type TCPPortDiscoveryModule struct {
	meta   engine.ModuleMetadata
	config TCPPortDiscoveryConfig
}

const (
	tcpPortDiscoveryModuleTypeName = "tcp-port-discovery"
	defaultTCPPortDiscoveryTimeout = 1 * time.Second
	defaultTCPConcurrency          = 100
	defaultTCPPorts                = "1-1024" // Default common ports or a well-known range
)

// newTCPPortDiscoveryModule is the internal constructor for the module.
// It sets up metadata and initializes the config with default values.
func newTCPPortDiscoveryModule() *TCPPortDiscoveryModule {
	defaultConfig := TCPPortDiscoveryConfig{
		Ports:       []string{defaultTCPPorts},
		Timeout:     defaultTCPPortDiscoveryTimeout,
		Concurrency: defaultTCPConcurrency,
	}
	return &TCPPortDiscoveryModule{
		meta: engine.ModuleMetadata{
			ID:          "tcp-port-discovery-instance",  // Unique ID for this module instance, can be generated dynamically
			Name:        tcpPortDiscoveryModuleTypeName, // Type name for factory registration
			Version:     "0.1.0",
			Description: "Discovers open TCP ports on target hosts based on a list or range.",
			Type:        engine.DiscoveryModuleType,
			Author:      "Pentora Team",
			Tags:        []string{"discovery", "port", "tcp"},
			Consumes: []engine.DataContractEntry{
				{
					Key: "config.targets",
					// DataTypeName: "[]string", // This is an initial input, stored directly
					// Cardinality: engine.CardinalitySingle, // Expects a single []string list
					DataTypeName: "[]string",               // The type of the data itself
					Cardinality:  engine.CardinalitySingle, // "config.targets" itself is a single list of strings
					IsOptional:   true,                     // Can also get targets from discovery.live_hosts
					Description:  "List of initial target strings (IPs, CIDRs, hostnames) to scan.",
				},
				{
					Key: "discovery.live_hosts",
					// DataTypeName: "discovery.ICMPPingDiscoveryResult", // This is what's inside the []interface{} list
					// Cardinality: engine.CardinalityList, // Expects a list of ICMPPingDiscoveryResult from DataContext
					DataTypeName: "discovery.ICMPPingDiscoveryResult", // The type of each item in the list
					Cardinality:  engine.CardinalityList,              // Expects a list of these items
					IsOptional:   false,
					Description:  "List of live hosts (as ICMPPingDiscoveryResult) from ICMP ping module.",
				},
				{
					Key:          "config.ports", // Optional: specific ports can also be an input
					DataTypeName: "string",       // e.g., "80,443,1000-1024"
					Cardinality:  engine.CardinalitySingle,
					IsOptional:   true,
					Description:  "Port string to scan, can override module's static config.",
				},
			},
			Produces: []engine.DataContractEntry{
				{
					Key:          "discovery.open_tcp_ports",
					DataTypeName: "discovery.TCPPortDiscoveryResult",
					Cardinality:  engine.CardinalityList, // Indicates this DataKey will hold a list of results
					Description:  "List of results, each detailing open TCP ports for a specific target.",
				},
			},
			ConfigSchema: map[string]engine.ParameterDefinition{
				"targets": {
					Description: "List of IPs, CIDRs, or hostnames to scan. Can be inherited from global config or previous modules.",
					Type:        "[]string",
					Required:    false, // Can be provided by 'discovery.live_hosts' input
				},
				"ports": {
					Description: "Comma-separated list or ranges of ports (e.g., '22,80,443', '1-1024').",
					Type:        "[]string", // Array of strings, each can be a port, a list, or a range
					Required:    false,
					Default:     []string{defaultTCPPorts},
				},
				"timeout": {
					Description: "Timeout for each port connection attempt (e.g., '1s', '500ms').",
					Type:        "duration",
					Required:    false,
					Default:     defaultTCPPortDiscoveryTimeout.String(),
				},
				"concurrency": {
					Description: "Number of concurrent port scanning goroutines.",
					Type:        "int",
					Required:    false,
					Default:     defaultTCPConcurrency,
				},
			},
			// ActivationTriggers: Usually none for a primary discovery module, unless it depends on a very specific prior state.
			// IsDynamic: false,
			EstimatedCost: 2, // 1-5 scale, TCP port scan is generally a bit more involved than ICMP.
		},
		config: defaultConfig,
	}
}

// Metadata returns the module's metadata.
func (m *TCPPortDiscoveryModule) Metadata() engine.ModuleMetadata {
	return m.meta
}

// Init initializes the module with the given configuration map.
// It parses the map and populates the module's config struct, overriding defaults.
func (m *TCPPortDiscoveryModule) Init(instanceID string, moduleConfig map[string]interface{}) error {
	cfg := m.config // Start with default config values

	m.meta.ID = instanceID // Set the unique ID for this module instance

	if targetsVal, ok := moduleConfig["targets"]; ok {
		cfg.Targets = cast.ToStringSlice(targetsVal)
	}
	if portsVal, ok := moduleConfig["ports"]; ok {
		cfg.Ports = cast.ToStringSlice(portsVal)
	}
	if timeoutStr, ok := moduleConfig["timeout"].(string); ok {
		if dur, err := time.ParseDuration(timeoutStr); err == nil {
			cfg.Timeout = dur
		} else {
			// Use fmt.Fprintf(os.Stderr, ...) for warnings/errors in production code for better logging control
			fmt.Printf("[WARN] Module '%s': Invalid 'timeout' format in config: '%s'. Using default: %s\n", m.meta.Name, timeoutStr, cfg.Timeout)
		}
	}
	if concurrencyVal, ok := moduleConfig["concurrency"]; ok {
		cfg.Concurrency = cast.ToInt(concurrencyVal)
		if cfg.Concurrency < 1 {
			fmt.Printf("[WARN] Module '%s': Concurrency in config is < 1 (%d). Setting to default: %d.\n", m.meta.Name, cfg.Concurrency, defaultTCPConcurrency)
			cfg.Concurrency = defaultTCPConcurrency
		}
	}

	// Sanitize final values
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultTCPPortDiscoveryTimeout
		fmt.Printf("[WARN] Module '%s': Invalid 'timeout' value. Setting to default: %s\n", m.meta.Name, cfg.Timeout)
	}
	if len(cfg.Ports) == 0 || (len(cfg.Ports) == 1 && strings.TrimSpace(cfg.Ports[0]) == "") {
		cfg.Ports = []string{defaultTCPPorts}
		fmt.Printf("[WARN] Module '%s': No ports specified. Using default: %s\n", m.meta.Name, defaultTCPPorts)
	}

	m.config = cfg
	// For debugging during development; consider a proper logging framework for production.
	log.Debug().
		Str("module", m.meta.Name).
		Str("instance_id", m.meta.ID).Interface("config", m.config).Msg("Module configuration initialized with config.")
	return nil
}

// Execute performs the TCP port discovery.
func (m *TCPPortDiscoveryModule) Execute(ctx context.Context, inputs map[string]interface{}, outputChan chan<- engine.ModuleOutput) error {
	var targetsToScan []string

	logger := log.With().Str("module", m.meta.Name).Str("instance_id", m.meta.ID).Logger()

	// Determine targets: prefer 'discovery.live_hosts' from input, then 'config.targets' from input, then module's own config.
	if liveHosts, ok := inputs["discovery.live_hosts"].(ICMPPingDiscoveryResult); ok && len(liveHosts.LiveHosts) > 0 {
		targetsToScan = append(targetsToScan, liveHosts.LiveHosts...) // Assuming LiveHosts contains IP addresses
		logger.Debug().Msgf("Using %d live hosts from input 'discovery.live_hosts'.", len(targetsToScan))
	} else if configTargets, ok := inputs["config.targets"].([]string); ok && len(configTargets) > 0 {
		targetsToScan = netutil.ParseAndExpandTargets(configTargets)
		logger.Debug().Msgf("Using %d targets from input 'config.targets', expanded to %d IPs.", len(configTargets), len(targetsToScan))
	} else if len(m.config.Targets) > 0 {
		targetsToScan = netutil.ParseAndExpandTargets(m.config.Targets)
		fmt.Printf("[DEBUG] Module '%s': Using %d targets from module config, expanded to %d IPs.\n", m.meta.Name, len(m.config.Targets), len(targetsToScan))
	} else {
		err := fmt.Errorf("module '%s': no targets specified through inputs or module configuration", m.meta.Name)
		outputChan <- engine.ModuleOutput{FromModuleName: m.meta.ID, Error: err, Timestamp: time.Now()}
		return err
	}

	portsToScanStr := strings.Join(m.config.Ports, ",")
	parsedPorts, err := netutil.ParsePortString(portsToScanStr)
	if err != nil {
		err = fmt.Errorf("module '%s': invalid port configuration '%s': %w", m.meta.Name, portsToScanStr, err)
		outputChan <- engine.ModuleOutput{FromModuleName: m.meta.ID, Error: err, Timestamp: time.Now()}
		return err
	}

	if len(targetsToScan) == 0 {
		fmt.Printf("[INFO] Module '%s': Effective target list is empty. Nothing to scan.\n", m.meta.Name)
		// Send an empty result to indicate completion without error but no data
		outputChan <- engine.ModuleOutput{
			FromModuleName: m.meta.ID,
			DataKey:        m.meta.Produces[0].Key, // "discovery.open_tcp_ports"
			Data:           []TCPPortDiscoveryResult{},
			Timestamp:      time.Now(),
		}
		return nil
	}
	if len(parsedPorts) == 0 {
		fmt.Printf("[INFO] Module '%s': Effective port list is empty. Nothing to scan.\n", m.meta.Name)
		outputChan <- engine.ModuleOutput{
			FromModuleName: m.meta.ID,
			DataKey:        m.meta.Produces[0].Key, // "discovery.open_tcp_ports"
			Data:           []TCPPortDiscoveryResult{},
			Timestamp:      time.Now(),
		}
		return nil
	}

	logger.Info().Msgf("Starting TCP Port Discovery for %d targets on %d unique ports. Concurrency: %d, Timeout per port: %s",
		len(targetsToScan), len(parsedPorts), m.config.Concurrency, m.config.Timeout)

	var wg sync.WaitGroup
	sem := make(chan struct{}, m.config.Concurrency) // Semaphore to limit concurrency

	// Group results by target
	openPortsByTarget := make(map[string][]int)
	var mapMutex sync.Mutex // To protect openPortsByTarget map

	batchSize := 10 // Gruplama büyüklüğü
	for i := 0; i < len(targetsToScan); i += batchSize {
		end := i + batchSize
		if end > len(targetsToScan) {
			end = len(targetsToScan)
		}
		ipBatch := targetsToScan[i:end]

		logger.Debug().Msgf("Scanning IP batch: %v", ipBatch)

		for _, targetIP := range ipBatch {
			logger.Debug().Msgf("Scanning target: %s", targetIP)
			for _, port := range parsedPorts {
				// Check for context cancellation before starting new goroutines
				select {
				case <-ctx.Done():
					fmt.Printf("[INFO] Module '%s' (instance: %s): Context canceled. Aborting further port scans.\n", m.meta.Name, m.meta.ID)
					goto endLoops // Break out of both loops
				default:
				}

				wg.Add(1)
				go func(ip string, p int) {
					defer wg.Done()
					sem <- struct{}{}        // Acquire semaphore
					defer func() { <-sem }() // Release semaphore

					// Check context again inside the goroutine
					select {
					case <-ctx.Done():
						return
					default:
					}

					address := net.JoinHostPort(ip, strconv.Itoa(p))
					conn, err := net.DialTimeout("tcp", address, m.config.Timeout)
					if err == nil {
						_ = conn.Close()
						mapMutex.Lock()
						openPortsByTarget[ip] = append(openPortsByTarget[ip], p)
						mapMutex.Unlock()
						// Optionally, send individual open port findings immediately if needed by other real-time modules.
						// For aggregated results, wait until all scans for a target (or all targets) are done.
					}
				}(targetIP, port)
			}
		}

		wg.Wait()

		logger.Debug().Msgf("Completed batch: %v", ipBatch)

	}

endLoops:
	wg.Wait() // Wait for all goroutines to complete or be canceled
	// Send aggregated results per target
	for target, openPorts := range openPortsByTarget {
		if len(openPorts) > 0 {
			// Sort openPorts for consistent output if necessary
			// sort.Ints(openPorts)
			result := TCPPortDiscoveryResult{Target: target, OpenPorts: openPorts}
			outputChan <- engine.ModuleOutput{
				FromModuleName: m.meta.ID,
				DataKey:        m.meta.Produces[0].Key, // "discovery.open_tcp_ports"
				Data:           result,
				Timestamp:      time.Now(),
				Target:         target,
			}
			logger.Info().
				Str("target", target).
				Ints("open_ports", openPorts).Msgf("Target %s - Open TCP Ports: %v", target, openPorts)
		}
	}
	// If no open ports were found for any target, we might still want to send an empty aggregate or signal completion.
	// The current logic sends per-target results, so if all targets have no open ports, nothing is sent from this loop.
	// Consider if an explicit "no open ports found for any target" message is needed.
	log.Info().Msg("TCP Port Discovery completed.")
	return nil // Indicate successful completion of the module's execution logic
}

// TCPPortDiscoveryModuleFactory creates a new TCPPortDiscoveryModule instance.
// This factory function is what's registered with the core engine.
func TCPPortDiscoveryModuleFactory() engine.Module {
	return newTCPPortDiscoveryModule()
}

func init() {
	// Register the module factory with Pentora's core module registry.
	// The name "tcp-port-discovery" will be used in DAG definitions to instantiate this module.
	engine.RegisterModuleFactory(tcpPortDiscoveryModuleTypeName, TCPPortDiscoveryModuleFactory)
}

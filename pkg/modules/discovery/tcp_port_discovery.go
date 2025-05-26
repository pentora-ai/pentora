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

	"github.com/pentora-ai/pentora/pkg/engine" // Engine interfaces
	"github.com/pentora-ai/pentora/pkg/utils"  // Utilities like target and port parsing
	"github.com/spf13/cast"
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
			// ID is set by the orchestrator for each instance in a DAG.
			Name:        tcpPortDiscoveryModuleTypeName, // Type name for factory registration
			Version:     "0.1.0",
			Description: "Discovers open TCP ports on target hosts based on a list or range.",
			Type:        engine.DiscoveryModuleType,
			Author:      "Pentora Team",
			Tags:        []string{"discovery", "port", "tcp"},
			Produces:    []string{"discovery.open_tcp_ports"},               // Output data key
			Consumes:    []string{"config.targets", "discovery.live_hosts"}, // Possible inputs
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
func (m *TCPPortDiscoveryModule) Init(moduleConfig map[string]interface{}) error {
	cfg := m.config // Start with default config values

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
	fmt.Printf("[DEBUG] Module '%s' (instance: %s) initialized. Final Config: %+v\n", m.meta.Name, m.meta.ID, m.config)
	return nil
}

// Execute performs the TCP port discovery.
func (m *TCPPortDiscoveryModule) Execute(ctx context.Context, inputs map[string]interface{}, outputChan chan<- engine.ModuleOutput) error {
	var targetsToScan []string

	// Determine targets: prefer 'discovery.live_hosts' from input, then 'config.targets' from input, then module's own config.
	if liveHosts, ok := inputs["discovery.live_hosts"].([]string); ok && len(liveHosts) > 0 {
		targetsToScan = liveHosts // Assumes these are already expanded IPs
		fmt.Printf("[DEBUG] Module '%s': Using %d live hosts from input 'discovery.live_hosts'.\n", m.meta.Name, len(targetsToScan))
	} else if configTargets, ok := inputs["config.targets"].([]string); ok && len(configTargets) > 0 {
		targetsToScan = utils.ParseAndExpandTargets(configTargets)
		fmt.Printf("[DEBUG] Module '%s': Using %d targets from input 'config.targets', expanded to %d IPs.\n", m.meta.Name, len(configTargets), len(targetsToScan))
	} else if len(m.config.Targets) > 0 {
		targetsToScan = utils.ParseAndExpandTargets(m.config.Targets)
		fmt.Printf("[DEBUG] Module '%s': Using %d targets from module config, expanded to %d IPs.\n", m.meta.Name, len(m.config.Targets), len(targetsToScan))
	} else {
		err := fmt.Errorf("module '%s': no targets specified through inputs or module configuration", m.meta.Name)
		outputChan <- engine.ModuleOutput{FromModuleName: m.meta.ID, Error: err, Timestamp: time.Now()}
		return err
	}

	portsToScanStr := strings.Join(m.config.Ports, ",")
	parsedPorts, err := utils.ParsePortString(portsToScanStr)
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
			DataKey:        m.meta.Produces[0], // "discovery.open_tcp_ports"
			Data:           []TCPPortDiscoveryResult{},
			Timestamp:      time.Now(),
		}
		return nil
	}
	if len(parsedPorts) == 0 {
		fmt.Printf("[INFO] Module '%s': Effective port list is empty. Nothing to scan.\n", m.meta.Name)
		outputChan <- engine.ModuleOutput{
			FromModuleName: m.meta.ID,
			DataKey:        m.meta.Produces[0],
			Data:           []TCPPortDiscoveryResult{},
			Timestamp:      time.Now(),
		}
		return nil
	}

	fmt.Printf("[INFO] Module '%s' (instance: %s): Starting TCP Port Discovery for %d targets on %d unique ports. Concurrency: %d, Timeout per port: %s\n",
		m.meta.Name, m.meta.ID, len(targetsToScan), len(parsedPorts), m.config.Concurrency, m.config.Timeout)

	var wg sync.WaitGroup
	sem := make(chan struct{}, m.config.Concurrency) // Semaphore to limit concurrency

	// Group results by target
	openPortsByTarget := make(map[string][]int)
	var mapMutex sync.Mutex // To protect openPortsByTarget map

	for _, targetIP := range targetsToScan {
		for _, port := range parsedPorts {
			// Check for context cancellation before starting new goroutines
			select {
			case <-ctx.Done():
				fmt.Printf("[INFO] Module '%s' (instance: %s): Context cancelled. Aborting further port scans.\n", m.meta.Name, m.meta.ID)
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
					conn.Close()
					mapMutex.Lock()
					openPortsByTarget[ip] = append(openPortsByTarget[ip], p)
					mapMutex.Unlock()
					// Optionally, send individual open port findings immediately if needed by other real-time modules.
					// For aggregated results, wait until all scans for a target (or all targets) are done.
				}
			}(targetIP, port)
		}
	}

endLoops:
	wg.Wait() // Wait for all goroutines to complete or be cancelled

	// Send aggregated results per target
	for target, openPorts := range openPortsByTarget {
		if len(openPorts) > 0 {
			// Sort openPorts for consistent output if necessary
			// sort.Ints(openPorts)
			result := TCPPortDiscoveryResult{Target: target, OpenPorts: openPorts}
			outputChan <- engine.ModuleOutput{
				FromModuleName: m.meta.ID,
				DataKey:        m.meta.Produces[0], // "discovery.open_tcp_ports"
				Data:           result,
				Timestamp:      time.Now(),
				Target:         target,
			}
			fmt.Printf("[INFO] Module '%s': Target %s - Open TCP Ports: %v\n", m.meta.Name, target, openPorts)
		}
	}
	// If no open ports were found for any target, we might still want to send an empty aggregate or signal completion.
	// The current logic sends per-target results, so if all targets have no open ports, nothing is sent from this loop.
	// Consider if an explicit "no open ports found for any target" message is needed.

	fmt.Printf("[INFO] Module '%s' (instance: %s): TCP Port Discovery completed.\n", m.meta.Name, m.meta.ID)
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

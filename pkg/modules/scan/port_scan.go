// pkg/modules/scan/port_scan.go
package scan

import (
	"context"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/pentora-ai/pentora/pkg/engine"
	"github.com/pentora-ai/pentora/pkg/modules/discovery"
	"github.com/pentora-ai/pentora/pkg/utils"
	"github.com/spf13/cast"
)

// PortScanConfig holds configuration for the TCP port scanning module.
type PortScanConfig struct {
	// Targets will usually come from the input "discovery.live_hosts"
	// but can be a fallback if no input is provided.
	Targets     []string      `mapstructure:"targets"`
	Ports       string        `mapstructure:"ports"` // Comma-separated, ranges: "22,80,443,1000-1024"
	Timeout     time.Duration `mapstructure:"timeout"`
	Concurrency int           `mapstructure:"concurrency"`
	ScanType    string        `mapstructure:"scan_type"` // e.g., "tcp_connect" (for now, only this)
}

// PortStatusInfo holds the result for a single port on a target.
type PortStatusInfo struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Status   string `json:"status"`          // "open", "closed", "filtered" (filtered might be hard with just net.Dial)
	Protocol string `json:"protocol"`        // "tcp" for now
	Error    string `json:"error,omitempty"` // Error message if scan failed for this port
}

// PortScanModule performs TCP connect scans on specified ports.
type PortScanModule struct {
	meta   engine.ModuleMetadata
	config PortScanConfig
}

func newPortScanModule() *PortScanModule {
	// Default configuration values
	defaultConfig := PortScanConfig{
		Ports:       "21,22,23,25,53,80,110,143,443,445,3306,3389,5432,5900,8080,8443", // Common ports
		Timeout:     1 * time.Second,
		Concurrency: 100,
		ScanType:    "tcp_connect",
	}

	return &PortScanModule{
		meta: engine.ModuleMetadata{
			ID:          "tcp-connect-scan-instance",
			Name:        "tcp-connect-scan",
			Version:     "0.1.0",
			Description: "Performs TCP connect scans to identify open ports on target hosts.",
			Type:        engine.ScanModuleType,
			Author:      "Pentora Team",
			Tags:        []string{"scan", "port", "tcp", "connect"},
			Produces:    []string{"scan.port_status"}, // Each output is a PortStatusInfo
			Consumes:    []string{"discovery.live_hosts", "config.targets_override"},
			ConfigSchema: map[string]engine.ParameterDefinition{
				"targets":     {Description: "Optional override for target IPs/CIDRs. Usually consumed from 'discovery.live_hosts'.", Type: "[]string", Required: false},
				"ports":       {Description: "Comma-separated list of ports and/or ranges (e.g., '22,80,1000-1024').", Type: "string", Required: false, Default: defaultConfig.Ports},
				"timeout":     {Description: "Timeout for each port connection attempt (e.g., '1s').", Type: "duration", Required: false, Default: defaultConfig.Timeout.String()},
				"concurrency": {Description: "Number of concurrent port scanning goroutines per host.", Type: "int", Required: false, Default: defaultConfig.Concurrency},
				"scan_type":   {Description: "Type of scan to perform (currently only 'tcp_connect').", Type: "string", Required: false, Default: defaultConfig.ScanType},
			},
		},
		config: defaultConfig,
	}
}

func (m *PortScanModule) Metadata() engine.ModuleMetadata {
	return m.meta
}

func (m *PortScanModule) Init(configMap map[string]interface{}) error {
	cfg := m.config // Start with defaults

	if targetsVal, ok := configMap["targets"]; ok {
		cfg.Targets = cast.ToStringSlice(targetsVal)
	}
	if portsStr, ok := configMap["ports"].(string); ok && portsStr != "" {
		cfg.Ports = portsStr
	}
	if timeoutStr, ok := configMap["timeout"].(string); ok {
		if dur, err := time.ParseDuration(timeoutStr); err == nil {
			cfg.Timeout = dur
		} else {
			fmt.Fprintf(os.Stderr, "[WARN] Module '%s': Invalid 'timeout' format: '%s'. Using default: %s\n", m.meta.Name, timeoutStr, cfg.Timeout)
		}
	}
	if concurrencyVal, ok := configMap["concurrency"]; ok {
		cfg.Concurrency = cast.ToInt(concurrencyVal)
	}
	if scanTypeStr, ok := configMap["scan_type"].(string); ok && scanTypeStr != "" {
		// For now, only tcp_connect is supported. Add validation if more types are added.
		if scanTypeStr == "tcp_connect" {
			cfg.ScanType = scanTypeStr
		} else {
			fmt.Fprintf(os.Stderr, "[WARN] Module '%s': Invalid 'scan_type': '%s'. Using default: %s\n", m.meta.Name, scanTypeStr, cfg.ScanType)
		}
	}

	if cfg.Concurrency < 1 {
		cfg.Concurrency = 1
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 1 * time.Second
	}

	m.config = cfg
	fmt.Printf("[DEBUG] Module '%s' initialized. Final Config: %+v\n", m.meta.Name, m.config)
	return nil
}

func (m *PortScanModule) Execute(ctx context.Context, inputs map[string]interface{}, outputChan chan<- engine.ModuleOutput) error {
	var targetsToScan []string
	if liveHostsResult, ok := inputs["discovery.live_hosts"]; ok {
		if discoveryResult, ok := liveHostsResult.(discovery.ICMPPingDiscoveryResult); ok { // Use the concrete type from your discovery module
			targetsToScan = discoveryResult.LiveHosts
			fmt.Printf("[INFO] Module '%s': Received %d live hosts from discovery module.\n", m.meta.Name, len(targetsToScan))
		} else if strSlice, ok := liveHostsResult.([]string); ok { // Allow direct string slice input
			targetsToScan = strSlice
			fmt.Printf("[INFO] Module '%s': Received %d targets directly as input.\n", m.meta.Name, len(targetsToScan))
		} else {
			fmt.Fprintf(os.Stderr, "[WARN] Module '%s': Invalid data type for 'discovery.live_hosts' input: %T. Falling back to config targets.\n", m.meta.Name, liveHostsResult)
		}
	}

	if len(targetsToScan) == 0 && len(m.config.Targets) > 0 {
		fmt.Printf("[INFO] Module '%s': No live hosts from input, using targets from module config: %v\n", m.meta.Name, m.config.Targets)
		targetsToScan = utils.ParseAndExpandTargets(m.config.Targets) // Use the same robust parser
	}

	if len(targetsToScan) == 0 {
		err := fmt.Errorf("module '%s': no targets to scan", m.meta.Name)
		// Send an empty result or an error marker if preferred
		outputChan <- engine.ModuleOutput{FromModuleName: m.meta.ID, DataKey: m.meta.Produces[0], Error: err, Timestamp: time.Now()}
		return err // Indicate module failure if no targets
	}

	portsToScan, err := parsePortsString(m.config.Ports)
	if err != nil {
		err = fmt.Errorf("module '%s': error parsing ports string '%s': %w", m.meta.Name, m.config.Ports, err)
		outputChan <- engine.ModuleOutput{FromModuleName: m.meta.ID, DataKey: m.meta.Produces[0], Error: err, Timestamp: time.Now()}
		return err
	}
	if len(portsToScan) == 0 {
		err = fmt.Errorf("module '%s': no ports specified or parsed for scanning", m.meta.Name)
		outputChan <- engine.ModuleOutput{FromModuleName: m.meta.ID, DataKey: m.meta.Produces[0], Error: err, Timestamp: time.Now()}
		return err
	}

	fmt.Printf("[INFO] Module '%s': Starting TCP Connect Scan for %d targets on %d ports. Concurrency per host: %d, Timeout per port: %s\n",
		m.meta.Name, len(targetsToScan), len(portsToScan), m.config.Concurrency, m.config.Timeout)

	var outerWg sync.WaitGroup // To wait for all hosts to be processed

	for _, targetIP := range targetsToScan {
		select {
		case <-ctx.Done():
			fmt.Printf("[INFO] Module '%s': Main context cancelled. Aborting further host scans.\n", m.meta.Name)
			return ctx.Err()
		default:
		}

		outerWg.Add(1)
		go func(ip string) { // Goroutine per host
			defer outerWg.Done()

			var hostWg sync.WaitGroup
			sem := make(chan struct{}, m.config.Concurrency)

			for _, port := range portsToScan {
				select {
				case <-ctx.Done(): // Check before starting new port scan for this host
					return
				default:
				}

				hostWg.Add(1)
				sem <- struct{}{}

				go func(p int) {
					defer hostWg.Done()
					defer func() { <-sem }()

					address := fmt.Sprintf("%s:%d", ip, p)
					conn, err := net.DialTimeout("tcp", address, m.config.Timeout)

					status := "closed"
					// errMsg := ""
					if err == nil {
						status = "open"
						conn.Close()
					} else {
						// Basic error type checking for "filtered" (can be improved)
						if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
							status = "filtered" // Or could still be "closed" if it's just slow
						}
						// errMsg = err.Error() // Optionally include error for closed/filtered
					}

					// Non-blocking send to outputChan or use a buffered channel
					// If outputChan is unbuffered and orchestrator is slow, this could block.
					// Consider making outputChan buffered in the orchestrator.
					select {
					case outputChan <- engine.ModuleOutput{
						FromModuleName: m.meta.ID,
						DataKey:        m.meta.Produces[0], // "scan.port_status"
						Target:         ip,                 // Associate with the target IP
						Data: PortStatusInfo{
							IP:       ip,
							Port:     p,
							Status:   status,
							Protocol: "tcp",
							// Error: errMsg, // Uncomment if you want to include error messages
						},
						Timestamp: time.Now(),
					}:
					case <-ctx.Done(): // If main context is done, stop sending
						return
					}
				}(port)
			}
			hostWg.Wait() // Wait for all ports on this host to be scanned
		}(targetIP)
	}

	outerWg.Wait() // Wait for all hosts to be processed
	fmt.Printf("[INFO] Module '%s': TCP Connect Scan completed for all targets.\n", m.meta.Name)
	return nil
}

// PortScanModuleFactory creates a new PortScanModule instance.
func PortScanModuleFactory() engine.Module {
	return newPortScanModule()
}

func init() {
	engine.RegisterModuleFactory("tcp-connect-scan", PortScanModuleFactory)
}

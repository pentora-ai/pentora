// pkg/modules/discovery/icmp_ping.go
// Package discovery provides various host discovery modules.
package discovery

import (
	// Required for bytes.Compare
	"context"
	"fmt"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/go-ping/ping"
	"github.com/pentora-ai/pentora/pkg/engine" // Assuming your core module interfaces are in pkg/engine
	"github.com/pentora-ai/pentora/pkg/utils"
	"github.com/spf13/cast"
)

// ICMPPingDiscoveryResult stores the outcome of the ping discovery.
type ICMPPingDiscoveryResult struct {
	LiveHosts []string `json:"live_hosts"`
}

// ICMPPingDiscoveryConfig holds configuration for the ICMP ping module.
type ICMPPingDiscoveryConfig struct {
	Targets       []string      `json:"targets"`
	Timeout       time.Duration `json:"timeout"`        // Overall timeout for all pings to a single host
	Count         int           `json:"count"`          // Number of echo requests per host
	Interval      time.Duration `json:"interval"`       // Interval between sending each echo request
	PacketTimeout time.Duration `json:"packet_timeout"` // Timeout for receiving a reply for each packet (used by go-ping Pinger.Timeout)
	Privileged    bool          `json:"privileged"`
	Concurrency   int           `json:"concurrency"`
	AllowLoopback bool          `json:"allow_loopback"`
}

// Pinger is an interface for the ping library.
type Pinger interface {
	Run() error
	Stop()
	Statistics() *ping.Statistics

	SetPrivileged(bool)
	SetNetwork(string)
	SetAddr(string)
	SetCount(int)
	SetInterval(time.Duration)
	SetTimeout(time.Duration)
	GetTimeout() time.Duration
}

// Pinger is an interface for the ping library.
type pingerFactoryFunc func(ip string) (Pinger, error)

// ICMPPingDiscoveryModule implements the engine.Module interface for ICMP host discovery.
type ICMPPingDiscoveryModule struct {
	meta          engine.ModuleMetadata
	config        ICMPPingDiscoveryConfig
	pingerFactory pingerFactoryFunc
}

// newICMPPingDiscoveryModule is the internal constructor for the module.
// It sets up metadata and initializes the config with default values.
func newICMPPingDiscoveryModule() *ICMPPingDiscoveryModule {
	// Default configuration values are set here.
	// These will be used if not overridden by the Init method.
	defaultConfig := ICMPPingDiscoveryConfig{
		Timeout:       3 * time.Second,
		Count:         1,
		Interval:      1 * time.Second,
		PacketTimeout: 1 * time.Second, // This will be used for pinger.Timeout
		Privileged:    false,
		Concurrency:   50,
		AllowLoopback: false,
	}

	return &ICMPPingDiscoveryModule{
		meta: engine.ModuleMetadata{
			ID:          "icmp-ping-discovery-instance", // Placeholder; actual instance ID is set by orchestrator/DAG
			Name:        "icmp-ping-discovery",          // Module type name, used by the factory
			Version:     "0.1.0",                        // Incremented version for clarity
			Description: "Detects live hosts using ICMP echo requests via the go-ping library.",
			Type:        engine.DiscoveryModuleType,
			Author:      "Pentora Team",
			Tags:        []string{"discovery", "host", "icmp", "ping"},
			Produces:    []string{"discovery.live_hosts"},
			Consumes:    []string{"config.targets"}, // Example: Consumes a list of targets from global config or previous stage
			ConfigSchema: map[string]engine.ParameterDefinition{
				"targets":        {Description: "List of IPs, CIDRs, or ranges to ping.", Type: "[]string", Required: false /* Can be provided by input */},
				"timeout":        {Description: "Overall timeout for all ping attempts to a single host (e.g., '3s'). This is for the module's management of the ping operation.", Type: "duration", Required: false, Default: defaultConfig.Timeout.String()},
				"count":          {Description: "Number of echo requests to send to each host.", Type: "int", Required: false, Default: defaultConfig.Count},
				"interval":       {Description: "Interval between sending each echo request (e.g., '1s').", Type: "duration", Required: false, Default: defaultConfig.Interval.String()},
				"packet_timeout": {Description: "Timeout for receiving a reply for each ping packet (e.g., '1s'). Used for Pinger.Timeout.", Type: "duration", Required: false, Default: defaultConfig.PacketTimeout.String()},
				"privileged":     {Description: "Set to true to attempt to use raw sockets (requires root/admin privileges).", Type: "bool", Required: false, Default: defaultConfig.Privileged},
				"concurrency":    {Description: "Number of concurrent ping operations.", Type: "int", Required: false, Default: defaultConfig.Concurrency},
				"allow_loopback": {Description: "Set to true to allow pinging loopback addresses (e.g., 127.0.0.1).", Type: "bool", Required: false, Default: defaultConfig.AllowLoopback},
			},
		},
		config: defaultConfig, // Initialize with defaults
		pingerFactory: func(ip string) (Pinger, error) {
			p, err := ping.NewPinger(ip)
			if err != nil {
				return nil, err
			}
			return &realPingerAdapter{p: p}, nil
		},
	}
}

// Metadata returns the module's metadata.
func (m *ICMPPingDiscoveryModule) Metadata() engine.ModuleMetadata {
	return m.meta
}

// Init initializes the module with the given configuration map.
// It parses the map and populates the module's config struct, overriding defaults.
func (m *ICMPPingDiscoveryModule) Init(configMap map[string]interface{}) error {
	// Start with default config values already set by newICMPPingDiscoveryModule
	cfg := m.config

	if targetsVal, ok := configMap["targets"]; ok {
		cfg.Targets = cast.ToStringSlice(targetsVal)
	}
	if timeoutStr, ok := configMap["timeout"].(string); ok {
		if dur, err := time.ParseDuration(timeoutStr); err == nil {
			cfg.Timeout = dur
		} else {
			fmt.Printf("[WARN] Module '%s': Invalid 'timeout' format in config: '%s'. Using default: %s\n", m.meta.Name, timeoutStr, cfg.Timeout)
		}
	}
	if countVal, ok := configMap["count"]; ok {
		cfg.Count = cast.ToInt(countVal)
	}
	if intervalStr, ok := configMap["interval"].(string); ok {
		if dur, err := time.ParseDuration(intervalStr); err == nil {
			cfg.Interval = dur
		} else {
			fmt.Printf("[WARN] Module '%s': Invalid 'interval' format in config: '%s'. Using default: %s\n", m.meta.Name, intervalStr, cfg.Interval)
		}
	}
	if packetTimeoutStr, ok := configMap["packet_timeout"].(string); ok {
		if dur, err := time.ParseDuration(packetTimeoutStr); err == nil {
			cfg.PacketTimeout = dur
		} else {
			fmt.Printf("[WARN] Module '%s': Invalid 'packet_timeout' format in config: '%s'. Using default: %s\n", m.meta.Name, packetTimeoutStr, cfg.PacketTimeout)
		}
	}
	if privilegedVal, ok := configMap["privileged"]; ok {
		cfg.Privileged = cast.ToBool(privilegedVal)
	}
	if concurrencyVal, ok := configMap["concurrency"]; ok {
		cfg.Concurrency = cast.ToInt(concurrencyVal)
	}
	if allowLoopbackVal, ok := configMap["allow_loopback"]; ok {
		cfg.AllowLoopback = cast.ToBool(allowLoopbackVal)
	}

	// Validate and sanitize config values
	if cfg.Count < 1 {
		fmt.Printf("[WARN] Module '%s': Ping count in config is < 1 (%d). Setting to 1.\n", m.meta.Name, cfg.Count)
		cfg.Count = 1
	}
	if cfg.Concurrency < 1 {
		fmt.Printf("[WARN] Module '%s': Concurrency in config is < 1 (%d). Setting to 1.\n", m.meta.Name, cfg.Concurrency)
		cfg.Concurrency = 1
	}
	if cfg.Timeout <= 0 { // Ensure overall timeout is also positive
		cfg.Timeout = 3 * time.Second // A sensible fallback
		fmt.Printf("[WARN] Module '%s': Invalid 'timeout'. Setting to default: %s\n", m.meta.Name, cfg.Timeout)
	}
	if cfg.PacketTimeout <= 0 {
		cfg.PacketTimeout = cfg.Timeout // Fallback if packet_timeout is invalid or not set appropriately
		fmt.Printf("[WARN] Module '%s': Invalid 'packet_timeout'. Using overall 'timeout' value: %s\n", m.meta.Name, cfg.PacketTimeout)
	}

	// Handle privileged mode warning/downgrade for non-Windows OS
	if cfg.Privileged && runtime.GOOS != "windows" {
		if os.Geteuid() != 0 {
			fmt.Printf("[WARN] Module '%s': Privileged ping requested, but process is not running as root. Falling back to unprivileged ping.\n", m.meta.Name)
			cfg.Privileged = false
		}
	} else if cfg.Privileged && runtime.GOOS == "windows" {
		// Inform the user about Windows behavior with privileged pings
		fmt.Printf("[INFO] Module '%s': Privileged mode for go-ping on Windows may rely on ICMP.DLL rather than raw sockets.\n", m.meta.Name)
	}

	m.config = cfg // Assign the processed config back to the module
	fmt.Printf("[DEBUG] Module '%s' initialized. Final Config: %+v\n", m.meta.Name, m.config)
	return nil
}

// Execute performs the host discovery using ICMP pings based on the initialized configuration.
func (m *ICMPPingDiscoveryModule) Execute(ctx context.Context, inputs map[string]interface{}, outputChan chan<- engine.ModuleOutput) error {
	var targetsToProcess []string

	// Determine targets: prefer inputs, fallback to module config
	if inputTargets, ok := inputs["targets"].([]string); ok && len(inputTargets) > 0 {
		targetsToProcess = utils.ParseAndExpandTargets(inputTargets)
	} else if len(m.config.Targets) > 0 {
		targetsToProcess = utils.ParseAndExpandTargets(m.config.Targets)
	} else {
		err := fmt.Errorf("module '%s': no targets specified (neither in input nor in module config)", m.meta.Name)
		outputChan <- engine.ModuleOutput{FromModuleName: m.meta.ID, Error: err, Timestamp: time.Now()}
		return err // Return error to orchestrator to indicate module failure
	}
	// fmt.Printf("[DEBUG-EXEC] Module '%s': Targets after parseAndExpandTargets: %v\n", m.meta.Name, targetsToProcess)

	finalTargetsToScan := []string{}
	if !m.config.AllowLoopback {
		for _, ipStr := range targetsToProcess {
			ip := net.ParseIP(ipStr)
			if ip != nil && ip.IsLoopback() {
				// fmt.Printf("[INFO] Module '%s': Skipping loopback address %s as per configuration (AllowLoopback=false).\n", m.meta.Name, ipStr)
				continue
			}
			finalTargetsToScan = append(finalTargetsToScan, ipStr)
		}
	} else {
		finalTargetsToScan = targetsToProcess
	}
	// fmt.Printf("[DEBUG-EXEC] Module '%s': Targets after loopback filtering (finalTargetsToScan): %v\n", m.meta.Name, finalTargetsToScan)

	if len(finalTargetsToScan) == 0 {
		msg := "effective target list is empty after all filters"
		if !m.config.AllowLoopback && len(targetsToProcess) > 0 {
			allWereLoopback := true
			for _, ipStr := range targetsToProcess {
				ip := net.ParseIP(ipStr)
				if ip == nil || !ip.IsLoopback() {
					allWereLoopback = false
					break
				}
			}
			if allWereLoopback {
				msg = "all specified targets were loopback addresses and loopback scanning is disabled"
			}
		}
		// Send an info output that no targets are left, but don't necessarily error the module itself
		// unless the orchestrator should stop on this.
		outputChan <- engine.ModuleOutput{
			FromModuleName: m.meta.ID,
			DataKey:        m.meta.Produces[0], // discovery.live_hosts
			Data:           ICMPPingDiscoveryResult{LiveHosts: []string{}},
			Timestamp:      time.Now(),
			Error:          fmt.Errorf("module '%s': %s", m.meta.Name, msg), // Informative error in output
		}
		fmt.Printf("[INFO] Module '%s': %s. No hosts to ping.\n", m.meta.Name, msg)
		return nil // Module itself didn't fail, just had no work.
	}

	var liveHosts []string
	var mu sync.Mutex // Protects liveHosts
	var wg sync.WaitGroup
	sem := make(chan struct{}, m.config.Concurrency)

	fmt.Printf("[INFO] Module '%s': Starting ICMP Ping for %d targets. Concurrency: %d, Count: %d, Interval: %s, PktTimeout: %s, Privileged: %v\n",
		m.meta.Name, len(finalTargetsToScan), m.config.Concurrency, m.config.Count, m.config.Interval, m.config.PacketTimeout, m.config.Privileged)

	for _, targetIP := range finalTargetsToScan {
		select {
		case <-ctx.Done():
			fmt.Printf("[INFO] Module '%s': Main context cancelled. Aborting further pings. Found %d live hosts so far.\n", m.meta.Name, len(liveHosts))
			return ctx.Err() // Propagate cancellation
		default:
		}

		wg.Add(1)
		sem <- struct{}{} // Acquire a spot in the semaphore

		go func(ip string) {
			defer wg.Done()
			defer func() { <-sem }() // Release spot

			select {
			case <-ctx.Done(): // Check parent context again before starting
				return
			default:
			}

			pinger, err := m.pingerFactory(ip)
			if err != nil {
				// fmt.Printf("[DEBUG] Module '%s': Failed to create pinger for %s: %v\n", m.meta.Name, ip, err)
				return
			}

			pinger.SetPrivileged(m.config.Privileged)
			pinger.SetCount(m.config.Count)
			pinger.SetInterval(m.config.Interval)
			pinger.SetTimeout(m.config.PacketTimeout) // go-ping's Pinger.Timeout is for the entire operation for this pinger instance

			// Create a context for this specific ping operation that respects the overall module context
			// and the pinger's configured timeout (plus a small buffer for cleanup).
			opCtx, opCancel := context.WithTimeout(ctx, pinger.GetTimeout()+(500*time.Millisecond))
			defer opCancel()

			// Ensure the pinger stops if the operation context is done.
			// This is crucial if pinger.Run() doesn't immediately react to parent ctx cancellation in all cases.
			go func() {
				<-opCtx.Done()
				if opCtx.Err() == context.DeadlineExceeded || opCtx.Err() == context.Canceled {
					pinger.Stop()
				}
			}()

			err = pinger.Run()           // This is a blocking call.
			stats := pinger.Statistics() // Get stats regardless of error from Run()

			if opCtx.Err() != nil { // Check if our operation context timed out or was cancelled
				// fmt.Printf("[DEBUG] Module '%s': Ping operation for %s context done: %v\n", m.meta.Name, ip, opCtx.Err())
				return
			}
			// Error from pinger.Run() itself might indicate network issues other than timeout,
			// or issues setting up the ping (e.g. privilege problems not caught earlier).
			// if err != nil {
			// 	fmt.Printf("[DEBUG] Module '%s': Pinger.Run() for %s reported error: %v. Stats: %+v\n", m.meta.Name, ip, err, stats)
			// }

			if stats.PacketsRecv > 0 {
				mu.Lock()
				liveHosts = append(liveHosts, ip)
				mu.Unlock()
			} else {
				// fmt.Printf("[DEBUG] Module '%s': Host %s did not respond (Sent: %d, Recv: %d, Loss: %.0f%%)\n",
				//  m.meta.Name, ip, stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
			}
		}(targetIP)
	}

	wg.Wait()

	resultData := ICMPPingDiscoveryResult{LiveHosts: liveHosts}
	outputChan <- engine.ModuleOutput{
		FromModuleName: m.meta.ID,
		DataKey:        m.meta.Produces[0], // "discovery.live_hosts"
		Data:           resultData,
		Timestamp:      time.Now(),
	}

	fmt.Printf("[INFO] Module '%s': ICMP Ping Discovery completed. Found %d live hosts out of %d processed targets.\n", m.meta.Name, len(liveHosts), len(finalTargetsToScan))
	return nil
}

// ICMPPingModuleFactory creates a new ICMPPingDiscoveryModule instance.
// This factory function is what's registered with the core engine.
func ICMPPingModuleFactory() engine.Module {
	return newICMPPingDiscoveryModule()
}

func init() {
	// Register the module factory with Pentora's core module registry.
	// The name "icmp-ping-discovery" will be used in DAG definitions to instantiate this module.
	engine.RegisterModuleFactory("icmp-ping-discovery", ICMPPingModuleFactory)
}

// internal adapter: wraps github.com/go-ping/ping.Pinger to implement our Pinger interface
type realPingerAdapter struct {
	p *ping.Pinger
}

func (r *realPingerAdapter) Run() error                   { return r.p.Run() }
func (r *realPingerAdapter) Stop()                        { r.p.Stop() }
func (r *realPingerAdapter) Statistics() *ping.Statistics { return r.p.Statistics() }

func (r *realPingerAdapter) SetPrivileged(v bool)        { r.p.SetPrivileged(v) }
func (r *realPingerAdapter) SetNetwork(n string)         { r.p.SetNetwork(n) }
func (r *realPingerAdapter) SetAddr(a string)            { r.p.SetAddr(a) }
func (r *realPingerAdapter) SetCount(c int)              { r.p.Count = c }
func (r *realPingerAdapter) SetInterval(i time.Duration) { r.p.Interval = i }
func (r *realPingerAdapter) SetTimeout(t time.Duration)  { r.p.Timeout = t }
func (r *realPingerAdapter) GetTimeout() time.Duration   { return r.p.Timeout }

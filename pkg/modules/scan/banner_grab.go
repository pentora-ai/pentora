// pkg/modules/scan/banner_grab.go
// Package scan provides modules related to active network scanning.
package scan

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pentora-ai/pentora/pkg/engine" // Your engine/core package
	"github.com/spf13/cast"
)

// BannerGrabConfig holds configuration for the banner grabbing module.
type BannerGrabConfig struct {
	// Input will typically be PortStatusInfo from PortScanModule
	ReadTimeout           time.Duration `mapstructure:"read_timeout"`             // Timeout for reading banner data from a connection
	ConnectTimeout        time.Duration `mapstructure:"connect_timeout"`          // Timeout for establishing the connection (if re-dialing)
	BufferSize            int           `mapstructure:"buffer_size"`              // Size of the buffer to read banner data
	Concurrency           int           `mapstructure:"concurrency"`              // Number of concurrent banner grabbing operations
	SendProbes            bool          `mapstructure:"send_probes"`              // Whether to send basic probes (e.g., HTTP GET)
	TLSInsecureSkipVerify bool          `mapstructure:"tls_insecure_skip_verify"` // For TLS connections, skip cert verification (not recommended for production)
	// Future: Define specific probes for common ports
	// HTTPProbes     []string      `mapstructure:"http_probes"`  // e.g., ["GET / HTTP/1.1\r\nHost: {HOST}\r\n\r\n", "HEAD / HTTP/1.0\r\n\r\n"]
	// GenericProbes  []string      `mapstructure:"generic_probes"`// e.g., ["\r\n\r\n", "HELP\r\n"]
}

// BannerGrabResult holds the banner information for a specific port.
// This will be the 'Data' in ModuleOutput with DataKey "service.banner.raw".
type BannerGrabResult struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // "tcp" for now
	Banner   string `json:"banner"`
	IsTLS    bool   `json:"is_tls"` // Indicates if the banner was grabbed over a TLS handshake
	Error    string `json:"error,omitempty"`
}

// BannerGrabModule attempts to grab banners from open TCP ports.
type BannerGrabModule struct {
	meta   engine.ModuleMetadata
	config BannerGrabConfig
}

// newBannerGrabModule is the internal constructor for the BannerGrabModule.
func newBannerGrabModule() *BannerGrabModule {
	defaultConfig := BannerGrabConfig{
		ReadTimeout:           3 * time.Second,
		ConnectTimeout:        2 * time.Second, // Should be less than or equal to PortScanModule's timeout for that port
		BufferSize:            2048,            // Increased buffer size
		Concurrency:           50,
		SendProbes:            true,  // Send basic probes by default
		TLSInsecureSkipVerify: false, // Default to secure TLS verification
	}

	return &BannerGrabModule{
		meta: engine.ModuleMetadata{
			ID:          "banner-grab-instance",
			Name:        "banner-grabber",
			Version:     "0.1.0",
			Description: "Grabs banners from open TCP ports, attempting generic and HTTP probes.",
			Type:        engine.ScanModuleType, // Could also be a more specific "fingerprint" type
			Author:      "Pentora Team",
			Tags:        []string{"scan", "banner", "fingerprint", "tcp"},
			Produces:    []string{"service.banner.raw"}, // Each output is a BannerGrabResult
			Consumes:    []string{"scan.port_status"},   // Consumes results from port scanner
			ConfigSchema: map[string]engine.ParameterDefinition{
				"read_timeout":    {Description: "Timeout for reading banner data from an open port (e.g., '3s').", Type: "duration", Required: false, Default: defaultConfig.ReadTimeout.String()},
				"connect_timeout": {Description: "Timeout for establishing connection if re-dialing (e.g., '2s').", Type: "duration", Required: false, Default: defaultConfig.ConnectTimeout.String()},
				"buffer_size":     {Description: "Size of the buffer (in bytes) for reading banner data.", Type: "int", Required: false, Default: defaultConfig.BufferSize},
				"concurrency":     {Description: "Number of concurrent banner grabbing operations.", Type: "int", Required: false, Default: defaultConfig.Concurrency},
				"send_probes":     {Description: "Whether to send basic HTTP/generic probes to elicit banners.", Type: "bool", Required: false, Default: defaultConfig.SendProbes},
			},
		},
		config: defaultConfig,
	}
}

// Metadata returns the module's descriptive metadata.
func (m *BannerGrabModule) Metadata() engine.ModuleMetadata {
	return m.meta
}

// Init initializes the module with the given configuration map.
func (m *BannerGrabModule) Init(configMap map[string]interface{}) error {
	cfg := m.config // Start with defaults

	if readTimeoutStr, ok := configMap["read_timeout"].(string); ok {
		if dur, err := time.ParseDuration(readTimeoutStr); err == nil {
			cfg.ReadTimeout = dur
		} else {
			fmt.Fprintf(os.Stderr, "[WARN] Module '%s': Invalid 'read_timeout': '%s'. Using default: %s\n", m.meta.Name, readTimeoutStr, cfg.ReadTimeout)
		}
	}
	if connectTimeoutStr, ok := configMap["connect_timeout"].(string); ok {
		if dur, err := time.ParseDuration(connectTimeoutStr); err == nil {
			cfg.ConnectTimeout = dur
		} else {
			fmt.Fprintf(os.Stderr, "[WARN] Module '%s': Invalid 'connect_timeout': '%s'. Using default: %s\n", m.meta.Name, connectTimeoutStr, cfg.ConnectTimeout)
		}
	}
	if bufferSizeVal, ok := configMap["buffer_size"]; ok {
		cfg.BufferSize = cast.ToInt(bufferSizeVal)
	}
	if concurrencyVal, ok := configMap["concurrency"]; ok {
		cfg.Concurrency = cast.ToInt(concurrencyVal)
	}
	if sendProbesVal, ok := configMap["send_probes"]; ok {
		cfg.SendProbes = cast.ToBool(sendProbesVal)
	}
	if tlsInsecureSkipVerify, ok := configMap["tls_insecure_skip_verify"].(bool); ok {
		cfg.TLSInsecureSkipVerify = cast.ToBool(tlsInsecureSkipVerify)
	}

	// Sanitize
	if cfg.ReadTimeout <= 0 {
		cfg.ReadTimeout = 3 * time.Second
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = 2 * time.Second
	}
	if cfg.BufferSize <= 0 || cfg.BufferSize > 16384 { // Max 16KB buffer
		cfg.BufferSize = 2048
	}
	if cfg.Concurrency < 1 {
		cfg.Concurrency = 1
	}

	m.config = cfg
	fmt.Printf("[DEBUG] Module '%s' initialized. Final Config: %+v\n", m.meta.Name, m.config)
	return nil
}

// isPotentiallyHTTP checks if a port is commonly used for HTTP/HTTPS.
func isPotentiallyHTTP(port int) bool {
	switch port {
	case 80, 81, 88, 443, 8000, 8008, 8080, 8081, 8443, 9080, 9443:
		return true
	default:
		return false
	}
}

// isPotentiallyTLS checks if a port is commonly used for TLS services.
// This is a heuristic and not exhaustive.
func isPotentiallyTLS(port int) bool {
	switch port {
	case 443, 8443, 990, 992, 993, 995, 587, 465, 636: // Common TLS ports (HTTPS, FTPS, SMTPS, IMAPS, POP3S, STARTTLS, LDAPS)
		return true
	default:
		return false
	}
}

// Execute attempts to grab banners from open ports.
// It consumes 'scan.port_status' which should be of type PortStatusInfo.
func (m *BannerGrabModule) Execute(ctx context.Context, inputs map[string]interface{}, outputChan chan<- engine.ModuleOutput) error {
	portStatusInput, ok := inputs["scan.port_status"]
	if !ok {
		// This module depends on port status information.
		// If not critical for the DAG, it could return nil.
		// Otherwise, it's a configuration error in the DAG.
		fmt.Fprintf(os.Stderr, "[WARN] Module '%s': Missing required input 'scan.port_status'. Skipping banner grabbing.\n", m.meta.Name)
		// Send an empty or marker output
		outputChan <- engine.ModuleOutput{FromModuleName: m.meta.ID, DataKey: m.meta.Produces[0], Data: BannerGrabResult{}, Timestamp: time.Now()}
		return nil // Or an error: fmt.Errorf("module '%s': missing required input 'scan.port_status'", m.meta.Name)
	}

	// The input can be a single PortStatusInfo or a slice/stream of them.
	// For this example, let's assume it's a single PortStatusInfo for simplicity in Execute.
	// An orchestrator would typically iterate over a list of open ports and call this for each,
	// or this module would handle a slice of PortStatusInfo.
	// Let's design it to handle a PortStatusInfo as input for a single IP/Port.
	// The orchestrator will feed these one by one or in batches.

	// We'll make this module process one PortStatusInfo at a time, matching a typical DAG node.
	// The orchestrator should ensure this module is called for each open port.
	// An alternative design is for this module to take a list of open ports.
	// For simplicity in this standalone module example, let's assume inputs["scan.port_status"] is ONE PortStatusInfo.
	// In a real DAG, you'd likely get a stream or a list.
	// Let's adjust to expect a PortStatusInfo directly for simplicity in this module's Execute.
	// If it were a list, we'd loop here.

	portInfo, ok := portStatusInput.(PortStatusInfo) // From previous PortScanModule
	if !ok {
		fmt.Fprintf(os.Stderr, "[WARN] Module '%s': Invalid data type for 'scan.port_status' input. Expected PortStatusInfo, got %T. Skipping.\n", m.meta.Name, portStatusInput)
		errMsg := fmt.Errorf("invalid input type %T", portStatusInput)
		outputChan <- engine.ModuleOutput{
			FromModuleName: m.meta.ID,
			DataKey:        m.meta.Produces[0],
			Data: BannerGrabResult{
				IP:    "unknown",
				Port:  0,
				Error: errMsg.Error(),
			},
			Error:     errMsg,
			Timestamp: time.Now(),
		}
		return nil
	}

	if portInfo.Status != "open" {
		// fmt.Printf("[DEBUG] Module '%s': Skipping banner grab for %s:%d as port is not open (status: %s).\n", m.meta.Name, portInfo.IP, portInfo.Port, portInfo.Status)
		return nil // Only grab banners from open ports
	}

	// fmt.Printf("[INFO] Module '%s': Attempting to grab banner from %s:%d.\n", m.meta.Name, portInfo.IP, portInfo.Port)

	var wg sync.WaitGroup
	sem := make(chan struct{}, m.config.Concurrency) // Although we process one portInfo, internal probes might be concurrent

	// For a single portInfo, concurrency is less about multiple hosts/ports here,
	// and more about potentially trying multiple probes concurrently on the same port if designed so.
	// For this version, we'll try probes sequentially.

	wg.Add(1) // Only one "task" for this single portInfo
	sem <- struct{}{}

	go func(pInfo PortStatusInfo) {
		defer wg.Done()
		defer func() { <-sem }()

		address := net.JoinHostPort(pInfo.IP, strconv.Itoa(pInfo.Port))
		var banner string
		var err error
		isTLS := false

		// 1. Attempt a simple TCP read (for non-HTTP, non-TLS services)
		// Only if not a common TLS port, or if SendProbes is off for those.
		if !isPotentiallyTLS(pInfo.Port) || !m.config.SendProbes {
			banner, err = m.grabGenericBanner(ctx, address)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[DEBUG] Generic banner grab for %s failed: %v\n", address, err)
			}
		}

		// 2. If generic banner is empty or failed, and SendProbes is true, try specific probes
		if (banner == "" || err != nil) && m.config.SendProbes {
			if isPotentiallyHTTP(pInfo.Port) {
				// Try HTTP GET probe
				httpBanner, httpErr := m.grabHTTPBanner(ctx, pInfo.IP, pInfo.Port, false)
				if httpErr == nil && httpBanner != "" {
					banner = httpBanner
					err = nil // Clear previous generic error
				} else {
					// If HTTP failed, and it's a common HTTPS port, try HTTPS
					if isPotentiallyTLS(pInfo.Port) { // Typically 443, 8443
						httpsBanner, httpsErr := m.grabHTTPBanner(ctx, pInfo.IP, pInfo.Port, true)
						if httpsErr == nil && httpsBanner != "" {
							banner = httpsBanner
							isTLS = true
							err = nil // Clear previous errors
						} else if httpErr != nil && banner == "" { // Preserve original httpErr if https also fails
							err = httpErr
						} else if httpsErr != nil && banner == "" {
							err = httpsErr
						}
					} else if httpErr != nil && banner == "" {
						err = httpErr // Preserve HTTP error if not trying HTTPS
					}
				}
			} else if isPotentiallyTLS(pInfo.Port) && banner == "" { // Non-HTTP TLS port (e.g. SMTPS, IMAPS)
				tlsBanner, tlsErr := m.grabTLSBanner(ctx, address)
				if tlsErr == nil && tlsBanner != "" {
					banner = tlsBanner
					isTLS = true
					err = nil
				} else if tlsErr != nil && banner == "" {
					err = tlsErr
				}
			}
			// Future: Add more probes for FTP, SMTP, SSH (though SSH usually sends banner first)
		}

		result := BannerGrabResult{
			IP:       pInfo.IP,
			Port:     pInfo.Port,
			Protocol: "tcp",
			Banner:   strings.TrimSpace(banner),
			IsTLS:    isTLS,
		}
		if err != nil {
			result.Error = err.Error()
		}

		select {
		case outputChan <- engine.ModuleOutput{
			FromModuleName: m.meta.ID,
			DataKey:        m.meta.Produces[0], // "service.banner.raw"
			Target:         pInfo.IP,
			Data:           result,
			Timestamp:      time.Now(),
		}:
		case <-ctx.Done():
			return
		}
	}(portInfo)

	wg.Wait()
	// fmt.Printf("[INFO] Module '%s': Banner grabbing attempt for %s:%d completed.\n", m.meta.Name, portInfo.IP, portInfo.Port)
	return nil
}

func (m *BannerGrabModule) grabGenericBanner(ctx context.Context, address string) (string, error) {
	conn, err := net.DialTimeout("tcp", address, m.config.ConnectTimeout)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(m.config.ReadTimeout))
	reader := bufio.NewReader(conn)
	buffer := make([]byte, m.config.BufferSize)
	n, readErr := reader.Read(buffer)

	// Prefer context error if available
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if readErr != nil && readErr != io.EOF {
		return "", readErr
	}
	return string(buffer[:n]), nil
}

func (m *BannerGrabModule) grabHTTPBanner(ctx context.Context, host string, port int, useTLS bool) (string, error) {
	var conn net.Conn
	var err error
	address := net.JoinHostPort(host, strconv.Itoa(port))

	dialer := &net.Dialer{Timeout: m.config.ConnectTimeout}

	if useTLS {
		// #nosec G402 -- InsecureSkipVerify can be a user option in the future if needed for self-signed certs
		// For now, we are not skipping verification. If it's a common need, add a config option.
		tlsConfig := &tls.Config{
			InsecureSkipVerify: m.config.TLSInsecureSkipVerify, // Default to secure
			ServerName:         host,                           // For SNI
		}
		conn, err = tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", address)
	}

	if err != nil {
		return "", err
	}
	defer conn.Close()

	// Simple HTTP GET request
	// Using Host header that matches the target IP/hostname is important for virtual hosting.
	// If 'host' is an IP, some servers might not respond as expected without a proper hostname.
	request := fmt.Sprintf("GET / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\nUser-Agent: PentoraBannerGrabber/0.1\r\n\r\n", host)
	_, err = conn.Write([]byte(request))
	if err != nil {
		return "", err
	}

	conn.SetReadDeadline(time.Now().Add(m.config.ReadTimeout))
	// Read up to BufferSize or until EOF/timeout
	var responseBuilder strings.Builder
	buffer := make([]byte, m.config.BufferSize)
	totalRead := 0

	for {
		select {
		case <-ctx.Done(): // Check context cancellation during read loop
			return responseBuilder.String(), ctx.Err()
		default:
		}

		n, readErr := conn.Read(buffer)
		if n > 0 {
			responseBuilder.Write(buffer[:n])
			totalRead += n
			// Stop if we've filled the buffer to avoid excessively large banners
			// or if we have enough data (e.g., just headers). This can be refined.
			if totalRead >= m.config.BufferSize {
				break
			}
		}
		if readErr != nil {
			if readErr != io.EOF { // EOF is expected when connection is closed by server
				err = readErr
			}
			break // EOF or other error
		}
	}

	response := responseBuilder.String()
	if strings.HasPrefix(response, "HTTP/") {
		statusLine := strings.SplitN(response, "\r\n", 2)[0]
		statusCode := strings.Split(statusLine, " ")[1]

		code, _ := strconv.Atoi(statusCode)

		if code >= 400 {
			return "", fmt.Errorf("HTTP error: %s", statusLine)
		}

		// Special cases requiring HTTPS
		if strings.Contains(response, "Upgrade Required") ||
			strings.Contains(response, "HTTP to HTTPS") {
			return "", fmt.Errorf("server requires HTTPS")
		}
	}

	return response, err // Return last error encountered during read, or nil
}

// grabTLSBanner attempts a TLS handshake and reads initial data.
// This is very basic and might only get server certificate info or an initial TLS alert.
// A more sophisticated approach would involve parsing the TLS handshake.
func (m *BannerGrabModule) grabTLSBanner(ctx context.Context, address string) (string, error) {
	dialer := &net.Dialer{Timeout: m.config.ConnectTimeout}
	// #nosec G402 -- InsecureSkipVerify: false by default. Add config if needed.
	conn, err := tls.DialWithDialer(dialer, "tcp", address, &tls.Config{
		InsecureSkipVerify: m.config.TLSInsecureSkipVerify, // Consider making this configurable for testing self-signed certs
	})
	if err != nil {
		return "", fmt.Errorf("TLS dial error: %w", err)
	}
	defer conn.Close()

	// Attempt to get some info from the handshake state
	// This isn't a "banner" in the traditional sense for many TLS services without an app protocol.
	// For HTTPS, the HTTP banner is grabbed over TLS. For others (SMTPS, IMAPS), the app protocol starts after TLS.
	// This might just confirm TLS is present.

	// Try to read a small amount of data after handshake. Some services send an initial message.
	conn.SetReadDeadline(time.Now().Add(m.config.ReadTimeout))
	buffer := make([]byte, m.config.BufferSize)
	n, readErr := conn.Read(buffer)

	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	var bannerParts []string
	if state := conn.ConnectionState(); state.HandshakeComplete {
		bannerParts = append(bannerParts, fmt.Sprintf("TLSv%x", state.Version))
		if len(state.PeerCertificates) > 0 {
			cert := state.PeerCertificates[0]
			if cert.Subject.CommonName != "" {
				bannerParts = append(bannerParts, fmt.Sprintf("CN=%s", cert.Subject.CommonName))
			}
			if len(cert.DNSNames) > 0 {
				bannerParts = append(bannerParts, fmt.Sprintf("SANs=%s", strings.Join(cert.DNSNames, ",")))
			}
		}
	}

	if n > 0 {
		bannerParts = append(bannerParts, "DATA="+strings.TrimSpace(string(buffer[:n])))
	} else if readErr != nil && readErr != io.EOF {
		// If there's a read error and we have no other TLS info, return the error
		if len(bannerParts) == 0 {
			return "", fmt.Errorf("TLS read error: %w", readErr)
		}
	}

	if len(bannerParts) > 0 {
		return strings.Join(bannerParts, "; "), nil
	}
	// If handshake completed but no data and no specific info, indicate TLS was established
	if conn.ConnectionState().HandshakeComplete {
		return "TLS Handshake Successful", nil
	}

	return "", fmt.Errorf("no data or significant TLS info received")
}

// BannerGrabModuleFactory creates a new BannerGrabModule instance.
func BannerGrabModuleFactory() engine.Module {
	return newBannerGrabModule()
}

func init() {
	engine.RegisterModuleFactory("banner-grabber", BannerGrabModuleFactory)
}

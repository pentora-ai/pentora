// pkg/modules/scan/banner_grab.go
// Package scan provides modules related to active network scanning.
package scan

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pentora-ai/pentora/pkg/engine" // Your engine/core package
	"github.com/pentora-ai/pentora/pkg/fingerprint"
	"github.com/pentora-ai/pentora/pkg/modules/discovery"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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
	IP       string             `json:"ip"`
	Port     int                `json:"port"`
	Protocol string             `json:"protocol"`
	Banner   string             `json:"banner"`
	IsTLS    bool               `json:"is_tls"`
	Error    string             `json:"error,omitempty"`
	Evidence []ProbeObservation `json:"evidence,omitempty"`
}

// ProbeObservation captures the outcome of a single probe execution in the layered banner grabbing flow.
type ProbeObservation struct {
	ProbeID     string          `json:"probe_id"`
	Description string          `json:"description,omitempty"`
	Protocol    string          `json:"protocol,omitempty"`
	IsTLS       bool            `json:"is_tls,omitempty"`
	Duration    time.Duration   `json:"duration_ns,omitempty"`
	Response    string          `json:"response,omitempty"`
	Error       string          `json:"error,omitempty"`
	TLS         *TLSObservation `json:"tls,omitempty"`
}

// TLSObservation holds key facts collected during a TLS handshake so downstream modules
// can reason about certificate metadata without re-dialing the host.
type TLSObservation struct {
	Version        string   `json:"version,omitempty"`
	CipherSuite    string   `json:"cipher_suite,omitempty"`
	ServerName     string   `json:"server_name,omitempty"`
	PeerCommonName string   `json:"peer_common_name,omitempty"`
	PeerDNSNames   []string `json:"peer_dns_names,omitempty"`
}

type commandProbeSpec struct {
	ProbeID         string
	Description     string
	Protocol        string
	Commands        []string
	UseTLS          bool
	SkipInitialRead bool
}

// BannerGrabModule attempts to grab banners from open TCP ports.
type BannerGrabModule struct {
	meta   engine.ModuleMetadata
	config BannerGrabConfig
	logger zerolog.Logger
}

type PortInfo struct {
	*discovery.TCPPortDiscoveryResult
}

// newBannerGrabModule is the internal constructor for the BannerGrabModule.
func newBannerGrabModule() *BannerGrabModule {
	defaultConfig := BannerGrabConfig{
		ReadTimeout:           3 * time.Second,
		ConnectTimeout:        2 * time.Second,
		BufferSize:            2048,
		Concurrency:           50,
		SendProbes:            true,
		TLSInsecureSkipVerify: false,
	}

	return &BannerGrabModule{
		meta: engine.ModuleMetadata{
			ID:          "banner-grab-instance",
			Name:        "banner-grabber",
			Version:     "0.1.0",
			Description: "Grabs banners from open TCP ports, attempting generic and protocol-aware probes.",
			Type:        engine.ScanModuleType,
			Author:      "Pentora Team",
			Tags:        []string{"scan", "banner", "fingerprint", "tcp"},
			Consumes: []engine.DataContractEntry{
				{
					Key:          "discovery.open_tcp_ports",
					DataTypeName: "discovery.TCPPortDiscoveryResult",
					Cardinality:  engine.CardinalityList,
					IsOptional:   false,
					Description:  "List of results, where each item details open TCP ports for a specific target.",
				},
			},
			Produces: []engine.DataContractEntry{
				{
					Key:          "service.banner.tcp",
					DataTypeName: "scan.BannerScanResult",
					Cardinality:  engine.CardinalityList,
					Description:  "List of banners (or errors) captured from TCP services, one result per target/port.",
				},
			},
			ConfigSchema: map[string]engine.ParameterDefinition{
				"read_timeout":    {Description: "Timeout for reading banner data from an open port (e.g., '3s').", Type: "duration", Required: false, Default: defaultConfig.ReadTimeout.String()},
				"connect_timeout": {Description: "Timeout for establishing connection if re-dialing (e.g., '2s').", Type: "duration", Required: false, Default: defaultConfig.ConnectTimeout.String()},
				"buffer_size":     {Description: "Size of the buffer (in bytes) for reading banner data.", Type: "int", Required: false, Default: defaultConfig.BufferSize},
				"concurrency":     {Description: "Number of concurrent banner grabbing operations.", Type: "int", Required: false, Default: defaultConfig.Concurrency},
				"send_probes":     {Description: "Whether to send protocol-specific probes after passive banner capture.", Type: "bool", Required: false, Default: defaultConfig.SendProbes},
			},
			EstimatedCost: 2,
		},
		config: defaultConfig,
	}
}

// Metadata returns the module's descriptive metadata.
func (m *BannerGrabModule) Metadata() engine.ModuleMetadata {
	return m.meta
}

// Init initializes the module with the given configuration map.
func (m *BannerGrabModule) Init(instanceID string, configMap map[string]interface{}) error {
	m.logger = log.With().Str("module", m.meta.Name).Str("instance_id", m.meta.ID).Logger()

	cfg := m.config

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

	if cfg.ReadTimeout <= 0 {
		cfg.ReadTimeout = 3 * time.Second
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = 2 * time.Second
	}
	if cfg.BufferSize <= 0 || cfg.BufferSize > 16384 {
		cfg.BufferSize = 2048
	}
	if cfg.Concurrency < 1 {
		cfg.Concurrency = 1
	}

	m.config = cfg
	m.logger.Debug().Interface("final_config", m.config).Msgf("Module initialized.")
	return nil
}

// TargetPortData represents a target IP and a port to scan.
type TargetPortData struct {
	Target string
	Port   int
}

// Execute attempts to grab banners from open ports.
// It consumes 'discovery.open_tcp_ports' which should be of type PortStatusInfo.
func (m *BannerGrabModule) Execute(ctx context.Context, inputs map[string]interface{}, outputChan chan<- engine.ModuleOutput) error {
	m.logger.Debug().Interface("received_inputs", inputs).Msg("Executing module")

	var scanTasks []TargetPortData

	if rawOpenTCPPorts, ok := inputs["discovery.open_tcp_ports"]; ok {
		m.logger.Debug().Type("type", rawOpenTCPPorts).Msg("Found 'discovery.open_tcp_ports' in inputs")
		if openTCPPortsList, listOk := rawOpenTCPPorts.([]interface{}); listOk {
			for _, item := range openTCPPortsList {
				if portResult, castOk := item.(discovery.TCPPortDiscoveryResult); castOk {
					for _, port := range portResult.OpenPorts {
						scanTasks = append(scanTasks, TargetPortData{Target: portResult.Target, Port: port})
					}
				} else {
					m.logger.Warn().Type("item_type", item).Msg("Item in 'discovery.open_tcp_ports' list is not of expected type discovery.TCPPortDiscoveryResult")
				}
			}
			m.logger.Info().Int("num_target_port_pairs", len(scanTasks)).Msg("Targets and ports loaded from 'discovery.open_tcp_ports' input")
		} else {
			m.logger.Warn().Type("type", rawOpenTCPPorts).Msg("'discovery.open_tcp_ports' input is not a list as expected")
		}
	} else {
		m.logger.Warn().Msg("'discovery.open_tcp_ports' not found in inputs. Banner grabbing will be limited or skipped unless targets/ports provided via other means (not fully implemented in this example).")
	}

	if len(scanTasks) == 0 {
		m.logger.Info().Msg("No target/port pairs to grab banners from. Module execution complete.")
		outputChan <- engine.ModuleOutput{
			FromModuleName: m.meta.ID,
			DataKey:        m.meta.Produces[0].Key,
			Data:           []BannerGrabResult{},
			Timestamp:      time.Now(),
		}
		return nil
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, m.config.Concurrency)
	var resultsMu sync.Mutex
	grabbedBanners := make([]BannerGrabResult, 0, len(scanTasks))

	m.logger.Info().Int("tasks", len(scanTasks)).Int("concurrency", m.config.Concurrency).Msg("Starting banner grabbing")

	for _, task := range scanTasks {
		select {
		case <-ctx.Done():
			m.logger.Info().Msg("Context cancelled. Aborting further banner grabbing.")
			goto endLoop
		default:
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(currentTarget string, currentPort int) {
			defer wg.Done()
			defer func() { <-sem }()

			result := m.runProbes(ctx, currentTarget, currentPort)

			resultsMu.Lock()
			grabbedBanners = append(grabbedBanners, result)
			resultsMu.Unlock()

			select {
			case outputChan <- engine.ModuleOutput{
				FromModuleName: m.meta.ID,
				DataKey:        m.meta.Produces[0].Key,
				Target:         currentTarget,
				Data:           result,
				Timestamp:      time.Now(),
			}:
			case <-ctx.Done():
				return
			}
		}(task.Target, task.Port)
	}

endLoop:
	wg.Wait()
	m.logger.Info().Int("results", len(grabbedBanners)).Msg("Service banner scanning completed.")

	return nil
}

func (m *BannerGrabModule) runProbes(ctx context.Context, target string, port int) BannerGrabResult {
	observations := make([]ProbeObservation, 0, 8)
	bestBanner := ""
	bestIsTLS := false
	var lastError string

	passive := m.runPassiveProbe(ctx, target, port)
	hintAcc := newHintAccumulator()

	catalog, catalogErr := fingerprint.GetProbeCatalog()
	if catalogErr != nil {
		m.logger.Warn().Err(catalogErr).Msg("failed to load probe catalog; continuing with passive banner only")
	} else {
		hintAcc.addAll(portHintsFromCatalog(catalog, port))
	}

	if respHint := protocolHintFromBanner(passive.Response); respHint != "" {
		hintAcc.add(respHint)
	}

	m.collectObservation(&observations, passive, &bestBanner, &bestIsTLS, &lastError)

	if m.config.SendProbes && ctx.Err() == nil && catalogErr == nil {
		candidateProbes := catalog.ProbesFor(port, hintAcc.slice())
		seen := make(map[string]struct{}, len(candidateProbes))

		for _, spec := range candidateProbes {
			if ctx.Err() != nil {
				break
			}
			if _, exists := seen[spec.ID]; exists {
				continue
			}
			seen[spec.ID] = struct{}{}

			obs := m.executeProbeSpec(ctx, target, port, spec)
			if respHint := protocolHintFromBanner(obs.Response); respHint != "" {
				hintAcc.add(respHint)
			}
			m.collectObservation(&observations, obs, &bestBanner, &bestIsTLS, &lastError)
		}
	}

	result := BannerGrabResult{
		IP:       target,
		Port:     port,
		Protocol: "tcp",
		Banner:   strings.TrimSpace(bestBanner),
		IsTLS:    bestIsTLS,
		Evidence: observations,
	}

	if result.Banner == "" && lastError != "" {
		result.Error = lastError
	}

	return result
}

func (m *BannerGrabModule) collectObservation(observations *[]ProbeObservation, obs ProbeObservation, bestBanner *string, bestIsTLS *bool, lastError *string) {
	if obs.ProbeID == "" {
		return
	}

	if obs.Response != "" {
		trimmed := strings.TrimSpace(obs.Response)
		obs.Response = trimmed
		if trimmed != "" {
			if *bestBanner == "" || strings.HasPrefix(obs.ProbeID, "http") || strings.HasPrefix(obs.ProbeID, "https") {
				*bestBanner = trimmed
				*bestIsTLS = obs.IsTLS
			}
			if obs.Error == "" {
				*lastError = ""
			}
		}
	}

	if obs.Error != "" {
		*lastError = obs.Error
	}

	*observations = append(*observations, obs)
}

func (m *BannerGrabModule) runPassiveProbe(ctx context.Context, target string, port int) ProbeObservation {
	obs := ProbeObservation{
		ProbeID:     "tcp-passive",
		Description: "Initial TCP banner read",
		Protocol:    "tcp",
	}

	banner, duration, err := m.grabGenericBanner(ctx, target, port)
	obs.Duration = duration
	obs.Response = banner
	if err != nil {
		obs.Error = err.Error()
	}

	return obs
}

func (m *BannerGrabModule) executeProbeSpec(ctx context.Context, host string, port int, spec fingerprint.ProbeSpec) ProbeObservation {
	commands := prepareProbeCommands(spec, host, port)
	cmdSpec := commandProbeSpec{
		ProbeID:         spec.ID,
		Description:     spec.Description,
		Protocol:        spec.Protocol,
		Commands:        commands,
		UseTLS:          spec.UseTLS,
		SkipInitialRead: spec.SkipInitialRead,
	}
	return m.runCommandProbe(ctx, host, port, cmdSpec)
}

func (m *BannerGrabModule) runCommandProbe(ctx context.Context, host string, port int, spec commandProbeSpec) ProbeObservation {
	obs := ProbeObservation{
		ProbeID:     spec.ProbeID,
		Description: spec.Description,
		Protocol:    spec.Protocol,
		IsTLS:       spec.UseTLS,
	}

	address := net.JoinHostPort(host, strconv.Itoa(port))
	dialer := &net.Dialer{Timeout: m.config.ConnectTimeout}
	start := time.Now()

	var (
		conn    net.Conn
		err     error
		tlsInfo *TLSObservation
	)

	if spec.UseTLS {
		var tlsConn *tls.Conn
		tlsConn, err = tls.DialWithDialer(dialer, "tcp", address, &tls.Config{
			InsecureSkipVerify: m.config.TLSInsecureSkipVerify,
			ServerName:         host,
		})
		if err == nil {
			tlsInfo = extractTLSObservation(tlsConn.ConnectionState())
			conn = tlsConn
		}
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", address)
	}

	if err != nil {
		obs.Duration = time.Since(start)
		obs.Error = err.Error()
		return obs
	}
	defer func() { _ = conn.Close() }()

	if tlsInfo != nil {
		obs.TLS = tlsInfo
	}

	responses := make([]string, 0, len(spec.Commands)+1)
	if !spec.SkipInitialRead {
		initial, readErr := m.readProbeResponse(ctx, conn)
		if initial != "" {
			responses = append(responses, initial)
		}
		if readErr != nil && readErr != io.EOF && ctx.Err() == nil {
			obs.Error = readErr.Error()
		}
	}

	for _, cmd := range spec.Commands {
		if ctx.Err() != nil {
			obs.Error = ctx.Err().Error()
			break
		}
		if _, writeErr := conn.Write([]byte(cmd)); writeErr != nil {
			obs.Error = writeErr.Error()
			break
		}
		resp, rErr := m.readProbeResponse(ctx, conn)
		if resp != "" {
			responses = append(responses, resp)
		}
		if rErr != nil && rErr != io.EOF && ctx.Err() == nil {
			obs.Error = rErr.Error()
			break
		}
	}

	obs.Duration = time.Since(start)

	if len(responses) > 0 {
		obs.Response = strings.TrimSpace(strings.Join(responses, "\n"))
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		obs.Error = ctxErr.Error()
	}

	return obs
}

func (m *BannerGrabModule) grabGenericBanner(ctx context.Context, host string, port int) (string, time.Duration, error) {
	address := net.JoinHostPort(host, strconv.Itoa(port))
	dialer := &net.Dialer{Timeout: m.config.ConnectTimeout}
	start := time.Now()

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return "", time.Since(start), err
	}
	defer func() { _ = conn.Close() }()

	if err := conn.SetReadDeadline(time.Now().Add(m.config.ReadTimeout)); err != nil {
		return "", time.Since(start), err
	}

	reader := bufio.NewReader(conn)
	buffer := make([]byte, m.config.BufferSize)
	n, readErr := reader.Read(buffer)
	duration := time.Since(start)

	if ctx.Err() != nil {
		return "", duration, ctx.Err()
	}
	if readErr != nil && readErr != io.EOF {
		return "", duration, readErr
	}
	if n == 0 {
		return "", duration, nil
	}

	return string(buffer[:n]), duration, nil
}

func (m *BannerGrabModule) readProbeResponse(ctx context.Context, conn net.Conn) (string, error) {
	buffer := make([]byte, m.config.BufferSize)
	var builder strings.Builder

	for {
		if ctx.Err() != nil {
			return builder.String(), ctx.Err()
		}

		if err := conn.SetReadDeadline(time.Now().Add(m.config.ReadTimeout)); err != nil {
			return builder.String(), err
		}

		n, err := conn.Read(buffer)
		if n > 0 {
			builder.Write(buffer[:n])
			if n < len(buffer) || builder.Len() >= m.config.BufferSize {
				return builder.String(), nil
			}
			continue
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return builder.String(), nil
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				if builder.Len() == 0 {
					return "", nil
				}
				return builder.String(), nil
			}
			return builder.String(), err
		}

		if builder.Len() > 0 {
			return builder.String(), nil
		}

		return "", nil
	}
}

func prepareProbeCommands(spec fingerprint.ProbeSpec, host string, port int) []string {
	if spec.Payload == "" {
		return nil
	}

	payload := strings.ReplaceAll(spec.Payload, "{HOST}", host)
	if port > 0 {
		payload = strings.ReplaceAll(payload, "{PORT}", strconv.Itoa(port))
	}

	return []string{payload}
}

func portHintsFromCatalog(catalog *fingerprint.ProbeCatalog, port int) []string {
	if catalog == nil {
		return nil
	}

	hints := make(map[string]struct{})
	for _, group := range catalog.Groups {
		if len(group.PortHints) > 0 && !portInList(group.PortHints, port) {
			continue
		}
		if group.ID != "" {
			hints[strings.ToLower(group.ID)] = struct{}{}
		}
		for _, hint := range group.ProtocolHints {
			if hint == "" {
				continue
			}
			hints[strings.ToLower(hint)] = struct{}{}
		}
	}

	if len(hints) == 0 {
		return nil
	}

	out := make([]string, 0, len(hints))
	for hint := range hints {
		out = append(out, hint)
	}
	return out
}

func portInList(list []int, port int) bool {
	for _, v := range list {
		if v == port {
			return true
		}
	}
	return false
}

func protocolHintFromBanner(banner string) string {
	banner = strings.ToLower(banner)
	switch {
	case strings.HasPrefix(banner, "ssh-"):
		return "ssh"
	case strings.Contains(banner, "http/") || strings.Contains(banner, "server:"):
		return "http"
	case strings.Contains(banner, "smtp"):
		return "smtp"
	case strings.Contains(banner, "ftp"):
		return "ftp"
	case strings.Contains(banner, "imap"):
		return "imap"
	case strings.Contains(banner, "pop3"):
		return "pop3"
	case strings.Contains(banner, "redis"):
		return "redis"
	}
	return ""
}

type hintAccumulator struct {
	set map[string]struct{}
}

func newHintAccumulator() hintAccumulator {
	return hintAccumulator{set: make(map[string]struct{})}
}

func (h *hintAccumulator) add(hint string) {
	if hint == "" {
		return
	}
	if h.set == nil {
		h.set = make(map[string]struct{})
	}
	h.set[strings.ToLower(hint)] = struct{}{}
}

func (h *hintAccumulator) addAll(hints []string) {
	for _, hint := range hints {
		h.add(hint)
	}
}

func (h hintAccumulator) slice() []string {
	if len(h.set) == 0 {
		return nil
	}
	out := make([]string, 0, len(h.set))
	for hint := range h.set {
		out = append(out, hint)
	}
	return out
}

func extractTLSObservation(state tls.ConnectionState) *TLSObservation {
	if !state.HandshakeComplete {
		return nil
	}

	obs := &TLSObservation{
		Version:     tlsVersionString(state.Version),
		CipherSuite: tls.CipherSuiteName(state.CipherSuite),
		ServerName:  state.ServerName,
	}

	if len(state.PeerCertificates) > 0 {
		cert := state.PeerCertificates[0]
		obs.PeerCommonName = cert.Subject.CommonName
		if len(cert.DNSNames) > 0 {
			obs.PeerDNSNames = append([]string(nil), cert.DNSNames...)
		}
	}

	return obs
}

func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS13:
		return "TLS1.3"
	case tls.VersionTLS12:
		return "TLS1.2"
	case tls.VersionTLS11:
		return "TLS1.1"
	case tls.VersionTLS10:
		return "TLS1.0"
	default:
		return fmt.Sprintf("0x%x", version)
	}
}

// BannerGrabModuleFactory creates a new BannerGrabModule instance.
func BannerGrabModuleFactory() engine.Module {
	return newBannerGrabModule()
}

func init() {
	engine.RegisterModuleFactory("banner-grabber", BannerGrabModuleFactory)
}

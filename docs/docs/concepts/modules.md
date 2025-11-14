# Module System

Modules are the building blocks of Vulntor scans. Each module performs a specific function within the scan pipeline and can be composed via the DAG engine to create custom workflows.

## What is a Module?

A module is a self-contained unit that:

- **Reads inputs** from DataContext
- **Performs a specific operation** (discovery, scanning, parsing, reporting)
- **Writes outputs** to DataContext
- **Declares dependencies** (what it needs to run)
- **Provides metadata** (name, description, configuration schema)

## Module Types

### Discovery Modules

Identify live hosts on the network.

**Examples**:
- `icmp_discovery`: ICMP echo (ping) probe
- `arp_discovery`: ARP requests for local network
- `tcp_probe_discovery`: TCP SYN to common ports

**Inputs**: `targets` (parsed target list)
**Outputs**: `discovered_hosts` (list of responsive hosts)

### Scanner Modules

Probe hosts for open ports and capture banners.

**Examples**:
- `syn_scanner`: TCP SYN port scan
- `connect_scanner`: Full TCP connect scan
- `udp_scanner`: UDP port scan
- `banner_grabber`: Connect and read service banners

**Inputs**: `discovered_hosts` or `targets`
**Outputs**: `open_ports`, `banners`

### Parser Modules

Extract structured data from raw scan output.

**Examples**:
- `http_parser`: Parse HTTP responses (headers, body, status)
- `ssh_parser`: Parse SSH banners (version, algorithms)
- `smtp_parser`: Parse SMTP banners and capabilities

**Inputs**: `banners`
**Outputs**: `parsed_services`

### Fingerprint Modules

Identify services, applications, and operating systems.

**Examples**:
- `fingerprint_coordinator`: Orchestrate layered detection
- `banner_fingerprinter`: Match banners against signatures
- `http_fingerprinter`: Detect web servers and frameworks
- `tls_fingerprinter`: Identify TLS implementations

**Inputs**: `banners`, `parsed_services`
**Outputs**: `service_fingerprints`

### Profiler Modules

Build comprehensive asset profiles.

**Examples**:
- `asset_profiler`: Fuse signals into device/OS/app profile
- `os_classifier`: Determine operating system
- `device_classifier`: Identify device type (server, IoT, etc.)

**Inputs**: `service_fingerprints`, `open_ports`
**Outputs**: `asset_profiles`

### Evaluation Modules

Assess vulnerabilities and compliance.

**Examples**:
- `cve_matcher`: Match versions against CVE database
- `misconfig_checker`: Detect common misconfigurations
- `weak_cipher_checker`: Identify weak cryptography
- `compliance_evaluator`: Check CIS/PCI/NIST rules (Enterprise)

**Inputs**: `asset_profiles`, `service_fingerprints`
**Outputs**: `vulnerabilities`, `compliance_violations`

### Reporter Modules

Generate output in various formats.

**Examples**:
- `json_reporter`: JSON/JSONL output
- `csv_reporter`: CSV export
- `pdf_reporter`: Executive PDF report (Enterprise)
- `html_reporter`: Interactive HTML dashboard

**Inputs**: All DataContext keys
**Outputs**: Files written to storage or stdout

## Module Interface

### Go Module Interface

```go
package module

import "context"

// Module interface that all modules must implement
type Module interface {
    // Metadata
    Name() string
    Description() string
    Version() string

    // Configuration
    ConfigSchema() Schema
    Configure(config Config) error

    // Dependencies
    Requires() []string  // DataContext keys needed
    Provides() []string  // DataContext keys produced

    // Execution
    Execute(ctx context.Context, data DataContext) error

    // Lifecycle
    Initialize() error
    Cleanup() error
}
```

### Example Module Implementation

```go
package discovery

import (
    "context"
    "github.com/vulntor/vulntor/pkg/module"
)

type ICMPModule struct {
    timeout  time.Duration
    retry    int
    icmpConn net.PacketConn
}

func (m *ICMPModule) Name() string {
    return "icmp_discovery"
}

func (m *ICMPModule) Description() string {
    return "Discover live hosts using ICMP echo requests"
}

func (m *ICMPModule) Version() string {
    return "1.0.0"
}

func (m *ICMPModule) ConfigSchema() module.Schema {
    return module.Schema{
        "timeout": {Type: "duration", Default: "2s"},
        "retry":   {Type: "int", Default: 2},
    }
}

func (m *ICMPModule) Configure(config module.Config) error {
    m.timeout = config.GetDuration("timeout")
    m.retry = config.GetInt("retry")
    return nil
}

func (m *ICMPModule) Requires() []string {
    return []string{"targets"}
}

func (m *ICMPModule) Provides() []string {
    return []string{"discovered_hosts"}
}

func (m *ICMPModule) Initialize() error {
    // Open raw ICMP socket
    conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
    if err != nil {
        return fmt.Errorf("failed to open ICMP socket: %w", err)
    }
    m.icmpConn = conn
    return nil
}

func (m *ICMPModule) Execute(ctx context.Context, data module.DataContext) error {
    // Read targets from context
    targets, err := data.GetTargets("targets")
    if err != nil {
        return err
    }

    var discovered []Host
    for _, target := range targets {
        // Send ICMP echo
        if m.ping(target) {
            discovered = append(discovered, Host{IP: target.IP})
        }
    }

    // Write results to context
    return data.Set("discovered_hosts", discovered)
}

func (m *ICMPModule) Cleanup() error {
    if m.icmpConn != nil {
        return m.icmpConn.Close()
    }
    return nil
}

func (m *ICMPModule) ping(target Target) bool {
    // ICMP ping implementation
    // ...
    return true
}
```

### Module Registration

Modules register themselves during package initialization:

```go
package discovery

import "github.com/vulntor/vulntor/pkg/module"

func init() {
    module.Register("icmp_discovery", &ICMPModule{})
    module.Register("arp_discovery", &ARPModule{})
    module.Register("tcp_probe_discovery", &TCPProbeModule{})
}
```

## Embedded vs External Modules

### Embedded Modules (Builtin)

Compiled into Vulntor binary:

**Advantages**:
- Fast (no IPC overhead)
- No external dependencies
- Simpler deployment
- Always available

**Disadvantages**:
- Requires recompilation to update
- All modules loaded into memory
- Language limited to Go

**Usage**:
```go
import _ "github.com/vulntor/vulntor/pkg/modules/discovery"
import _ "github.com/vulntor/vulntor/pkg/modules/scanner"
import _ "github.com/vulntor/vulntor/pkg/modules/fingerprint"
```

All builtin modules auto-register via `init()`.

### External Modules (Plugins)

Isolated processes or libraries:

**Advantages**:
- Hot-reloadable without Vulntor restart
- Isolated failures (crash doesn't kill Vulntor)
- Any language (via gRPC)
- Memory efficient (loaded on demand)
- Third-party distribution

**Disadvantages**:
- IPC overhead (~10-50ms per call)
- More complex deployment
- Requires plugin management

**Types**:

#### 1. Go Plugins (`.so` shared objects)

```go
// plugin-vuln/main.go
package main

import "github.com/vulntor/vulntor/pkg/module"

type CustomVulnChecker struct{}

func (m *CustomVulnChecker) Name() string {
    return "custom_vuln_checker"
}

// ... implement Module interface ...

var Plugin = &CustomVulnChecker{}
```

Build:
```bash
go build -buildmode=plugin -o vuln-checker.so plugin-vuln/main.go
```

Load:
```bash
vulntor scan --plugin vuln-checker.so --targets 192.168.1.100
```

#### 2. gRPC Plugins (any language)

```proto
// module.proto
service ModuleService {
    rpc Execute(ExecuteRequest) returns (ExecuteResponse);
    rpc GetMetadata(Empty) returns (Metadata);
}

message ExecuteRequest {
    map<string, bytes> context = 1;
    bytes config = 2;
}

message ExecuteResponse {
    map<string, bytes> context = 1;
    string error = 2;
}
```

Python example:
```python
# custom_module.py
import grpc
from vulntor_pb2 import ExecuteRequest, ExecuteResponse
from vulntor_pb2_grpc import ModuleServiceServicer

class CustomModule(ModuleServiceServicer):
    def Execute(self, request, context):
        # Read inputs
        targets = request.context.get('targets')

        # Custom logic
        results = self.scan(targets)

        # Write outputs
        return ExecuteResponse(
            context={'custom_results': results}
        )
```

Register:
```yaml
plugins:
  - name: custom_module
    type: grpc
    endpoint: localhost:50051
    timeout: 30s
```

#### 3. WASM Plugins (experimental)

WebAssembly modules for sandboxed execution:

```rust
// custom_scanner.rs
use vulntor_sdk::*;

#[no_mangle]
pub extern "C" fn execute(context: *const Context) -> i32 {
    let targets = context.get("targets");
    let results = scan(targets);
    context.set("results", results);
    0
}
```

Compile to WASM:
```bash
cargo build --target wasm32-wasi --release
```

Load:
```bash
vulntor scan --plugin custom_scanner.wasm --targets 192.168.1.100
```

## Module Lifecycle

```
┌──────────────┐
│ Registration │  (init() or plugin load)
└──────┬───────┘
       │
┌──────▼───────┐
│ Initialize   │  (one-time setup: open sockets, load data)
└──────┬───────┘
       │
┌──────▼───────┐
│ Configure    │  (apply runtime config)
└──────┬───────┘
       │
       ├─────────────┐
       │             │
┌──────▼───────┐    │
│ Execute      │◄───┘ (called per scan, potentially many times)
└──────┬───────┘
       │
┌──────▼───────┐
│ Cleanup      │  (release resources)
└──────────────┘
```

### Registration Phase

Module announces itself to registry:

```go
module.Register("my_module", &MyModule{})
```

### Initialize Phase

One-time setup before any scans:

```go
func (m *MyModule) Initialize() error {
    // Open persistent connections
    m.db = openDatabase()

    // Load data files
    m.signatures = loadSignatures()

    return nil
}
```

Called once when Vulntor starts or plugin loads.

### Configure Phase

Apply scan-specific configuration:

```go
func (m *MyModule) Configure(config module.Config) error {
    m.timeout = config.GetDuration("timeout")
    m.concurrency = config.GetInt("concurrency")
    return nil
}
```

Called before each scan with DAG node config.

### Execute Phase

Perform module operation:

```go
func (m *MyModule) Execute(ctx context.Context, data module.DataContext) error {
    // Read inputs
    targets, _ := data.Get("targets")

    // Perform work
    results := m.scan(targets)

    // Write outputs
    data.Set("results", results)

    return nil
}
```

Called once per scan (or multiple times for parallel instances).

### Cleanup Phase

Release resources:

```go
func (m *MyModule) Cleanup() error {
    if m.db != nil {
        m.db.Close()
    }
    return nil
}
```

Called when Vulntor exits or plugin unloads.

## Module Configuration

### Schema Definition

Modules declare configuration schema:

```go
func (m *ScannerModule) ConfigSchema() module.Schema {
    return module.Schema{
        "ports": {
            Type:        "string",
            Description: "Port list (e.g., '80,443' or '1-1000')",
            Default:     "1-1000",
            Required:    false,
        },
        "rate": {
            Type:        "int",
            Description: "Packets per second",
            Default:     1000,
            Min:         1,
            Max:         100000,
        },
        "timeout": {
            Type:        "duration",
            Description: "Connection timeout",
            Default:     "3s",
        },
        "protocol": {
            Type:        "string",
            Description: "Protocol to scan",
            Enum:        []string{"tcp", "udp"},
            Default:     "tcp",
        },
    }
}
```

### Runtime Configuration

Provided in DAG node definition:

```yaml
nodes:
  - instance_id: port_scan
    module_type: syn_scanner
    config:
      ports: "1-10000"
      rate: 5000
      timeout: 5s
      protocol: tcp
```

### Configuration Validation

Vulntor validates config against schema before execution:

```bash
vulntor dag validate my-scan.yaml
```

Checks:
- Required fields present
- Types correct (int, string, duration, bool)
- Values within allowed ranges
- Enum values valid

## Module Communication

### DataContext Keys

Modules communicate via shared keys:

```go
// Producer
data.Set("open_ports", []Port{
    {Host: "192.168.1.100", Port: 22},
    {Host: "192.168.1.100", Port: 80},
})

// Consumer
ports, err := data.Get("open_ports")
if err != nil {
    return fmt.Errorf("missing required input: %w", err)
}

for _, port := range ports.([]Port) {
    // Process port
}
```

### Type Safety

Use typed getters for safety:

```go
// module/context.go
type DataContext interface {
    Get(key string) (interface{}, error)

    // Typed accessors
    GetTargets(key string) ([]Target, error)
    GetHosts(key string) ([]Host, error)
    GetPorts(key string) ([]Port, error)
    GetBanners(key string) ([]Banner, error)
    GetFingerprints(key string) ([]Fingerprint, error)
}
```

### Namespace Conventions

Avoid key collisions:

```
<module_type>.<instance_id>.<output>
```

Example:
```
icmp_discovery.main.discovered_hosts
syn_scanner.port_scan_1.open_ports
banner_grabber.banner_1.banners
```

For standard keys, use simple names:
```
targets
discovered_hosts
open_ports
banners
service_fingerprints
```

## Error Handling

### Return Errors

Modules should return descriptive errors:

```go
func (m *Module) Execute(ctx context.Context, data DataContext) error {
    targets, err := data.GetTargets("targets")
    if err != nil {
        return fmt.Errorf("failed to read targets: %w", err)
    }

    results, err := m.scan(targets)
    if err != nil {
        return fmt.Errorf("scan failed: %w", err)
    }

    if err := data.Set("results", results); err != nil {
        return fmt.Errorf("failed to write results: %w", err)
    }

    return nil
}
```

### Partial Results

Write partial results before returning error:

```go
func (m *Module) Execute(ctx context.Context, data DataContext) error {
    var results []Result

    for _, target := range targets {
        result, err := m.scan(target)
        if err != nil {
            // Log error but continue
            log.Warn().Err(err).Str("target", target).Msg("scan failed")
            continue
        }
        results = append(results, result)
    }

    // Write partial results
    data.Set("results", results)

    if len(results) == 0 {
        return errors.New("all scans failed")
    }

    return nil
}
```

### Context Cancellation

Respect context cancellation:

```go
func (m *Module) Execute(ctx context.Context, data DataContext) error {
    for _, target := range targets {
        select {
        case <-ctx.Done():
            return ctx.Err()  // Cancelled or timeout
        default:
            result := m.scan(target)
            results = append(results, result)
        }
    }
    return nil
}
```

## Module Distribution

### Builtin Modules

Shipped with Vulntor:

- Discovery: ICMP, ARP, TCP probe
- Scanner: SYN, Connect, UDP, banner grab
- Parser: HTTP, SSH, SMTP, FTP, TLS
- Fingerprint: Banner matching, HTTP detection
- Profiler: Asset classification
- Reporter: JSON, CSV, JSONL

Always available, no installation required.

### Official Plugin Repository

Vulntor-maintained plugins:

```bash
# List available plugins
vulntor plugin list

# Install plugin
vulntor plugin install vuln/nmap-nse-wrapper

# Update plugin
vulntor plugin update vuln/nmap-nse-wrapper

# Remove plugin
vulntor plugin remove vuln/nmap-nse-wrapper
```

Plugins installed to `~/.local/share/vulntor/plugins/`.

### Third-Party Plugins

Community-developed modules:

```bash
# Install from URL
vulntor plugin install https://github.com/user/custom-scanner/releases/latest/plugin.so

# Install from file
vulntor plugin install /path/to/plugin.so
```

**Security**: Signature verification required (Enterprise):

```yaml
plugins:
  require_signature: true
  trusted_publishers:
    - fingerprint: A1B2C3D4E5F6...
      name: TrustedVendor
```

### Enterprise Plugin Marketplace

Browse and install plugins via UI (Enterprise):

1. Navigate to **Plugins** → **Marketplace**
2. Search/filter by category
3. Click **Install**
4. Configure plugin settings
5. Enable in scan profiles

Licensing enforced per plugin.

## Best Practices

### 1. Minimize State

Keep modules stateless where possible:

```go
// Bad: Shared state
type Module struct {
    results []Result  // Shared across scans
}

// Good: State in DataContext
func (m *Module) Execute(ctx context.Context, data DataContext) error {
    var results []Result
    // ...
    data.Set("results", results)
    return nil
}
```

### 2. Validate Inputs

Check DataContext inputs:

```go
func (m *Module) Execute(ctx context.Context, data DataContext) error {
    targets, err := data.GetTargets("targets")
    if err != nil {
        return fmt.Errorf("missing targets: %w", err)
    }

    if len(targets) == 0 {
        return errors.New("no targets provided")
    }

    // ... proceed with scan
}
```

### 3. Structured Logging

Use Zerolog with context:

```go
import "github.com/rs/zerolog/log"

func (m *Module) Execute(ctx context.Context, data DataContext) error {
    logger := log.With().
        Str("module", m.Name()).
        Str("instance", m.instanceID).
        Logger()

    logger.Info().Msg("execution started")

    // ... perform work

    logger.Info().
        Int("results", len(results)).
        Dur("duration", elapsed).
        Msg("execution completed")

    return nil
}
```

### 4. Respect Timeouts

Honor context deadlines:

```go
func (m *Module) scan(ctx context.Context, target Target) (Result, error) {
    deadline, ok := ctx.Deadline()
    if ok {
        timeout := time.Until(deadline)
        conn.SetDeadline(time.Now().Add(timeout))
    }

    // ... perform scan
}
```

### 5. Handle Concurrency

If module spawns goroutines:

```go
func (m *Module) Execute(ctx context.Context, data DataContext) error {
    var wg sync.WaitGroup
    resultsChan := make(chan Result)

    for _, target := range targets {
        wg.Add(1)
        go func(t Target) {
            defer wg.Done()
            result := m.scan(ctx, t)
            resultsChan <- result
        }(target)
    }

    go func() {
        wg.Wait()
        close(resultsChan)
    }()

    var results []Result
    for result := range resultsChan {
        results = append(results, result)
    }

    data.Set("results", results)
    return nil
}
```

## Testing Modules

### Unit Tests

Test module in isolation:

```go
func TestICMPModule_Execute(t *testing.T) {
    // Setup
    module := &ICMPModule{}
    module.Configure(module.Config{
        "timeout": "1s",
        "retry": 1,
    })
    module.Initialize()
    defer module.Cleanup()

    // Create test context
    ctx := context.Background()
    data := module.NewTestDataContext()
    data.Set("targets", []Target{
        {IP: "127.0.0.1"},
    })

    // Execute
    err := module.Execute(ctx, data)
    require.NoError(t, err)

    // Verify
    hosts, err := data.GetHosts("discovered_hosts")
    require.NoError(t, err)
    assert.Len(t, hosts, 1)
    assert.Equal(t, "127.0.0.1", hosts[0].IP)
}
```

### Integration Tests

Test module in DAG:

```go
func TestScanPipeline(t *testing.T) {
    dag := `
nodes:
  - instance_id: targets
    module_type: target_parser
  - instance_id: discover
    module_type: icmp_discovery
    depends_on: [targets]
  - instance_id: scan
    module_type: syn_scanner
    depends_on: [discover]
`

    orchestrator := engine.NewOrchestrator()
    orchestrator.LoadDAG([]byte(dag))

    result, err := orchestrator.Execute(context.Background())
    require.NoError(t, err)
    assert.Equal(t, "completed", result.Status)
}
```

## Next Steps

- [DAG Engine](/concepts/dag-engine) - How modules are orchestrated
- [Custom Modules](/advanced/custom-modules) - Writing your own modules
- [External Plugins](/advanced/external-plugins) - gRPC and WASM plugins
- [Module API Reference](/api/modules/interface) - Full API documentation

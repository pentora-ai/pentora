# DAG Execution Engine

Pentora's DAG (Directed Acyclic Graph) engine orchestrates scan execution by managing module dependencies, enabling parallel execution, and coordinating data flow between stages.

## What is a DAG?

A Directed Acyclic Graph is a mathematical structure where:

- **Nodes** represent modules (discovery, port scan, fingerprint, etc.)
- **Edges** represent dependencies (execution order and data flow)
- **Directed** means edges have a direction (from producer to consumer)
- **Acyclic** means no cycles (no circular dependencies)

## Why Use a DAG?

### 1. Explicit Dependencies

Modules declare what they need:

```
Banner Grab Module:
  Requires: open_ports (from Port Scanner)
  Produces: banners

Fingerprint Module:
  Requires: banners (from Banner Grab)
  Produces: service_fingerprints
```

### 2. Parallel Execution

Independent modules run concurrently:

```
        Discovery
            ↓
      Port Scanner
       ↙    ↓    ↘
   Banner  Banner  Banner
   (Port 22) (80)  (443)
       ↘    ↓    ↙
     Fingerprinter
            ↓
    Asset Profiler
```

Banners can be grabbed from multiple ports simultaneously.

### 3. Failure Isolation

If one branch fails, others continue:

```
Port Scanner
  ↙        ↘
Banner 22   Banner 80
  ↓           ↓
Failed!    Fingerprint 80
              ↓
           Success!
```

### 4. Deterministic Execution

DAG ensures consistent, reproducible results regardless of hardware or timing.

## DAG Structure

### Node Definition

Each node represents a module instance:

```yaml
nodes:
  - instance_id: target_ingestion
    module_type: target_parser
    depends_on: []
    config:
      blocklists: [127.0.0.0/8]

  - instance_id: discovery
    module_type: icmp_discovery
    depends_on: [target_ingestion]
    config:
      timeout: 2s
      retry: 2

  - instance_id: port_scan
    module_type: syn_scanner
    depends_on: [discovery]
    config:
      ports: 1-1000
      rate: 1000

  - instance_id: banner_grab
    module_type: banner_grabber
    depends_on: [port_scan]
    config:
      timeout: 5s

  - instance_id: fingerprint
    module_type: fingerprint_parser
    depends_on: [banner_grab]
    config:
      catalog: builtin

  - instance_id: reporter
    module_type: json_reporter
    depends_on: [fingerprint]
    config:
      output: results.jsonl
```

### Dependency Declaration

Modules specify dependencies via `depends_on`:

- **No dependencies**: `depends_on: []` (root nodes)
- **Single dependency**: `depends_on: [discovery]`
- **Multiple dependencies**: `depends_on: [port_scan, banner_grab]`

### Validation

DAG validator checks:

1. **No cycles**: Dependency chains cannot loop
2. **All dependencies exist**: Referenced nodes must be defined
3. **Unique IDs**: No duplicate `instance_id` values
4. **Valid module types**: Module must be registered
5. **Configuration validity**: Module config must match schema

```bash
# Validate DAG definition
pentora dag validate scan-profile.yaml
```

## Execution Model

### Phases

DAG execution proceeds in phases:

#### Phase 1: Planning

1. **Parse DAG definition** (YAML/JSON)
2. **Validate structure** (cycles, dependencies, config)
3. **Resolve modules** (lookup registered implementations)
4. **Build execution plan** (topological sort)

#### Phase 2: Execution

1. **Initialize DataContext** (shared key-value store)
2. **Execute nodes in dependency order**:
   - Wait for all dependencies to complete
   - Read required inputs from DataContext
   - Execute module logic
   - Write outputs to DataContext
   - Mark node as completed
3. **Handle failures** (skip dependents or retry)
4. **Cleanup resources**

### Topological Sort

Determines execution order respecting dependencies:

```
Input DAG:
  A → B → D
  A → C → D

Topological Sort:
  [A, B, C, D]  or  [A, C, B, D]
  (Both valid; B and C can run in parallel)
```

### Parallelism

Nodes without dependencies between them run concurrently:

```go
// Pseudocode execution
plan := TopologicalSort(dag)

for _, layer := range GroupByLevel(plan) {
    var wg sync.WaitGroup

    for _, node := range layer {
        wg.Add(1)
        go func(n Node) {
            defer wg.Done()
            ExecuteNode(n, context)
        }(node)
    }

    wg.Wait()  // Wait for entire layer before next
}
```

**Example layer grouping**:

```
Layer 0: [Target Ingestion]
Layer 1: [Discovery]
Layer 2: [Port Scanner]
Layer 3: [Banner Grab Port 22, Banner Grab Port 80, Banner Grab Port 443]
Layer 4: [Fingerprint Parser]
Layer 5: [Asset Profiler]
Layer 6: [Vulnerability Evaluator]
Layer 7: [Reporter]
```

Nodes in Layer 3 execute in parallel (3x speedup).

## Data Flow

### DataContext

Shared state container passed through DAG:

```go
type DataContext interface {
    // Read value by key
    Get(key string) (interface{}, error)

    // Write value
    Set(key string, value interface{}) error

    // Check key existence
    Has(key string) bool

    // List all keys
    Keys() []string
}
```

### Standard Keys

Modules use conventional keys for interoperability:

| Key                  | Producer         | Consumer           | Type                 |
|----------------------|------------------|--------------------|--------------------|
| `targets`            | Target Ingestion | Discovery          | `[]Target`         |
| `discovered_hosts`   | Discovery        | Port Scanner       | `[]Host`           |
| `open_ports`         | Port Scanner     | Banner Grab        | `[]Port`           |
| `banners`            | Banner Grab      | Fingerprint Parser | `[]Banner`         |
| `service_fingerprints` | Fingerprint    | Asset Profiler     | `[]Fingerprint`    |
| `asset_profiles`     | Asset Profiler   | Vuln Evaluator     | `[]AssetProfile`   |
| `vulnerabilities`    | Vuln Evaluator   | Reporter           | `[]Vulnerability`  |

### Example Flow

```go
// Discovery module writes hosts
context.Set("discovered_hosts", []Host{
    {IP: "192.168.1.100", Latency: 1.2},
    {IP: "192.168.1.101", Latency: 2.5},
})

// Port scanner reads hosts
hosts, _ := context.Get("discovered_hosts")
for _, host := range hosts.([]Host) {
    ports := ScanPorts(host)
    // ...
}

// Port scanner writes ports
context.Set("open_ports", []Port{
    {Host: "192.168.1.100", Port: 22, Protocol: "tcp"},
    {Host: "192.168.1.100", Port: 80, Protocol: "tcp"},
})
```

### Data Isolation

Each scan gets its own DataContext instance:

- No interference between concurrent scans
- Memory garbage collected after scan completion
- Thread-safe access (mutex-protected)

## Failure Handling

### Failure Modes

1. **Node failure**: Module execution error (timeout, crash, assertion)
2. **Dependency failure**: Required input missing from DataContext
3. **Configuration error**: Invalid module config

### Strategies

#### Fail-Fast (Default)

Stop entire DAG on first error:

```yaml
engine:
  fail_fast: true
```

```
Discovery → Port Scan → FAILED!
                        ↓
              [Execution Stopped]
```

**Use case**: Critical failures where partial results are useless.

#### Continue-on-Error

Skip failed node and dependents, continue other branches:

```yaml
engine:
  fail_fast: false
```

```
            Port Scan → Banner Grab (Port 80) → Fingerprint
                ↓
         Banner Grab (Port 22) → FAILED!
                                   ↓
                             [Skipped dependents]
```

**Use case**: Large scans where partial results are valuable.

#### Retry Logic

Retry transient failures with backoff:

```yaml
engine:
  retry:
    enabled: true
    max_attempts: 3
    backoff: exponential  # 1s, 2s, 4s
    retry_on:
      - timeout
      - network_error
```

```
Attempt 1: FAILED (timeout)
  ↓ wait 1s
Attempt 2: FAILED (timeout)
  ↓ wait 2s
Attempt 3: SUCCESS
```

**Use case**: Network instability, rate limiting, transient errors.

### Dependent Skipping

When a node fails, dependents are automatically skipped:

```
Node A → FAILED
  ↓
Node B (depends on A) → SKIPPED
  ↓
Node C (depends on B) → SKIPPED
```

Reporter always runs to capture partial results and errors.

### Error Context

Failed nodes record detailed error information:

```json
{
  "node": "port_scan",
  "status": "failed",
  "error": "timeout after 30s",
  "stack_trace": "...",
  "timestamp": "2023-10-06T14:35:22Z",
  "retry_attempts": 3
}
```

Available in scan status and logs.

## Orchestrator Architecture

### Components

```
┌──────────────────────────────────────┐
│          Orchestrator                │
├──────────────────────────────────────┤
│  - DAG Parser                        │
│  - Validator                         │
│  - Planner (Topological Sort)       │
│  - Executor (Node Runner)            │
│  - DataContext Manager               │
│  - Error Handler                     │
└──────────────────────────────────────┘
         ↓                    ↓
┌────────────────┐   ┌────────────────┐
│ Module Registry│   │   Event Bus    │
└────────────────┘   └────────────────┘
```

### Orchestrator Interface

```go
type Orchestrator interface {
    // Load DAG from definition
    LoadDAG(definition []byte) error

    // Validate DAG structure
    Validate() error

    // Execute DAG with context
    Execute(ctx context.Context) (*Result, error)

    // Get execution status
    Status() *Status

    // Cancel execution
    Cancel() error
}
```

### Module Registry

Maintains mapping of module types to implementations:

```go
registry := module.NewRegistry()

// Register builtin modules
registry.Register("icmp_discovery", &discovery.ICMPModule{})
registry.Register("syn_scanner", &scanner.SYNModule{})
registry.Register("banner_grabber", &scanner.BannerModule{})
registry.Register("fingerprint_parser", &fingerprint.ParserModule{})

// Register external plugin
registry.RegisterPlugin("custom_vuln_check", "/path/to/plugin.so")
```

See [Module System](/concepts/modules) for registration details.

## Configuration

### DAG Definition

Define custom scan flows:

```yaml
# custom-scan.yaml
name: web-application-scan
description: Focused scan for web applications

nodes:
  - instance_id: targets
    module_type: target_parser
    depends_on: []

  - instance_id: discover
    module_type: tcp_probe_discovery
    depends_on: [targets]
    config:
      ports: [80, 443, 8080, 8443]

  - instance_id: http_scan
    module_type: http_scanner
    depends_on: [discover]
    config:
      methods: [GET, HEAD, OPTIONS]
      headers:
        User-Agent: Pentora/1.0

  - instance_id: ssl_scan
    module_type: ssl_analyzer
    depends_on: [discover]
    config:
      check_cert: true
      check_ciphers: true

  - instance_id: webapp_fingerprint
    module_type: webapp_fingerprinter
    depends_on: [http_scan]

  - instance_id: report
    module_type: json_reporter
    depends_on: [webapp_fingerprint, ssl_scan]
```

### Using Custom DAGs

```bash
# Execute custom DAG
pentora scan --targets example.com --dag custom-scan.yaml

# Validate DAG before execution
pentora dag validate custom-scan.yaml
```

### Built-in DAGs

Pentora includes predefined DAGs for common scenarios:

- `standard.yaml`: Full 9-stage pipeline
- `discovery-only.yaml`: Target ingestion + discovery + reporting
- `port-scan.yaml`: Discovery + port scan + reporting
- `vuln-scan.yaml`: Full pipeline with vulnerability evaluation

Accessed via scan profiles:

```bash
pentora scan --targets 192.168.1.0/24 --profile standard
# Uses builtin/standard.yaml DAG
```

## Observability

### Logging

Each node logs with structured context:

```json
{
  "level": "info",
  "timestamp": "2023-10-06T14:30:45Z",
  "component": "orchestrator",
  "node": "port_scan",
  "message": "Node execution started"
}
```

```json
{
  "level": "info",
  "timestamp": "2023-10-06T14:31:15Z",
  "component": "orchestrator",
  "node": "port_scan",
  "duration_ms": 30200,
  "message": "Node execution completed"
}
```

### Event Hooks

Subscribe to execution events:

```go
orchestrator.OnNodeStart(func(node Node) {
    fmt.Printf("Starting %s\n", node.ID)
})

orchestrator.OnNodeComplete(func(node Node, result Result) {
    fmt.Printf("Completed %s in %v\n", node.ID, result.Duration)
})

orchestrator.OnNodeFailed(func(node Node, err error) {
    fmt.Printf("Failed %s: %v\n", node.ID, err)
})
```

See [Hook System](/advanced/hooks-events) for event details.

### Progress Tracking

Track execution progress:

```bash
pentora scan --targets 192.168.1.0/24 --progress
```

```
[=====>                    ] 25% (2/8 nodes completed)
Currently running: port_scan, banner_grab
```

Progress events available via API for UI integration.

## Performance Tuning

### Concurrency Limits

Control parallelism:

```yaml
engine:
  max_parallel_nodes: 10      # Max concurrent nodes
  max_parallel_targets: 100   # Max concurrent targets per node
```

**Low concurrency**: Lower memory, slower execution
**High concurrency**: Higher memory, faster execution

### Memory Management

DataContext can grow large with many targets:

```yaml
engine:
  data_context:
    max_size: 1GB             # Limit context size
    evict_policy: lru         # Least-recently-used eviction
```

### Execution Timeout

Prevent hung scans:

```yaml
engine:
  global_timeout: 1h          # Abort after 1 hour
  node_timeout: 10m           # Abort individual node after 10 min
```

## Advanced Topics

### Dynamic DAG Construction

Build DAGs programmatically:

```go
builder := dag.NewBuilder()

builder.AddNode("targets", "target_parser", nil)
builder.AddNode("discover", "icmp_discovery", []string{"targets"})

// Conditionally add vulnerability checks
if enableVuln {
    builder.AddNode("vuln", "cve_matcher", []string{"discover"})
}

dag := builder.Build()
orchestrator.LoadDAG(dag)
```

### Sub-DAGs

Compose reusable DAG fragments:

```yaml
# discovery-subdag.yaml
nodes:
  - instance_id: icmp
    module_type: icmp_discovery
  - instance_id: arp
    module_type: arp_discovery
  - instance_id: merge
    module_type: discovery_merger
    depends_on: [icmp, arp]
```

```yaml
# main-dag.yaml
nodes:
  - instance_id: targets
    module_type: target_parser

  - instance_id: discover
    module_type: subdag
    subdag_file: discovery-subdag.yaml
    depends_on: [targets]

  - instance_id: scan
    module_type: port_scanner
    depends_on: [discover]
```

### Conditional Execution

Skip nodes based on runtime conditions:

```yaml
nodes:
  - instance_id: vuln_check
    module_type: cve_matcher
    depends_on: [fingerprint]
    condition: ${vuln_enabled}  # Variable from context
```

## Troubleshooting

### Cycle Detection

```
Error: Cycle detected in DAG: A → B → C → A
```

**Solution**: Remove circular dependency. Restructure modules or split into multiple DAG runs.

### Missing Dependency

```
Error: Node 'banner_grab' depends on 'port_scan' which does not exist
```

**Solution**: Ensure all dependencies are defined in DAG.

### Data Not Available

```
Error: Node 'fingerprint' requires 'banners' but key not found in DataContext
```

**Solution**: Check that producer module is running and writing expected key. Enable debug logging to trace data flow.

### Deadlock

If execution hangs:

```bash
# Enable detailed logging
pentora scan --targets 192.168.1.100 --log-level debug

# Check for circular dependencies
pentora dag validate my-dag.yaml
```

## Next Steps

- [Module System](/concepts/modules) - Writing custom modules
- [Data Flow](/architecture/data-flow) - DataContext internals
- [Hook System](/advanced/hooks-events) - Event-driven customization
- [Engine Architecture](/architecture/engine) - Implementation details

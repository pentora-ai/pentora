# Core Concepts Overview

Pentora is a modular, high-performance security scanner that rapidly discovers network services, captures banners, and maps findings into vulnerability intelligence. This page introduces the fundamental concepts that power Pentora.

## What is Pentora?

Pentora provides a structured approach to security scanning through:

- **Modular architecture**: Composable scan modules organized in a directed acyclic graph (DAG)
- **Layered fingerprinting**: Multi-stage protocol detection with confidence scoring
- **Persistent storage**: Shared scan storage with retention policies
- **Flexible execution**: CLI for ad-hoc scans, server mode for scheduled operations
- **Enterprise features**: Distributed scanning, multi-tenancy, and advanced integrations

## Key Components

### 1. Scan Pipeline

Every scan flows through a structured 9-stage pipeline:

1. **Target Ingestion** - Parse and validate input targets
2. **Asset Discovery** - Identify live hosts via ICMP/ARP/TCP
3. **Port Scanning** - Probe TCP/UDP ports with rate limiting
4. **Service Fingerprinting** - Multi-layer protocol detection
5. **Asset Profiling** - Fuse signals into device/OS/application profiles
6. **Vulnerability Evaluation** - CVE matching and misconfiguration checks
7. **Compliance & Risk Scoring** - CIS/PCI/NIST rule evaluation
8. **Reporting & Notification** - Export results and trigger integrations
9. **Archival & Analytics** - Store results and compute trends

See [Scan Pipeline](/concepts/scan-pipeline) for detailed explanation.

### 2. Storage

The storage directory is a shared directory structure that stores:

- **Scan results**: Structured JSON output per scan
- **Request metadata**: Original scan parameters
- **Status tracking**: Execution state and timing
- **Artifacts**: Banner captures, raw probe data
- **Queue**: Scheduled scans (server mode)
- **Cache**: Fingerprint databases, temporary data

Default locations:
- **Linux**: `~/.local/share/pentora` (follows XDG Base Directory spec)
- **macOS**: `~/Library/Application Support/Pentora`
- **Windows**: `%AppData%\Pentora`

See [Storage Concept](/concepts/storage) for structure details.

### 3. DAG Engine

Pentora's execution engine uses a directed acyclic graph (DAG) to:

- Define module dependencies
- Enable parallel execution where possible
- Manage data flow between modules
- Handle failures gracefully

Example DAG flow:
```
Target Ingestion
      ↓
Asset Discovery
      ↓
Port Scanner → Banner Grab → Service Parser
                    ↓             ↓
              Fingerprinter → Asset Profiler
                                  ↓
                         Vulnerability Evaluator
                                  ↓
                               Reporter
```

See [DAG Engine](/concepts/dag-engine) for execution model.

### 4. Module System

Modules are the building blocks of scans:

- **Discovery modules**: Host detection (ICMP, ARP, TCP SYN)
- **Scanner modules**: Port scanning, banner grabbing
- **Parser modules**: Protocol parsing, data extraction
- **Fingerprint modules**: Service identification
- **Profiler modules**: Asset classification
- **Evaluation modules**: Vulnerability and compliance checks
- **Reporter modules**: Output generation

Modules can be:
- **Embedded**: Compiled into Pentora binary (Go code)
- **External**: Isolated plugins (gRPC/WASM) with signature verification

See [Module System](/concepts/modules) for details.

### 5. Fingerprinting System

Pentora uses a layered approach to service identification:

1. **Initial heuristics**: Port number, initial banner
2. **Protocol-specific probes**: HTTP requests, TLS handshakes, IMAP CAPABILITY
3. **Confidence scoring**: Aggregate evidence from multiple sources
4. **Multiple matches**: Surface all detected technologies (web server + framework + language)

Example fingerprint result:
```json
{
  "fingerprints": [
    {
      "match": "nginx",
      "version": "1.18.0",
      "confidence": 95,
      "source": "http_header"
    },
    {
      "match": "php",
      "version": "7.4",
      "confidence": 85,
      "source": "x_powered_by"
    }
  ]
}
```

See [Fingerprinting System](/concepts/fingerprinting) for probe details.

## Execution Modes

### CLI Mode (Ad-hoc Scans)

Direct execution for immediate results:

```bash
pentora scan --targets 192.168.1.0/24
```

- No daemon required
- Results written to storage
- Progress displayed in terminal
- Suitable for manual operations

### Server Mode (Scheduled Operations)

Long-running daemon for automated scanning:

```bash
pentora server start
```

- REST API for scan submission
- Job queue and scheduler
- Worker pools (Enterprise)
- Web UI for visualization

See [Server Mode Deployment](/deployment/server-mode) for setup.

## Data Flow

### DataContext

The `DataContext` is a shared key-value store that flows through the DAG:

```
Discovery Module
  → Sets: discovered_hosts = [...]

Port Scanner
  → Reads: discovered_hosts
  → Sets: open_ports = [...]

Banner Grabber
  → Reads: open_ports
  → Sets: banners = [...]

Service Parser
  → Reads: banners
  → Sets: parsed_services = [...]
```

Each module:
1. Reads required inputs from context
2. Performs its operation
3. Writes outputs to context
4. Passes context to dependent modules

See [Data Flow](/architecture/data-flow) for implementation.

## Configuration Model

Pentora uses a hierarchical configuration system:

1. **Default values**: Compiled-in defaults
2. **System config**: `/etc/pentora/config.yaml` (Linux) or OS equivalent
3. **User config**: `~/.config/pentora/config.yaml`
4. **Storage config**: `<storage>/config/pentora.yaml`
5. **CLI flags**: Override all file-based settings

Configuration sections:
- `storage.*`: Storage directory and retention policies
- `scanner.*`: Scan timing and concurrency
- `fingerprint.*`: Detection rules and probes
- `logging.*`: Output verbosity and format
- `server.*`: API and worker settings
- `enterprise.*`: Licensing and advanced features (Enterprise edition)

See [Configuration Overview](/configuration/overview) for structure.

## Licensing Model

### Open Source (Starter)

Core scanning capabilities:
- Full scan pipeline (all 9 stages)
- Embedded modules
- CLI and basic server mode
- Local storage
- JSON/CSV export

### Enterprise Tiers

Enhanced capabilities via JWT-based licensing:

- **Team** ($399/month): Scheduling, web UI, webhooks, Slack
- **Business** ($1,499/month): Distributed scanning, SIEM integrations, SSO
- **Enterprise** ($80k-$120k/year): Multi-tenancy, compliance packs, air-gapped

License verification:
- Signed JWT containing plan, features, expiry
- Offline grace period (7 days)
- Feature gating via `feature.Check("distributed")`

See [Enterprise Overview](/enterprise/overview) for feature matrix.

## Philosophy

### CLI vs UI

- **CLI**: Targets technical operators for ad-hoc scans, storage inspection, and troubleshooting
- **UI**: Empowers non-technical stakeholders with simplified workflows and dashboards
- **Separation**: CLI communicates via REST API; never accesses server internals directly

### Design Principles

1. **Incremental delivery**: Each feature includes tests and documentation
2. **Backward compatibility**: Config migrations and deprecation warnings
3. **Observability**: Structured logging (Zerolog), event hooks, metrics
4. **Security**: Least privilege, signature verification, audit logs
5. **Extensibility**: Plugin system, customizable DAGs, external modules

## Common Use Cases

### Network Discovery

Identify all live hosts and services on a network segment:

```bash
pentora scan --targets 10.0.0.0/16 --only-discover
pentora scan --targets @discovered --profile standard
```

### Vulnerability Assessment

Find CVEs and misconfigurations:

```bash
pentora scan --targets critical-servers.txt --vuln --profile deep
```

### Compliance Auditing

Check against regulatory frameworks (Enterprise):

```bash
pentora scan --targets dmz.txt --compliance cis-level1
```

### Continuous Monitoring

Scheduled recurring scans with alerting (Server mode):

```
POST /api/scans
{
  "targets": ["192.168.1.0/24"],
  "profile": "standard",
  "schedule": "0 2 * * *",
  "notifications": ["slack://security-alerts"]
}
```

## Next Steps

Dive deeper into specific concepts:

- [Scan Pipeline](/concepts/scan-pipeline) - All 9 stages explained
- [Storage](/concepts/storage) - Directory structure and retention
- [DAG Engine](/concepts/dag-engine) - Execution orchestration
- [Modules](/concepts/modules) - Module types and lifecycle
- [Fingerprinting](/concepts/fingerprinting) - Service detection internals

Or explore practical guides:

- [Your First Scan](/getting-started/first-scan) - Hands-on tutorial
- [Network Scanning](/guides/network-scanning) - Best practices
- [Vulnerability Assessment](/guides/vulnerability-assessment) - Security testing

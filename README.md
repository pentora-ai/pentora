# Pentora

[![Go Report Card](https://goreportcard.com/badge/github.com/pentora-ai/pentora)](https://goreportcard.com/report/github.com/pentora-ai/pentora)
[![License: Apache-2.0](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](https://opensource.org/license/apache-2-0)
[![Build](https://img.shields.io/github/actions/workflow/status/pentora-ai/pentora/test.yml?branch=main)](https://github.com/pentora-ai/pentora/actions)

> ⚠️ **Active Development Notice**: Pentora is currently under active development and has not been released yet. APIs, CLI commands, and core features are subject to change. This project is not production-ready.

**Pentora** is a modular, high-performance security scanner that rapidly discovers network services, captures banners, and maps findings into vulnerability intelligence. Built with a powerful DAG-based execution engine, Pentora enables security teams to perform comprehensive network assessments with precision and efficiency.

## ✨ Key Features

### 🚀 Performance & Scalability

- **Lightning-fast scanning** with intelligent rate limiting and concurrent execution
- **DAG-based execution engine** for parallel task processing and optimized workflows
- **Adaptive host discovery** using ICMP, ARP, and TCP probes
- **Efficient resource management** with configurable concurrency controls

### 🔍 Advanced Capabilities

- **Modular service detection** with extensible protocol parsers (SSH, HTTP, FTP, DNS, and more)
- **Fingerprint-based asset profiling** for accurate service version identification
- **Plugin-based vulnerability matching** with CVE correlation
- **Workspace management** for organizing and querying scan results
- **Hook system** for custom automation and event-driven workflows

### 🛠️ Deployment Options

- **Standalone CLI** for quick ad-hoc scans and automation
- **Server mode** with REST/gRPC API for centralized scanning
- **Web UI** for interactive scan management and visualization (In Development)
- **Cross-platform support** (macOS, Linux, Windows)

## 🎯 Use Cases

- **Network Discovery**: Rapidly identify live hosts and services across large IP ranges
- **Asset Inventory**: Maintain accurate records of network infrastructure and services
- **Vulnerability Assessment**: Detect known vulnerabilities through banner analysis and CVE matching
- **Security Monitoring**: Continuous scanning for unauthorized services and configuration changes
- **Penetration Testing**: Reconnaissance and information gathering during security assessments

## 🚀 Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/pentora-ai/pentora.git
cd pentora

# Build the binary
go build -o pentora ./cmd/pentora

# Verify installation
./pentora version
```

### Basic Usage

```bash
# Scan a single target with default ports (22,80,443)
pentora scan --targets 192.168.1.1

# Scan a CIDR range with custom ports
pentora scan --targets 192.168.1.0/24 --ports 21,22,80,443,8080

# Perform port discovery (scan ports 1-1000)
pentora scan --targets 192.168.1.1 --discover

# Enable vulnerability detection
pentora scan --targets 192.168.1.1 --vuln

# Scan with fingerprinting enabled
pentora scan --targets 10.0.0.0/24 --fingerprint

# List previous scan results
pentora workspace list

# Export scan results to JSON
pentora workspace export <scan-id> --format json -o results.json
```

### Server Mode

```bash
# Start the Pentora server
pentora server start --host 0.0.0.0 --port 8080

# Submit a scan via API
curl -X POST http://localhost:8080/api/v1/scans \
  -H "Content-Type: application/json" \
  -d '{"targets": ["192.168.1.0/24"], "ports": [22,80,443]}'

# Check scan status
pentora server status
```

### DAG Management

Validate and manage DAG (Directed Acyclic Graph) definitions:

```bash
# Validate a DAG definition file
pentora dag validate scan-dag.yaml

# Export the internal scan DAG to YAML for inspection
pentora dag export --targets 192.168.1.0/24 --output scan.yaml

# Export with vulnerability evaluation enabled
pentora dag export --targets 10.0.0.1 --vuln --output full-scan.yaml

# Strict validation (treat warnings as errors)
pentora dag validate dag.yaml --strict

# Output validation results as JSON (for CI/CD)
pentora dag validate dag.yaml --json
```

### YAML Plugin System (New)

Create custom vulnerability checks without writing Go code using YAML plugins:

```bash
# Load and evaluate YAML plugins
pentora scan --targets 192.168.1.1 --plugins ./my-plugins/

# List available plugins
pentora plugin list

# Validate a plugin definition
pentora plugin validate ssh-cve-check.yaml
```

Example YAML plugin (`ssh-vuln-check.yaml`):

```yaml
name: SSH Vulnerability Check
version: 1.0.0
type: evaluation
author: security-team

metadata:
  cve: CVE-2024-XXXXX
  severity: high
  tags: [ssh, authentication, cve]

# Trigger when SSH version is detected
triggers:
  - data_key: ssh.version
    condition: exists
    value: true

# Match vulnerable versions
match:
  logic: AND
  rules:
    - field: ssh.version
      operator: version_lt
      value: "8.5"
    - field: ssh.banner
      operator: contains
      value: "OpenSSH"

output:
  vulnerability: true
  message: "SSH version vulnerable to authentication bypass"
  remediation: "Upgrade OpenSSH to version 8.5 or higher"
```

**Supported Operators**:
- String: `equals`, `contains`, `startsWith`, `endsWith`, `matches` (regex)
- Numeric: `gt`, `gte`, `lt`, `lte`, `between`
- Version: `version_eq`, `version_lt`, `version_gt`, `version_lte`, `version_gte`, `version_between`
- Logical: `exists`, `in`, `notIn`

**Match Logic**: `AND`, `OR`, `NOT` for combining rules

See [pkg/plugin/testdata/plugins/](pkg/plugin/testdata/plugins/) for more examples.

## 📂 Project Structure

```
pentora/
├── cmd/pentora/           # CLI entry point
├── pkg/
│   ├── api/              # REST and gRPC API servers
│   ├── appctx/           # Application context management
│   ├── cli/              # CLI command implementations
│   ├── config/           # Configuration loading and validation
│   ├── engine/           # DAG execution engine
│   ├── event/            # Event system for hooks and notifications
│   ├── fingerprint/      # Service fingerprinting engine
│   ├── hook/             # Hook system for custom automation
│   ├── logging/          # Structured logging utilities
│   ├── modules/          # Core scanning modules (discovery, probing, etc.)
│   ├── netutil/          # Network utilities and helpers
│   ├── parser/           # Protocol parsers (SSH, HTTP, FTP, etc.)
│   ├── plugin/           # YAML plugin system + legacy CVE matchers
│   ├── scan/             # Scan orchestration and coordination
│   ├── scanexec/         # Scan execution logic
│   ├── scanner/          # Port scanner and banner grabber
│   ├── server/           # HTTP/gRPC server implementation
│   ├── workspace/        # Scan result storage and queries
│   └── version/          # Version information
├── ui/                   # Web UI (React/TypeScript) - In Development
├── docs/                 # Documentation website (Docusaurus)
└── scripts/              # Build and packaging scripts
```

## 🔧 Architecture

Pentora is built around several core concepts:

### DAG Execution Engine

The DAG (Directed Acyclic Graph) engine orchestrates scan workflows by modeling dependencies between tasks. This enables:

- Parallel execution of independent tasks
- Automatic dependency resolution
- Efficient resource utilization
- Flexible workflow composition

### Module System

Modules are self-contained units that perform specific scan phases:

- **Discovery Module**: Host discovery using multiple techniques
- **Port Scanner Module**: Fast TCP/UDP port scanning
- **Banner Grabber Module**: Service banner collection
- **Fingerprint Module**: Service version identification
- **Vulnerability Module**: CVE matching and risk assessment

### Workspace

All scan results are stored in a local workspace with:

- SQLite database for structured data
- File-based artifact storage
- Full-text search capabilities
- Historical scan tracking

## 🔌 Extending Pentora

### Custom Parsers

Add support for new protocols:

```go
package parser

func init() {
    Register("myprotocol", &MyProtocolParser{})
}

type MyProtocolParser struct{}

func (p *MyProtocolParser) Parse(banner string) (map[string]string, error) {
    // Parse banner and extract metadata
    return map[string]string{
        "service": "myservice",
        "version": "1.0.0",
    }, nil
}
```

### Custom Plugins

#### YAML Plugins (Recommended)

Create vulnerability detection rules without writing Go code:

```yaml
name: HTTP Vulnerable App Detection
version: 1.0.0
type: evaluation
author: security-team

metadata:
  cve: CVE-2024-XXXXX
  severity: high
  tags: [http, web, cve]

triggers:
  - data_key: http.server
    condition: exists
    value: true

match:
  logic: AND
  rules:
    - field: http.server
      operator: contains
      value: "VulnerableApp/1.0"
    - field: service.port
      operator: equals
      value: 8080

output:
  vulnerability: true
  message: "Vulnerable application detected"
  remediation: "Upgrade to version 2.0 or higher"
```

**Benefits**:
- ✅ No recompilation needed
- ✅ Easy to create and share
- ✅ Declarative and readable
- ✅ Supports complex matching logic

#### Go Plugins (Legacy)

For advanced use cases, you can still create Go-based plugins:

```go
package plugin

func init() {
    Register(&Plugins{
        ID:   "custom_vuln_check",
        Name: "Custom Vulnerability Check",
        RequirePorts: []int{8080},
        RequireKeys:  []string{"http/server"},
        MatchFunc: func(ctx map[string]string) *MatchResult {
            if strings.Contains(ctx["http/server"], "VulnerableApp/1.0") {
                return &MatchResult{
                    CVE:     []string{"CVE-2024-XXXXX"},
                    Summary: "Vulnerable application detected",
                }
            }
            return nil
        },
    })
}
```

### Event Hooks

React to scan events:

```go
package main

import "github.com/pentora-ai/pentora/pkg/hook"

func init() {
    hook.Register("on_scan_complete", func(data interface{}) error {
        // Send notification, update database, etc.
        return nil
    })
}
```

## 📖 Documentation

For comprehensive documentation, visit:

- **Documentation Site**: https://docs.pentora.ai (In Development)
- **Getting Started Guide**: [docs/getting-started/installation.md](docs/docs/getting-started/installation.md)
- **CLI Reference**: [docs/cli/overview.md](docs/docs/cli/overview.md)
- **Architecture Overview**: [docs/architecture/overview.md](docs/docs/architecture/overview.md)

## 🤝 Contributing

We welcome contributions! Pentora is in active development, and we're building the foundation for a powerful security scanning platform.

Areas where we need help:

- Core scanning engine improvements
- Protocol parser implementations
- Vulnerability detection plugins
- Documentation and examples
- UI/UX design and implementation
- Testing and bug reports

Please read our contributing guidelines before submitting pull requests.

## 📋 Roadmap

- [ ] Complete core scanning engine
- [ ] REST/gRPC API implementation
- [ ] Web UI for scan management
- [ ] Distributed scanning support
- [ ] Enhanced reporting capabilities
- [ ] Integration with popular security tools
- [ ] Cloud deployment guides
- [ ] First stable release (v1.0.0)

## 📜 License

Licensed under the Apache License, Version 2.0. See [LICENSE.md](LICENSE.md) for details.

## 🔗 Links

- **Website**: https://pentora.ai
- **Documentation**: https://docs.pentora.ai
- **GitHub**: https://github.com/pentora-ai/pentora
- **Issues**: https://github.com/pentora-ai/pentora/issues
- **Discussions**: https://github.com/pentora-ai/pentora/discussions

---

**Note**: Pentora is under active development. Star the repository to stay updated on releases and new features!

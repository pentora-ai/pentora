# CLI Overview

The Pentora command-line interface (CLI) provides direct access to all scanning capabilities, workspace management, and server control. The CLI is designed for technical operators performing ad-hoc scans, troubleshooting, and integration into automation workflows.

## Philosophy

### CLI vs UI Design

Pentora separates technical and non-technical user experiences:

**CLI** targets:
- Security operators and penetration testers
- DevOps engineers and SREs
- Automation and CI/CD pipelines
- Power users requiring fine-grained control

**UI** targets:
- Security managers and executives
- Compliance auditors
- Non-technical stakeholders
- Scheduled scan management

The CLI remains fully functional without the server component, while the UI requires the server for centralized operations.

### Self-Sufficiency

The CLI never accesses server internals directly. When interacting with a remote Pentora server, the CLI uses the REST/gRPC API just like any external client.

**Local Mode** (no server):
```bash
pentora scan --targets 192.168.1.0/24
# Executes scan locally, writes to local workspace
```

**Remote Mode** (with server):
```bash
pentora scan --targets 192.168.1.0/24 --server https://pentora.company.com
# Submits scan job to server via API
```

## Command Structure

```
pentora <command> [subcommand] [flags] [arguments]
```

## Primary Commands

### pentora scan

Execute security scans:

```bash
pentora scan --targets 192.168.1.0/24
```

Performs complete scan pipeline or selective phases. Most commonly used command.

See [Scan Command Reference](./scan.md) for details.

### pentora workspace

Manage workspace and scan results:

```bash
pentora workspace list              # List all scans
pentora workspace show <scan-id>    # Show scan details
pentora workspace gc                # Garbage collection
```

See [Workspace Commands](./workspace.md) for details.

### pentora server

Control Pentora server daemon:

```bash
pentora server start                # Start server
pentora server stop                 # Stop server
pentora server status               # Check server status
```

See [Server Commands](./server.md) for details.

### pentora fingerprint

Manage fingerprint catalogs:

```bash
pentora fingerprint sync            # Update fingerprint database
pentora fingerprint list            # List available rules
```

See [Fingerprint Commands](./fingerprint.md) for details.

### pentora version

Display version information:

```bash
pentora version
```

Output:
```
Pentora version 1.0.0
Build: 20231006-a1b2c3d
Go version: go1.21.3
Platform: linux/amd64
```

### pentora dag

Validate and inspect DAG definitions:

```bash
pentora dag validate scan-profile.yaml   # Validate DAG
pentora dag show scan-profile.yaml       # Visualize DAG
```

## Quick Start

### Basic Network Scan

```bash
pentora scan --targets 192.168.1.0/24
```

### Scan with Vulnerability Detection

```bash
pentora scan --targets 192.168.1.100 --vuln
```

### List Scan Results

```bash
pentora workspace list
```

### Export Results

```bash
pentora workspace export <scan-id> --format json -o report.json
```

## Learn More

<div className="row" style={{marginTop: '1.5rem'}}>
  <div className="col col--6">
    <div className="card">
      <div className="card__header">
        <h3>📋 Common Workflows</h3>
      </div>
      <div className="card__body">
        <p>Learn common scanning patterns and use cases</p>
        <a href="./common-workflows" className="button button--primary">View Workflows</a>
      </div>
    </div>
  </div>
  <div className="col col--6">
    <div className="card">
      <div className="card__header">
        <h3>📊 Output Formats</h3>
      </div>
      <div className="card__body">
        <p>Understand different output formats and verbosity levels</p>
        <a href="./output-formats" className="button button--primary">Learn More</a>
      </div>
    </div>
  </div>
</div>

<div className="row" style={{marginTop: '1rem'}}>
  <div className="col col--6">
    <div className="card">
      <div className="card__header">
        <h3>⚙️ Configuration</h3>
      </div>
      <div className="card__body">
        <p>Configure CLI using files, environment variables, and flags</p>
        <a href="./configuration" className="button button--primary">Configure CLI</a>
      </div>
    </div>
  </div>
  <div className="col col--6">
    <div className="card">
      <div className="card__header">
        <h3>🔗 Integrations</h3>
      </div>
      <div className="card__body">
        <p>Integrate with CI/CD, automation tools, and scripts</p>
        <a href="./integrations" className="button button--primary">View Examples</a>
      </div>
    </div>
  </div>
</div>

## Command Reference

| Command | Description |
|---------|-------------|
| [scan](./scan.md) | Execute security scans |
| [workspace](./workspace.md) | Manage scan results and workspace |
| [server](./server.md) | Control Pentora server |
| [fingerprint](./fingerprint.md) | Manage fingerprint database |
| [dag](./dag.md) | Validate and inspect DAG definitions |
| [version](./version.md) | Show version information |

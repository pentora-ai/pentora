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

### Global Flags

Available on all commands:

```bash
--config string          Config file path (default: ~/.config/pentora/config.yaml)
--workspace-dir string   Workspace directory (default: OS-specific)
--no-workspace          Disable workspace persistence
--log-level string      Logging level: debug, info, warn, error (default: info)
--log-format string     Log format: json, text (default: text)
--verbosity int         Increase logging verbosity (0-3)
--quiet                 Suppress non-error output
--no-color              Disable colored output
--help, -h              Show help
--version, -v           Show version
```

### Environment Variables

Override config and flags:

```bash
PENTORA_CONFIG           # Config file path
PENTORA_WORKSPACE        # Workspace directory
PENTORA_LOG_LEVEL        # Logging level
PENTORA_SERVER           # Server URL for remote mode
PENTORA_API_TOKEN        # API authentication token
```

## Primary Commands

### pentora scan

Execute security scans:

```bash
pentora scan --targets 192.168.1.0/24
```

Performs complete scan pipeline or selective phases. Most commonly used command.

See [Scan Command Reference](/docs/cli/scan) for details.

### pentora workspace

Manage workspace and scan results:

```bash
pentora workspace list              # List all scans
pentora workspace show <scan-id>    # Show scan details
pentora workspace gc                # Garbage collection
```

See [Workspace Commands](/docs/cli/workspace) for details.

### pentora server

Control Pentora server daemon:

```bash
pentora server start                # Start server
pentora server stop                 # Stop server
pentora server status               # Check server status
```

See [Server Commands](/docs/cli/server) for details.

### pentora fingerprint

Manage fingerprint catalogs:

```bash
pentora fingerprint sync            # Update fingerprint database
pentora fingerprint list            # List available rules
```

See [Fingerprint Commands](/docs/cli/fingerprint) for details.

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

## Common Workflows

### Quick Network Scan

Discover hosts and identify services:

```bash
pentora scan --targets 192.168.1.0/24 --profile standard
```

### Vulnerability Assessment

Full scan with CVE checks:

```bash
pentora scan --targets critical-servers.txt --vuln --output report.json
```

### Discovery Only

Identify live hosts without port scanning:

```bash
pentora scan --targets 10.0.0.0/16 --only-discover
```

### Resume from Discovery

Scan previously discovered hosts:

```bash
# First, discover hosts
pentora scan --targets 10.0.0.0/16 --only-discover

# Then scan discovered hosts
pentora workspace show <scan-id> | jq -r '.discovered_hosts[].ip' > live-hosts.txt
pentora scan --target-file live-hosts.txt --no-discover
```

### Custom Workspace

Use non-default workspace location:

```bash
pentora scan --targets 192.168.1.100 --workspace-dir /data/scans
```

### Stateless Scan

No workspace persistence (ephemeral):

```bash
pentora scan --targets 192.168.1.100 --no-workspace --output results.json
```

### Remote Execution

Submit scan to remote server:

```bash
export PENTORA_SERVER=https://pentora.company.com
export PENTORA_API_TOKEN=your-token-here

pentora scan --targets 192.168.1.0/24 --server $PENTORA_SERVER
```

### Scheduled Scan (Server Mode)

Create recurring scan:

```bash
pentora scan --targets 192.168.1.0/24 \
  --schedule "0 2 * * *" \
  --profile standard \
  --notify slack://security-alerts
```

Requires server running.

## Output Formats

### Terminal Output (default)

Human-readable text output:

```bash
pentora scan --targets 192.168.1.100
```

```
[INFO] Scan started: scan-20231006-143022
[INFO] Discovery: 1/1 hosts live
[INFO] Port scanning: 192.168.1.100
[INFO] Found 5 open ports
[INFO] Fingerprinting services...
[INFO] Scan complete in 45s
[INFO] Results: ~/.local/share/pentora/scans/20231006-143022-a1b2c3/
```

### JSON Output

Machine-readable structured output:

```bash
pentora scan --targets 192.168.1.100 --output json
```

```json
{
  "scan_id": "20231006-143022-a1b2c3",
  "timestamp": "2023-10-06T14:30:22Z",
  "targets": ["192.168.1.100"],
  "results": [
    {
      "host": "192.168.1.100",
      "ports": [
        {"port": 22, "protocol": "tcp", "state": "open", "service": "ssh"}
      ]
    }
  ]
}
```

### JSONL Output

Line-delimited JSON (streamable):

```bash
pentora scan --targets 192.168.1.100 --output jsonl
```

```jsonl
{"timestamp":"2023-10-06T14:30:22Z","host":"192.168.1.100","state":"up"}
{"timestamp":"2023-10-06T14:30:45Z","host":"192.168.1.100","port":22,"state":"open"}
```

### CSV Output

Tabular format for spreadsheets:

```bash
pentora scan --targets 192.168.1.100 --output csv
```

```csv
host,port,protocol,state,service,version
192.168.1.100,22,tcp,open,ssh,OpenSSH 8.2p1
192.168.1.100,80,tcp,open,http,nginx 1.18.0
```

### File Output

Write results to file:

```bash
pentora scan --targets 192.168.1.100 -o results.json
pentora scan --targets 192.168.1.100 --output-file report.csv --format csv
```

## Progress and Verbosity

### Progress Display

Show real-time progress:

```bash
pentora scan --targets 192.168.1.0/24 --progress
```

```
[=====>                    ] 25% (50/200 hosts scanned)
Currently scanning: 192.168.1.50-192.168.1.60
Elapsed: 2m 15s | Estimated remaining: 6m 45s
```

### Verbosity Levels

Increase log detail:

```bash
# Standard logging (info level)
pentora scan --targets 192.168.1.100

# Verbose (debug level)
pentora scan --targets 192.168.1.100 -v

# Very verbose (trace level with module details)
pentora scan --targets 192.168.1.100 -vv

# Maximum verbosity (all events and data flow)
pentora scan --targets 192.168.1.100 -vvv
```

### Quiet Mode

Suppress all output except errors:

```bash
pentora scan --targets 192.168.1.100 --quiet
```

Use in scripts where only exit code matters:

```bash
if pentora scan --targets 192.168.1.100 --quiet; then
    echo "Scan successful"
else
    echo "Scan failed"
fi
```

## Configuration Files

### Precedence

Configuration loaded in order (later overrides earlier):

1. Builtin defaults
2. System config: `/etc/pentora/config.yaml`
3. User config: `~/.config/pentora/config.yaml`
4. Workspace config: `<workspace>/config/pentora.yaml`
5. Custom config: `--config /path/to/config.yaml`
6. Environment variables: `PENTORA_*`
7. CLI flags: `--flag value`

### Config File Format

YAML format:

```yaml
# ~/.config/pentora/config.yaml
workspace:
  dir: ~/.local/share/pentora
  enabled: true

scanner:
  default_profile: standard
  rate: 1000
  timeout: 3s

fingerprint:
  cache_dir: ${workspace}/cache/fingerprints
  catalog:
    remote_url: https://catalog.pentora.io/fingerprints.yaml

logging:
  level: info
  format: text

server:
  bind: 0.0.0.0:8080
  workers: 4
```

See [Configuration Overview](/docs/configuration/overview) for complete schema.

## Exit Codes

Pentora uses standard Unix exit codes:

| Code | Meaning |
|------|---------|
| 0    | Success |
| 1    | General error |
| 2    | Usage error (invalid arguments) |
| 3    | Configuration error |
| 4    | Network error |
| 5    | Permission error |
| 6    | Timeout |
| 7    | Scan failed (partial results may be available) |

Use in scripts:

```bash
pentora scan --targets 192.168.1.100
case $? in
    0) echo "Success" ;;
    4) echo "Network error" ;;
    7) echo "Scan failed, check logs" ;;
    *) echo "Unknown error" ;;
esac
```

## Shell Completion

Generate shell completion scripts:

### Bash

```bash
pentora completion bash > /etc/bash_completion.d/pentora
source /etc/bash_completion.d/pentora
```

### Zsh

```bash
pentora completion zsh > ~/.zsh/completion/_pentora
# Add to ~/.zshrc:
fpath=(~/.zsh/completion $fpath)
autoload -U compinit && compinit
```

### Fish

```bash
pentora completion fish > ~/.config/fish/completions/pentora.fish
```

### PowerShell

```powershell
pentora completion powershell | Out-String | Invoke-Expression
```

## Debugging

### Enable Debug Logging

```bash
pentora scan --targets 192.168.1.100 --log-level debug
```

### Log to File

```bash
pentora scan --targets 192.168.1.100 2> scan-debug.log
```

### Trace Execution

Show detailed execution flow:

```bash
pentora scan --targets 192.168.1.100 --log-level trace --log-format json | jq
```

### Dry Run

Validate configuration without executing:

```bash
pentora scan --targets 192.168.1.100 --dry-run
```

Shows what would be executed without actually running the scan.

## Integration Examples

### Cron Scheduling

```bash
# /etc/cron.d/pentora-scan
0 2 * * * pentora-user /usr/local/bin/pentora scan --targets /etc/pentora/targets.txt --quiet
```

### CI/CD Pipeline

```yaml
# .gitlab-ci.yml
security-scan:
  stage: test
  image: pentora/pentora:latest
  script:
    - pentora scan --targets $CI_ENVIRONMENT_URL --output report.json
  artifacts:
    reports:
      pentora: report.json
```

### Ansible Playbook

```yaml
- name: Run Pentora scan
  command: >
    pentora scan
    --targets {{ target_network }}
    --profile standard
    --output /tmp/scan-results.json
  register: scan_result

- name: Parse scan results
  set_fact:
    vulnerabilities: "{{ lookup('file', '/tmp/scan-results.json') | from_json }}"
```

### Python Script

```python
import subprocess
import json

result = subprocess.run(
    ['pentora', 'scan', '--targets', '192.168.1.100', '--output', 'json'],
    capture_output=True,
    text=True
)

if result.returncode == 0:
    scan_data = json.loads(result.stdout)
    print(f"Found {len(scan_data['results'])} hosts")
else:
    print(f"Scan failed: {result.stderr}")
```

## Best Practices

### 1. Use Configuration Files

Avoid long command lines:

```bash
# Instead of:
pentora scan --targets 192.168.1.0/24 --rate 5000 --timeout 5s --vuln --profile deep

# Use config file:
pentora scan --targets 192.168.1.0/24 --config deep-scan.yaml
```

### 2. Leverage Profiles

Create reusable scan profiles:

```yaml
# ~/.config/pentora/profiles/production.yaml
scanner:
  rate: 500          # Conservative rate for production
  profile: standard
  vuln: true

fingerprint:
  enabled: true

notifications:
  channels: [slack://security-prod]
```

```bash
pentora scan --targets production.txt --profile production
```

### 3. Separate Discovery and Scanning

For large networks, split phases:

```bash
# Phase 1: Fast discovery
pentora scan --targets 10.0.0.0/8 --only-discover -o live-hosts.txt

# Phase 2: Detailed scan of live hosts
pentora scan --target-file live-hosts.txt --no-discover --vuln
```

### 4. Use Workspaces for Organization

Separate workspaces per project:

```bash
pentora scan --targets client-a.txt --workspace-dir /data/pentora/client-a
pentora scan --targets client-b.txt --workspace-dir /data/pentora/client-b
```

### 5. Automate Cleanup

Prevent workspace bloat:

```bash
# Weekly cleanup cron job
0 0 * * 0 pentora workspace gc --older-than 30d --quiet
```

## Next Steps

- [Scan Command Reference](/docs/cli/scan) - Complete scan command documentation
- [Workspace Commands](/docs/cli/workspace) - Workspace management
- [Server Commands](/docs/cli/server) - Server control
- [Configuration Overview](/docs/configuration/overview) - Configuration file structure

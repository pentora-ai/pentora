---
sidebar_position: 2
---

# Quick Start Guide

Get up and running with Pentora in 5 minutes. This guide walks you through basic scanning operations.

## Prerequisites

- Pentora installed ([Installation Guide](./installation.md))
- Network access to target systems
- Administrator/root privileges for network scanning

## Your First Scan

### Basic Network Scan

Scan a single IP address:

```bash
pentora scan 192.168.1.100
```

Output:

```
[INFO] Starting scan: 192.168.1.100
[INFO] Discovery: 1 host alive
[INFO] Scanning ports: 22,80,443,3306,5432,6379,8080...
[INFO] Open ports found: 22, 80, 443

Host: 192.168.1.100
┌──────┬──────────┬─────────┬────────────────────────────┐
│ Port │ Protocol │ State   │ Service                    │
├──────┼──────────┼─────────┼────────────────────────────┤
│ 22   │ tcp      │ open    │ SSH-2.0-OpenSSH_8.9p1      │
│ 80   │ tcp      │ open    │ HTTP/1.1 (nginx/1.21.6)    │
│ 443  │ tcp      │ open    │ HTTPS (nginx/1.21.6)       │
└──────┴──────────┴─────────┴────────────────────────────┘

Scan completed in 2.34s
```

### Scan Multiple Hosts

```bash
# Scan CIDR range
pentora scan 192.168.1.0/24

# Scan multiple IPs
pentora scan 192.168.1.100,192.168.1.101,192.168.1.102

# Scan IP range
pentora scan 192.168.1.100-110
```

### Custom Port Scanning

```bash
# Scan specific ports
pentora scan 192.168.1.100 --ports 22,80,443,8080

# Scan port range
pentora scan 192.168.1.100 --ports 1-1000

# Scan all common ports
pentora scan 192.168.1.100 --ports common

# Scan all 65535 ports
pentora scan 192.168.1.100 --ports all
```

## Scan Modes

### Discovery Only

Quickly find live hosts without port scanning:

```bash
pentora scan 192.168.1.0/24 --only-discover
```

Output:

```
Discovered 12 active hosts:
192.168.1.1    (gateway)
192.168.1.10   (server)
192.168.1.100  (workstation)
...
```

### Skip Discovery

Scan known hosts directly (faster when targets are known):

```bash
pentora scan 192.168.1.100 --no-discover
```

### Vulnerability Scanning

Enable vulnerability assessment:

```bash
pentora scan 192.168.1.100 --vuln
```

Output includes CVE matches:

```
Host: 192.168.1.100
Port 22: SSH-2.0-OpenSSH_7.4
  ⚠️  CVE-2018-15919 (Medium): OpenSSH remote code execution
  ⚠️  CVE-2016-0777 (High): Information disclosure vulnerability

Port 80: Apache/2.4.29
  🔴 CVE-2021-44790 (Critical): Buffer overflow in mod_lua
```

## Output Formats

### JSON Export

```bash
pentora scan 192.168.1.100 --format json -o results.json
```

### CSV Export

```bash
pentora scan 192.168.1.100 --format csv -o results.csv
```

### PDF Report

```bash
pentora scan 192.168.1.100 --format pdf -o report.pdf
```

### Multiple Formats

```bash
pentora scan 192.168.1.100 -o results.json -o report.pdf
```

## Performance Tuning

### Concurrency

Control scan speed with concurrency settings:

```bash
# Slow, stealthy scan
pentora scan 192.168.1.0/24 --rate 10

# Fast scan (default)
pentora scan 192.168.1.0/24 --rate 100

# Maximum speed (aggressive)
pentora scan 192.168.1.0/24 --rate 1000 --timeout 500ms
```

### Timeout Configuration

```bash
# Quick timeout for fast networks
pentora scan 192.168.1.100 --timeout 200ms

# Longer timeout for slow networks
pentora scan 192.168.1.100 --timeout 5s
```

## Storage Operations

### List Scans

View all stored scans:

```bash
pentora storage list
```

Output:

```
┌───────────────────┬────────────┬─────────────────────┬────────┐
│ Scan ID           │ Targets    │ Timestamp           │ Status │
├───────────────────┼────────────┼─────────────────────┼────────┤
│ scan-2025-10-06-1 │ 192.168... │ 2025-10-06 10:30:15 │ done   │
│ scan-2025-10-06-2 │ 10.0.0...  │ 2025-10-06 11:15:42 │ done   │
└───────────────────┴────────────┴─────────────────────┴────────┘
```

### View Scan Details

```bash
pentora storage show scan-2025-10-06-1
```

### Export Scan

```bash
pentora storage export scan-2025-10-06-1 --format json -o export.json
```

### Clean Up Old Scans

```bash
# Delete scans older than 30 days
pentora storage gc --older-than 30d

# Delete all but last 10 scans
pentora storage gc --keep-last 10
```

## Practical Examples

### Web Server Assessment

```bash
pentora scan example.com --ports 80,443,8080,8443 --vuln
```

### Database Server Scan

```bash
pentora scan 192.168.1.50 --ports 3306,5432,1433,27017 --vuln
```

### Full Network Audit

```bash
pentora scan 192.168.1.0/24 \
  --ports all \
  --vuln \
  --format pdf \
  -o network-audit-$(date +%Y%m%d).pdf
```

### Continuous Monitoring

```bash
# Scan and compare with previous results
pentora scan 192.168.1.0/24 --compare-with scan-2025-10-05-1
```

## Configuration File

Create a reusable scan profile:

```yaml
# ~/.config/pentora/config.yaml
scan:
  default_ports: [22,80,443,3306,5432,8080]
  timeout: 2s
  rate: 100

storage:
  dir: /var/pentora/storage
  retention: 90d

logging:
  level: info
  format: json
```

Run with config:

```bash
pentora scan 192.168.1.0/24 --config ~/.config/pentora/config.yaml
```

## Common Use Cases

### 1. Quick Port Check

```bash
pentora scan 192.168.1.100 --ports 22,80,443
```

### 2. Service Discovery

```bash
pentora scan 192.168.1.0/24 --only-discover
```

### 3. Vulnerability Assessment

```bash
pentora scan 192.168.1.100 --vuln --format pdf -o vuln-report.pdf
```

### 4. Compliance Scan

```bash
pentora scan 192.168.1.0/24 --compliance pci-dss --format pdf
```

### 5. Scheduled Scanning

```bash
# Add to crontab
0 2 * * * /usr/local/bin/pentora scan 192.168.1.0/24 --vuln -o /var/reports/daily-scan.json
```

## Stateless Mode

Run without storage persistence (like Nmap):

```bash
pentora scan 192.168.1.100 --no-storage
```

Results print to stdout only, nothing saved to disk.

## Getting Help

### Command Help

```bash
# General help
pentora --help

# Command-specific help
pentora scan --help

# List all commands
pentora --help
```

### Check Version

```bash
pentora version
```

### Enable Verbose Output

```bash
# Detailed logs
pentora scan 192.168.1.100 --verbose

# Debug-level logging
pentora scan 192.168.1.100 --verbosity debug
```

## Next Steps

Now that you've run basic scans, explore:

- 📖 [First Scan Tutorial](./first-scan.md) - Detailed walkthrough with explanations
- 🎯 [Core Concepts](../concepts/overview.md) - Understand Pentora's architecture
- 🔧 [CLI Reference](../cli/overview.md) - Complete command reference
- ⚙️ [Configuration](../configuration/overview.md) - Advanced configuration options
- 🛡️ [Vulnerability Scanning](../guides/vulnerability-assessment.md) - Deep dive into vuln assessment

## Troubleshooting

### Permission Errors

```bash
# Run with sudo for network scans
sudo pentora scan 192.168.1.0/24
```

### Slow Scans

```bash
# Increase concurrency
pentora scan 192.168.1.0/24 --rate 500
```

### No Results

```bash
# Enable debug logging
pentora scan 192.168.1.100 --verbosity debug
```

For more troubleshooting, see the [Troubleshooting Guide](../troubleshooting/common-issues.md).

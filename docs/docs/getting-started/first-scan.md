# Your First Scan

This guide walks you through running your first scan with Pentora, from installation to viewing results.

## Prerequisites

Before starting your first scan, ensure you have:

- Pentora installed on your system
- Appropriate network permissions to scan target systems
- Authorization to scan the target networks (never scan without permission)

## Basic Scan Workflow

### Step 1: Verify Installation

First, verify that Pentora is correctly installed:

```bash
pentora version
```

You should see version information and build details.

### Step 2: Run a Simple Discovery Scan

Start with a basic discovery scan to identify live hosts:

```bash
pentora scan --targets 192.168.1.0/24 --only-discover
```

This command will:
1. Parse and validate the target CIDR range
2. Apply any configured blocklists
3. Perform ICMP/ARP probes to identify live hosts
4. Display results in your terminal

**Expected output:**
```
[INFO] Starting discovery scan for 192.168.1.0/24
[INFO] Found 15 live hosts
[INFO] Results saved to storage: ~/.local/share/pentora/scans/<scan-id>/
```

### Step 3: Run a Full Port Scan

Once you've identified live hosts, run a comprehensive scan:

```bash
pentora scan --targets 192.168.1.100
```

This performs the complete scan pipeline:
- **Target ingestion**: Validates and expands your target list
- **Asset discovery**: Identifies live hosts (can be skipped with `--no-discover`)
- **Port scanning**: TCP/UDP probes with rate limiting
- **Service fingerprinting**: Banner grabbing and protocol detection
- **Asset profiling**: Device/OS/application identification
- **Vulnerability evaluation**: CVE matching and misconfiguration checks
- **Reporting**: Structured output generation

### Step 4: Understanding Results

Scan results are stored in your storage directory (default: `~/.local/share/pentora/scans/<scan-id>/`):

```
scans/
└── 20231006-143022-a1b2c3/
    ├── request.json      # Original scan parameters
    ├── status.json       # Scan status and metadata
    ├── results.jsonl     # Line-delimited JSON results
    └── artifacts/        # Additional scan artifacts
```

View results directly:

```bash
cat ~/.local/share/pentora/scans/<scan-id>/results.jsonl | jq
```

### Step 5: Run Vulnerability Assessment

Enable vulnerability checks to identify security issues:

```bash
pentora scan --targets 192.168.1.100 --vuln
```

The `--vuln` flag activates vulnerability evaluation modules that:
- Match service versions against CVE databases
- Check for common misconfigurations
- Identify weak protocols and ciphers
- Flag exposed sensitive services

## Common Scan Options

### Target Specification

Multiple formats are supported:

```bash
# Single IP
pentora scan --targets 192.168.1.100

# CIDR notation
pentora scan --targets 192.168.1.0/24

# IP range
pentora scan --targets 192.168.1.1-192.168.1.254

# Hostname
pentora scan --targets example.com

# Multiple targets (comma-separated)
pentora scan --targets "192.168.1.100,192.168.1.200,10.0.0.1/24"

# From file
pentora scan --target-file targets.txt
```

### Controlling Scan Phases

Control which phases execute:

```bash
# Discovery only (identify live hosts)
pentora scan --targets 192.168.1.0/24 --only-discover

# Skip discovery (targets known to be live)
pentora scan --targets 192.168.1.100 --no-discover

# Full scan with vulnerability checks
pentora scan --targets 192.168.1.100 --vuln
```

### Scan Profiles

Use predefined profiles for common scenarios:

```bash
# Quick scan (top 100 ports)
pentora scan --targets 192.168.1.100 --profile quick

# Standard scan (top 1000 ports)
pentora scan --targets 192.168.1.100 --profile standard

# Deep scan (all 65535 ports)
pentora scan --targets 192.168.1.100 --profile deep

# Web application focus
pentora scan --targets example.com --profile webapp
```

### Storage Control

Manage where scan data is stored:

```bash
# Use custom storage directory
pentora scan --targets 192.168.1.100 --storage-dir /path/to/storage

# Disable storage (stateless mode, no persistence)
pentora scan --targets 192.168.1.100 --no-storage

# Clean old scans
pentora storage gc --older-than 30d
```

## Understanding Output

### Terminal Output

During execution, Pentora displays:

```
[INFO] Scan started: scan-20231006-143022
[INFO] Target ingestion: 1 target, 0 blocked
[INFO] Discovery: 1/1 hosts live
[INFO] Port scanning: 5 open ports found
[INFO] Fingerprinting: 5 services identified
[INFO] Vulnerability checks: 2 issues found
[INFO] Scan complete in 45s
[INFO] Results: ~/.local/share/pentora/scans/20231006-143022-a1b2c3/
```

### JSON Results

Results are stored in line-delimited JSON format:

```json
{
  "timestamp": "2023-10-06T14:30:45Z",
  "target": "192.168.1.100",
  "port": 22,
  "protocol": "tcp",
  "state": "open",
  "service": {
    "name": "ssh",
    "product": "OpenSSH",
    "version": "8.2p1",
    "os": "Ubuntu Linux"
  },
  "fingerprints": [
    {
      "match": "openssh",
      "confidence": 95,
      "source": "banner"
    }
  ]
}
```

## Next Steps

Now that you've completed your first scan, explore:

- [Scan Pipeline Concepts](/concepts/scan-pipeline) - Understanding the 9-stage pipeline
- [CLI Reference](/cli/scan) - Complete command reference
- [Fingerprinting System](/concepts/fingerprinting) - How service detection works
- [Vulnerability Assessment Guide](/guides/vulnerability-assessment) - Deep dive into vuln checks

## Troubleshooting

### Permission Errors

If you encounter permission errors:

```bash
# Run with sudo for raw socket access (required for ICMP)
sudo pentora scan --targets 192.168.1.0/24
```

### No Hosts Found

If discovery returns no results:
- Verify network connectivity: `ping <target>`
- Check firewall rules blocking ICMP
- Try TCP-based discovery: `--discover-profile tcp`
- Use `--no-discover` if hosts are known to be live

### Slow Scans

If scans run slowly:
- Reduce concurrency: `--rate 100`
- Use a faster profile: `--profile quick`
- Skip unnecessary phases: `--no-vuln`

## Related Documentation

- [Storage Concept](/concepts/storage) - Understanding storage structure
- [Scan Profiles](/configuration/scan-profiles) - Customizing scan behavior
- [Network Scanning Best Practices](/guides/network-scanning) - Advanced techniques

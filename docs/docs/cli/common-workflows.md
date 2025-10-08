# Common Workflows

Learn the most common scanning patterns and workflows for different use cases.

## Quick Network Scan

Discover hosts and identify services:

```bash
pentora scan --targets 192.168.1.0/24 --profile standard
```

## Vulnerability Assessment

Full scan with CVE checks:

```bash
pentora scan --targets critical-servers.txt --vuln --output report.json
```

## Discovery Only

Identify live hosts without port scanning:

```bash
pentora scan --targets 10.0.0.0/16 --only-discover
```

## Resume from Discovery

Scan previously discovered hosts:

```bash
# First, discover hosts
pentora scan --targets 10.0.0.0/16 --only-discover

# Then scan discovered hosts
pentora workspace show <scan-id> | jq -r '.discovered_hosts[].ip' > live-hosts.txt
pentora scan --target-file live-hosts.txt --no-discover
```

## Custom Workspace

Use non-default workspace location:

```bash
pentora scan --targets 192.168.1.100 --workspace-dir /data/scans
```

## Stateless Scan

No workspace persistence (ephemeral):

```bash
pentora scan --targets 192.168.1.100 --no-workspace --output results.json
```

## Remote Execution

Submit scan to remote server:

```bash
export PENTORA_SERVER=https://pentora.company.com
export PENTORA_API_TOKEN=your-token-here

pentora scan --targets 192.168.1.0/24 --server $PENTORA_SERVER
```

## Scheduled Scan (Server Mode)

Create recurring scan:

```bash
pentora scan --targets 192.168.1.0/24 \
  --schedule "0 2 * * *" \
  --profile standard \
  --notify slack://security-alerts
```

Requires server running.

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

# pentora workspace

Manage Pentora workspace and scan results.

## Synopsis

```bash
pentora workspace <subcommand> [flags]
```

## Description

The `workspace` command provides tools for managing scan storage, viewing results, and performing maintenance operations on the workspace directory.

## Subcommands

### info

Display workspace information.

```bash
pentora workspace info
```

**Output**:
```
Workspace: /home/user/.local/share/pentora
Size: 2.3 GB
Scans: 145
Oldest scan: 2023-09-01 (35 days ago)
Newest scan: 2023-10-06 (today)
Retention policy: 90 days
```

**Flags**:
- `--workspace-dir`: Specify workspace directory

### list

List all scans in workspace.

```bash
pentora workspace list
```

**Output**:
```
SCAN ID                     DATE                TARGETS  STATUS      SIZE
20231006-143022-a1b2c3     2023-10-06 14:30    192.168.1.0/24  completed   45 MB
20231005-120000-d4e5f6     2023-10-05 12:00    10.0.0.1        completed   12 MB
20231004-093000-g7h8i9     2023-10-04 09:30    example.com     failed      2 MB
```

**Flags**:
- `--format`: Output format (table, json, csv)
- `--filter`: Filter by status, tags, or date
- `--limit`: Limit number of results
- `--sort`: Sort by date, size, or status

**Examples**:
```bash
# List recent scans
pentora workspace list --limit 10

# List failed scans
pentora workspace list --filter status=failed

# JSON output
pentora workspace list --format json

# Sort by size
pentora workspace list --sort size --desc
```

### show

Display detailed scan information.

```bash
pentora workspace show <scan-id>
```

**Output**:
```
Scan ID: 20231006-143022-a1b2c3
Date: 2023-10-06 14:30:22
Duration: 5m 23s
Status: completed
Targets: 192.168.1.0/24
Profile: standard

Results:
  Live hosts: 15
  Open ports: 73
  Services identified: 68
  Vulnerabilities: 12 (3 critical, 5 high, 4 medium)

Location: /home/user/.local/share/pentora/scans/20231006-143022-a1b2c3/
```

**Flags**:
- `--format`: Output format (text, json)
- `--show-results`: Include full results in output

**Examples**:
```bash
# Show scan details
pentora workspace show 20231006-143022-a1b2c3

# JSON format
pentora workspace show 20231006-143022-a1b2c3 --format json

# Include full results
pentora workspace show 20231006-143022-a1b2c3 --show-results
```

### export

Export scan results.

```bash
pentora workspace export <scan-id> [flags]
```

**Flags**:
- `--output, -o`: Output file path
- `--format`: Export format (json, jsonl, csv, pdf)
- `--include-artifacts`: Include banners and screenshots

**Examples**:
```bash
# Export to JSON
pentora workspace export 20231006-143022-a1b2c3 --output scan.json

# Export to CSV
pentora workspace export 20231006-143022-a1b2c3 --format csv -o scan.csv

# Include artifacts
pentora workspace export 20231006-143022-a1b2c3 --include-artifacts -o scan-full.tar.gz
```

### delete

Delete scan from workspace.

```bash
pentora workspace delete <scan-id>
```

**Flags**:
- `--force`: Skip confirmation prompt

**Examples**:
```bash
# Delete with confirmation
pentora workspace delete 20231006-143022-a1b2c3

# Force delete
pentora workspace delete 20231006-143022-a1b2c3 --force
```

### gc

Garbage collection - clean up old scans.

```bash
pentora workspace gc [flags]
```

**Flags**:
- `--older-than`: Delete scans older than duration (e.g., 30d, 12w)
- `--keep-last`: Keep only N most recent scans
- `--dry-run`: Show what would be deleted without deleting
- `--force`: Skip confirmation

**Examples**:
```bash
# Delete scans older than 30 days
pentora workspace gc --older-than 30d

# Keep only last 100 scans
pentora workspace gc --keep-last 100

# Dry run
pentora workspace gc --older-than 60d --dry-run

# Apply both policies
pentora workspace gc --older-than 90d --keep-last 200

# Force cleanup
pentora workspace gc --older-than 7d --force
```

### clean-cache

Clear workspace cache.

```bash
pentora workspace clean-cache
```

Removes:
- Fingerprint catalog cache
- DNS resolution cache
- Temporary files

**Flags**:
- `--fingerprints`: Clean fingerprint cache only
- `--dns`: Clean DNS cache only
- `--force`: Skip confirmation

**Examples**:
```bash
# Clean all caches
pentora workspace clean-cache

# Clean fingerprint cache only
pentora workspace clean-cache --fingerprints

# Force clean
pentora workspace clean-cache --force
```

### init

Initialize new workspace.

```bash
pentora workspace init <path>
```

Creates workspace directory structure at specified path.

**Examples**:
```bash
# Initialize workspace
pentora workspace init /data/pentora-workspace

# Initialize with default location
pentora workspace init
```

### check

Verify workspace integrity.

```bash
pentora workspace check
```

Checks for:
- Corrupted scan data
- Missing required files
- Disk space issues
- Permission problems

**Flags**:
- `--fix`: Attempt to repair issues
- `--verbose`: Show detailed check results

**Examples**:
```bash
# Check workspace
pentora workspace check

# Check and fix
pentora workspace check --fix

# Verbose output
pentora workspace check --verbose
```

### migrate

Migrate workspace to new format version.

```bash
pentora workspace migrate
```

Upgrades workspace structure when Pentora version changes.

**Flags**:
- `--backup`: Create backup before migration
- `--force`: Skip confirmation

**Examples**:
```bash
# Migrate with backup
pentora workspace migrate --backup

# Force migration
pentora workspace migrate --force
```

## Global Flags

- `--workspace-dir`: Specify workspace directory
- `--config`: Config file path
- `--log-level`: Logging verbosity
- `--quiet`: Suppress output

## Examples

### View Workspace Status

```bash
pentora workspace info
```

### List Recent Scans

```bash
pentora workspace list --limit 20 --sort date
```

### Export Scan Results

```bash
pentora workspace show 20231006-143022-a1b2c3 --format json > scan.json
pentora workspace export 20231006-143022-a1b2c3 -o scan.csv --format csv
```

### Clean Up Old Scans

```bash
# Preview cleanup
pentora workspace gc --older-than 60d --dry-run

# Execute cleanup
pentora workspace gc --older-than 60d
```

### Manage Multiple Workspaces

```bash
# Client A workspace
pentora workspace info --workspace-dir /data/pentora/client-a
pentora workspace list --workspace-dir /data/pentora/client-a

# Client B workspace
pentora workspace info --workspace-dir /data/pentora/client-b
pentora workspace list --workspace-dir /data/pentora/client-b
```

### Automated Cleanup Script

```bash
#!/bin/bash
# cleanup-pentora.sh

# Clean old scans (keep last 90 days)
pentora workspace gc --older-than 90d --force

# Clean cache
pentora workspace clean-cache --force

# Check integrity
pentora workspace check
```

## See Also

- [Workspace Concept](/concepts/workspace) - Understanding workspace structure
- [CLI Overview](/cli/overview) - Command structure
- [Configuration](/configuration/workspace-config) - Workspace configuration

# Workspace

The workspace is Pentora's shared storage location for scan data, configuration, cache, and operational state. It provides persistent storage for scan results and enables coordination between CLI and server modes.

## What is a Workspace?

A workspace is a directory structure that stores:

- **Scan results**: Complete scan output with metadata
- **Queue**: Scheduled scans waiting for execution (server mode)
- **Cache**: Fingerprint databases, DNS lookups, temporary data
- **Reports**: Generated reports and exports
- **Audit logs**: User actions and system events (Enterprise)
- **Configuration**: Workspace-specific settings

## Default Locations

Pentora respects OS-specific conventions for application data:

### Linux
Follows XDG Base Directory Specification:
```
$XDG_DATA_HOME/pentora      (typically ~/.local/share/pentora)
```

### macOS
Uses Application Support directory:
```
~/Library/Application Support/Pentora
```

### Windows
Uses AppData directory:
```
%AppData%\Pentora           (e.g., C:\Users\username\AppData\Roaming\Pentora)
```

## Directory Structure

```
pentora/                      # Workspace root
├── scans/                    # Scan results (organized by ID)
│   ├── 20231006-143022-a1b2c3/
│   │   ├── request.json      # Scan parameters
│   │   ├── status.json       # Execution state
│   │   ├── results.jsonl     # Main results
│   │   └── artifacts/        # Additional data
│   │       ├── banners/      # Banner captures
│   │       ├── screenshots/  # Web screenshots
│   │       └── pcaps/        # Packet captures
│   └── 20231006-150000-d4e5f6/
│       └── ...
├── queue/                    # Scheduled scans (server mode)
│   ├── pending/              # Queued jobs
│   ├── running/              # Active jobs
│   └── failed/               # Failed job metadata
├── cache/                    # Cached data
│   ├── fingerprints/         # Fingerprint catalogs
│   │   ├── builtin.yaml
│   │   ├── remote-sync.yaml
│   │   └── custom.yaml
│   ├── dns/                  # DNS resolution cache
│   └── geo/                  # GeoIP database
├── reports/                  # Generated reports
│   ├── executive-2023-10.pdf
│   ├── vuln-summary.csv
│   └── compliance-pci.json
├── audit/                    # Audit logs (Enterprise)
│   ├── 2023-10-01.jsonl
│   └── 2023-10-02.jsonl
└── config/                   # Workspace-specific config
    ├── pentora.yaml          # Workspace configuration
    ├── license.key           # License JWT (Enterprise)
    └── profiles/             # Custom scan profiles
        ├── webapp.yaml
        └── infrastructure.yaml
```

## Scan Directory Structure

Each scan creates a unique directory identified by timestamp and ID:

```
scans/20231006-143022-a1b2c3/
├── request.json              # Original scan request
├── status.json               # Execution metadata
├── results.jsonl             # Line-delimited JSON results
├── artifacts/
│   ├── banners/              # Raw protocol banners
│   │   ├── 192.168.1.100-22.txt
│   │   └── 192.168.1.100-80.txt
│   ├── screenshots/          # Web screenshots (if enabled)
│   │   └── 192.168.1.100-80.png
│   └── pcaps/                # Network captures (if enabled)
│       └── 192.168.1.100.pcap
└── reports/
    ├── summary.json          # Scan summary
    ├── vulnerabilities.csv   # CVE list
    └── executive.pdf         # Executive report (Enterprise)
```

### request.json

Original scan parameters:

```json
{
  "scan_id": "20231006-143022-a1b2c3",
  "timestamp": "2023-10-06T14:30:22Z",
  "user": "operator",
  "targets": ["192.168.1.0/24"],
  "profile": "standard",
  "options": {
    "discover": true,
    "vuln": true,
    "rate": 1000,
    "ports": "1-1000"
  },
  "notifications": ["slack://security-alerts"]
}
```

### status.json

Execution state and timing:

```json
{
  "scan_id": "20231006-143022-a1b2c3",
  "state": "completed",
  "started_at": "2023-10-06T14:30:22Z",
  "completed_at": "2023-10-06T14:35:45Z",
  "duration_seconds": 323,
  "stages": {
    "target_ingestion": {"status": "completed", "duration": 0.5},
    "discovery": {"status": "completed", "duration": 12.3, "hosts_found": 15},
    "port_scan": {"status": "completed", "duration": 45.2, "ports_found": 73},
    "fingerprint": {"status": "completed", "duration": 89.5},
    "profile": {"status": "completed", "duration": 2.1},
    "vulnerability": {"status": "completed", "duration": 156.3, "vulns_found": 12},
    "reporting": {"status": "completed", "duration": 17.1}
  },
  "errors": [],
  "warnings": [
    "Rate limit exceeded, throttling to 500 pps"
  ]
}
```

### results.jsonl

Main scan results in line-delimited JSON format (one object per line):

```jsonl
{"timestamp":"2023-10-06T14:31:00Z","host":"192.168.1.100","state":"up","latency":1.2}
{"timestamp":"2023-10-06T14:31:15Z","host":"192.168.1.100","port":22,"protocol":"tcp","state":"open","service":{"name":"ssh","product":"OpenSSH","version":"8.2p1"}}
{"timestamp":"2023-10-06T14:31:16Z","host":"192.168.1.100","port":80,"protocol":"tcp","state":"open","service":{"name":"http","product":"nginx","version":"1.18.0"}}
{"timestamp":"2023-10-06T14:32:30Z","host":"192.168.1.100","port":80,"vulnerability":{"id":"CVE-2021-23017","severity":"critical","cvss":9.8}}
```

**Why JSONL?**
- Streamable: Process results line-by-line
- Append-friendly: Add results as scan progresses
- Grep-friendly: Search with standard tools
- Resilient: Corruption affects single line, not entire file

## Configuration

### Global Configuration

Control workspace behavior in `~/.config/pentora/config.yaml`:

```yaml
workspace:
  # Workspace root directory
  dir: ~/.local/share/pentora

  # Enable workspace persistence
  enabled: true

  # Automatically create directory if missing
  auto_create: true

  # Retention policies
  retention:
    enabled: true
    max_age: 90d              # Delete scans older than 90 days
    max_scans: 1000           # Keep at most 1000 scans
    min_free_space: 10GB      # Delete oldest when space low

  # Scan storage options
  scans:
    compress: false           # Gzip compress results
    keep_artifacts: true      # Store banners/screenshots
    keep_pcaps: false         # Store packet captures (large)

  # Cache settings
  cache:
    fingerprints:
      ttl: 7d                 # Refresh fingerprint catalog after 7 days
    dns:
      ttl: 24h
      max_entries: 10000
```

### Workspace-Specific Configuration

Override global settings per workspace in `<workspace>/config/pentora.yaml`:

```yaml
# This workspace uses custom settings
workspace:
  retention:
    max_age: 30d              # Shorter retention for this workspace

scanner:
  # Default profile for this workspace
  default_profile: webapp

notifications:
  # Workspace-specific notification channels
  default_channels:
    - slack://workspace-alerts
    - email://security-team@company.com
```

## CLI Control

### Specify Workspace Directory

Override the default workspace location:

```bash
# Use custom workspace
pentora scan --targets 192.168.1.100 --workspace-dir /data/pentora-workspace

# Use environment variable
export PENTORA_WORKSPACE=/data/pentora-workspace
pentora scan --targets 192.168.1.100
```

### Disable Workspace (Stateless Mode)

Run scans without persisting to workspace:

```bash
pentora scan --targets 192.168.1.100 --no-workspace
```

**Use cases for stateless mode**:
- Quick ad-hoc scans
- CI/CD pipelines (ephemeral environments)
- Minimal disk usage
- Privacy (no scan history)

Results still output to stdout/specified file but not saved to workspace.

### Workspace Management Commands

```bash
# Show workspace information
pentora workspace info

# List all scans
pentora workspace list

# Show specific scan
pentora workspace show <scan-id>

# Garbage collection (cleanup old scans)
pentora workspace gc

# Remove scans older than 30 days
pentora workspace gc --older-than 30d

# Keep only last 100 scans
pentora workspace gc --keep-last 100

# Clean cache
pentora workspace clean-cache

# Export scan
pentora workspace export <scan-id> --format json --output scan.json

# Delete specific scan
pentora workspace delete <scan-id>

# Initialize new workspace
pentora workspace init /path/to/new/workspace
```

## Retention Policies

Automatic cleanup based on configured policies:

### Age-Based Retention

Delete scans older than specified duration:

```yaml
workspace:
  retention:
    max_age: 90d              # 90 days
```

Supported units: `d` (days), `w` (weeks), `m` (months), `y` (years)

### Count-Based Retention

Keep only the N most recent scans:

```yaml
workspace:
  retention:
    max_scans: 1000           # Keep last 1000 scans
```

Oldest scans deleted first.

### Space-Based Retention

Delete oldest scans when disk space is low:

```yaml
workspace:
  retention:
    min_free_space: 10GB      # Maintain at least 10GB free
```

Monitors workspace filesystem and triggers cleanup when threshold reached.

### Manual Garbage Collection

Run cleanup on demand:

```bash
# Apply configured retention policies
pentora workspace gc

# Override with custom rules
pentora workspace gc --older-than 30d --keep-last 50

# Dry run (show what would be deleted)
pentora workspace gc --dry-run

# Force deletion without confirmation
pentora workspace gc --force
```

## Permissions

### File Permissions

Workspace directories and files are created with user-only access:

- Directories: `0700` (rwx------)
- Files: `0600` (rw-------)

This prevents other users on the system from reading scan results.

### Multi-User Scenarios

For shared workspaces:

1. **Use a dedicated user**:
   ```bash
   sudo useradd -r -s /bin/false pentora
   sudo mkdir /var/lib/pentora
   sudo chown pentora:pentora /var/lib/pentora
   ```

2. **Run server as dedicated user**:
   ```bash
   sudo -u pentora pentora server start --workspace-dir /var/lib/pentora
   ```

3. **Grant specific users access**:
   ```bash
   sudo usermod -a -G pentora alice
   sudo usermod -a -G pentora bob
   ```

### Enterprise Multi-Tenant

In Enterprise multi-tenant mode, workspace structure changes:

```
pentora/
├── tenants/
│   ├── customer-a/
│   │   ├── scans/
│   │   ├── reports/
│   │   └── config/
│   ├── customer-b/
│   │   └── ...
```

Tenant-level ACLs enforce isolation. See [Multi-Tenant Deployment](/enterprise/multi-tenant).

## Workspace Lifecycle

### Initialization

Workspace is automatically created on first use:

```bash
pentora scan --targets 192.168.1.100
# Creates ~/.local/share/pentora/ if missing
```

Explicit initialization:

```bash
pentora workspace init /path/to/workspace
```

### Migration

When upgrading Pentora, workspace format may change:

```bash
# Check if migration needed
pentora workspace check

# Perform migration
pentora workspace migrate

# Backup before migration
cp -r ~/.local/share/pentora ~/.local/share/pentora.backup
```

### Backup

Backup entire workspace:

```bash
# Filesystem backup
tar -czf pentora-backup-$(date +%Y%m%d).tar.gz ~/.local/share/pentora

# Incremental backup with rsync
rsync -av ~/.local/share/pentora /mnt/backups/pentora/
```

Export individual scans:

```bash
pentora workspace export <scan-id> --output scan-backup.json
```

### Restoration

Restore from backup:

```bash
# Stop server if running
pentora server stop

# Restore workspace
tar -xzf pentora-backup-20231006.tar.gz -C ~/

# Verify integrity
pentora workspace check

# Restart server
pentora server start
```

## Performance Considerations

### Disk Space

Monitor workspace disk usage:

```bash
# Show workspace size
du -sh ~/.local/share/pentora

# Show per-directory breakdown
du -h --max-depth=1 ~/.local/share/pentora
```

**Space usage estimates**:
- Small scan (10 hosts, 100 ports): ~1-5 MB
- Medium scan (100 hosts, 1000 ports): ~10-50 MB
- Large scan (1000 hosts, 1000 ports): ~100-500 MB
- With packet captures: 10-100x larger

### Compression

Enable compression to save space:

```yaml
workspace:
  scans:
    compress: true            # Gzip compress results.jsonl
```

Reduces storage by 70-90% for text-based results. Transparent decompression when reading.

### Database Backend (Enterprise)

For very large deployments, use database storage instead of filesystem:

```yaml
workspace:
  backend: postgresql
  database:
    host: localhost
    port: 5432
    database: pentora
    user: pentora
    password: ${PENTORA_DB_PASSWORD}
```

Benefits:
- Better performance with millions of scans
- Transactional integrity
- Easier replication and backup
- Advanced querying

## Server Mode Integration

In server mode, workspace serves additional roles:

### Job Queue

Scheduled scans stored in `queue/`:

```
queue/
├── pending/
│   ├── job-001.json
│   └── job-002.json
├── running/
│   └── job-001.json
└── failed/
    └── job-003.json
```

Worker processes poll `queue/pending/`, execute scans, and write results to `scans/`.

### Distributed Scanning (Enterprise)

Workers on multiple hosts share a centralized workspace (via NFS, object storage, or database):

```
[CLI] → [Queue] → [Worker Pool] → [Shared Workspace]
                      ↓
                  [Worker 1]
                  [Worker 2]
                  [Worker 3]
```

See [Distributed Scanning](/enterprise/distributed-scanning) for architecture.

## Troubleshooting

### Workspace Permission Errors

```
Error: Failed to create workspace directory: permission denied
```

**Solution**: Ensure you have write permissions:

```bash
mkdir -p ~/.local/share/pentora
chmod 700 ~/.local/share/pentora
```

### Disk Space Exhausted

```
Error: Failed to write results: no space left on device
```

**Solutions**:
1. Run garbage collection: `pentora workspace gc --older-than 7d`
2. Enable compression: Set `workspace.scans.compress: true`
3. Reduce artifact retention: Set `workspace.scans.keep_artifacts: false`
4. Use different workspace location with more space

### Corrupted Scan Data

If a scan directory is incomplete or corrupted:

```bash
# Check scan status
pentora workspace show <scan-id>

# Delete corrupted scan
pentora workspace delete <scan-id>

# Re-run scan
pentora scan --targets <original-targets>
```

### Workspace Migration Failed

If migration fails:

```bash
# Restore from backup
mv ~/.local/share/pentora ~/.local/share/pentora.failed
tar -xzf pentora-backup.tar.gz -C ~/

# Report issue with logs
pentora workspace check --verbose > migration-error.log
```

## Next Steps

- [Scan Pipeline](/concepts/scan-pipeline) - How scan data is generated
- [Server Mode Deployment](/deployment/server-mode) - Running Pentora as a service
- [Workspace Configuration](/configuration/workspace-config) - Detailed configuration
- [Distributed Scanning](/enterprise/distributed-scanning) - Multi-host workspace sharing

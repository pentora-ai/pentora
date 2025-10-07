# Workspace Configuration

Configure workspace behavior, retention policies, and storage options.

## Workspace Settings

```yaml
workspace:
  # Root directory
  dir: ~/.local/share/pentora
  
  # Enable workspace persistence
  enabled: true
  
  # Auto-create if missing
  auto_create: true
  
  # Retention policies
  retention:
    enabled: true
    max_age: 90d              # Delete scans older than 90 days
    max_scans: 1000           # Keep at most 1000 scans
    min_free_space: 10GB      # Delete oldest when space low
  
  # Scan storage
  scans:
    compress: false           # Gzip compress results
    keep_artifacts: true      # Store banners/screenshots
    keep_pcaps: false         # Store packet captures
  
  # Cache settings
  cache:
    fingerprints:
      ttl: 7d                 # Refresh fingerprint catalog after 7 days
    dns:
      ttl: 24h
      max_entries: 10000
```

## Platform-Specific Defaults

### Linux
```yaml
workspace:
  dir: ${XDG_DATA_HOME}/pentora  # ~/.local/share/pentora
```

### macOS
```yaml
workspace:
  dir: ~/Library/Application Support/Pentora
```

### Windows
```yaml
workspace:
  dir: ${APPDATA}\Pentora  # C:\Users\username\AppData\Roaming\Pentora
```

## Retention Policies

### Age-Based
```yaml
workspace:
  retention:
    max_age: 90d  # Supported: d (days), w (weeks), m (months), y (years)
```

### Count-Based
```yaml
workspace:
  retention:
    max_scans: 1000  # Keep only 1000 most recent scans
```

### Space-Based
```yaml
workspace:
  retention:
    min_free_space: 10GB  # Maintain at least 10GB free space
```

## CLI Overrides

```bash
# Custom workspace directory
pentora scan --targets 192.168.1.100 --workspace-dir /data/pentora

# Disable workspace
pentora scan --targets 192.168.1.100 --no-workspace

# Environment variable
export PENTORA_WORKSPACE=/data/pentora
pentora scan --targets 192.168.1.100
```

See [Workspace Concept](/docs/concepts/workspace) for detailed structure.

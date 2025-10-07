# Logging Configuration

Configure logging output, verbosity, and format for Pentora.

## Log Levels

- `debug`: Detailed debugging information
- `info`: General informational messages (default)
- `warn`: Warning messages
- `error`: Error messages only

## Configuration

```yaml
logging:
  # Log level
  level: info
  
  # Output format (text, json)
  format: text
  
  # Output destination (stdout, stderr, file)
  output: stderr
  
  # File logging
  file:
    enabled: false
    path: /var/log/pentora/pentora.log
    max_size: 100MB
    max_backups: 10
    max_age: 30d
    compress: true
  
  # Structured fields
  fields:
    component: true     # Log component name
    caller: false       # Log caller info (file:line)
    timestamp: true     # Log timestamp
```

## CLI Flags

```bash
# Set log level
pentora scan --targets 192.168.1.100 --log-level debug

# JSON format
pentora scan --targets 192.168.1.100 --log-format json

# Verbosity shortcuts
pentora scan --targets 192.168.1.100 -v    # verbose (debug)
pentora scan --targets 192.168.1.100 -vv   # very verbose (trace)

# Quiet mode
pentora scan --targets 192.168.1.100 --quiet
```

## Log Examples

### Text Format
```
2023-10-06T14:30:22Z INFO  Scan started scan_id=20231006-143022-a1b2c3
2023-10-06T14:30:25Z INFO  Discovery completed hosts_found=15
2023-10-06T14:31:30Z INFO  Port scanning ports_found=73
2023-10-06T14:33:45Z INFO  Scan completed duration=3m23s
```

### JSON Format
```json
{"level":"info","timestamp":"2023-10-06T14:30:22Z","component":"orchestrator","scan_id":"20231006-143022-a1b2c3","message":"Scan started"}
{"level":"info","timestamp":"2023-10-06T14:30:25Z","component":"discovery","hosts_found":15,"message":"Discovery completed"}
```

## Environment Variables

```bash
PENTORA_LOG_LEVEL=debug
PENTORA_LOG_FORMAT=json
```

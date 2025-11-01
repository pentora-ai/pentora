# pentora server

Control Pentora server daemon for centralized scan orchestration.

## Synopsis

```bash
pentora server <subcommand> [flags]
```

## Description

The `server` command manages the Pentora server daemon, which provides:
- REST API for scan submission
- Job queue and scheduler
- Worker pools for distributed scanning (Enterprise)
- Web UI hosting
- Multi-tenant storage (Enterprise)

## Subcommands

### start

Start Pentora server.

```bash
pentora server start [flags]
```

**Flags**:
- `--bind`: Address to bind (default: `0.0.0.0:8080`)
- `--workers`: Number of worker threads (default: CPU cores)
- `--daemon, -d`: Run as background daemon
- `--pid-file`: PID file location

**Examples**:
```bash
# Start server on default port
pentora server start

# Custom bind address
pentora server start --bind 127.0.0.1:9090

# Run as daemon
pentora server start --daemon --pid-file /var/run/pentora.pid

# With custom workers
pentora server start --workers 8
```

### stop

Stop running Pentora server.

```bash
pentora server stop
```

**Flags**:
- `--force`: Force shutdown (don't wait for running scans)
- `--timeout`: Graceful shutdown timeout (default: 30s)

**Examples**:
```bash
# Graceful stop
pentora server stop

# Force stop
pentora server stop --force

# Custom timeout
pentora server stop --timeout 60s
```

### status

Check server status.

```bash
pentora server status
```

**Output**:
```
Pentora Server Status
---------------------
Status: running
Uptime: 5 days, 3 hours
PID: 12345
Bind: 0.0.0.0:8080
Workers: 4
Active scans: 2
Queued jobs: 5
Total scans: 1,234
```

**Flags**:
- `--format`: Output format (text, json)

**Examples**:
```bash
# Text status
pentora server status

# JSON status
pentora server status --format json
```

### restart

Restart server (stop then start).

```bash
pentora server restart
```

**Flags**:
- `--force`: Force restart without graceful shutdown

**Examples**:
```bash
# Graceful restart
pentora server restart

# Force restart
pentora server restart --force
```

### logs

Display server logs.

```bash
pentora server logs [flags]
```

**Flags**:
- `--follow, -f`: Follow log output
- `--tail`: Show last N lines (default: 100)
- `--since`: Show logs since time (e.g., `1h`, `2023-10-06`)
- `--level`: Filter by log level

**Examples**:
```bash
# Show last 100 lines
pentora server logs

# Follow logs
pentora server logs --follow

# Last 1000 lines
pentora server logs --tail 1000

# Since 1 hour ago
pentora server logs --since 1h

# Errors only
pentora server logs --level error
```

### reload

Reload server configuration without restart.

```bash
pentora server reload
```

Reloads:
- Configuration files
- Scan profiles
- Notification channels

Does not reload:
- License keys (requires restart)
- Server bind address (requires restart)

**Examples**:
```bash
# Reload config
pentora server reload
```

## Configuration

Server configuration via YAML:

```yaml
# ~/.config/pentora/config.yaml
server:
  bind: 0.0.0.0:8080
  workers: 4
  api:
    enabled: true
    auth: true
    rate_limit: 100  # requests per minute
  ui:
    enabled: true
    path: /ui
    static_dir: /usr/share/pentora/ui
  tls:
    enabled: false
    cert_file: /etc/pentora/tls/cert.pem
    key_file: /etc/pentora/tls/key.pem
  cors:
    enabled: true
    origins: ["https://pentora.company.com"]
  queue:
    max_jobs: 1000
    retention: 7d
  workers:
    min: 2
    max: 10
    auto_scale: true
```

## Systemd Service

### Service File

`/etc/systemd/system/pentora.service`:

```ini
[Unit]
Description=Pentora Security Scanner Server
After=network.target

[Service]
Type=simple
User=pentora
Group=pentora
WorkingDirectory=/var/lib/pentora
ExecStart=/usr/local/bin/pentora server start --bind 0.0.0.0:8080
ExecStop=/usr/local/bin/pentora server stop
Restart=on-failure
RestartSec=5s

# Security
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/pentora /var/log/pentora

# Resources
LimitNOFILE=65536
MemoryMax=4G

[Install]
WantedBy=multi-user.target
```

### Systemd Commands

```bash
# Enable service
sudo systemctl enable pentora

# Start service
sudo systemctl start pentora

# Check status
sudo systemctl status pentora

# View logs
sudo journalctl -u pentora -f

# Restart service
sudo systemctl restart pentora

# Stop service
sudo systemctl stop pentora
```

## API Endpoints

Server exposes REST API at `/api/v1`:

### Scans

- `POST /api/v1/scans` - Submit new scan
- `GET /api/v1/scans` - List scans
- `GET /api/v1/scans/{id}` - Get scan details
- `DELETE /api/v1/scans/{id}` - Delete scan

### Jobs

- `POST /api/v1/jobs` - Submit job (Enterprise)
- `GET /api/v1/jobs` - List jobs
- `GET /api/v1/jobs/{id}` - Get job status

### System

- `GET /api/v1/health` - Health check
- `GET /api/v1/version` - Version info
- `GET /api/v1/license` - License status (Enterprise)

See [REST API Documentation](/api/rest/scans) for details.

## TLS Configuration

Enable HTTPS:

```yaml
server:
  tls:
    enabled: true
    cert_file: /etc/pentora/tls/cert.pem
    key_file: /etc/pentora/tls/key.pem
    # Optional: Client certificate authentication
    client_auth: false
    client_ca_file: /etc/pentora/tls/ca.pem
```

Generate self-signed certificate:

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
```

## Authentication

### API Tokens

Generate API token:

```bash
pentora server token create --name "CI Pipeline" --scopes scan:read,scan:write
```

Use token:

```bash
curl -H "Authorization: Bearer <token>" https://pentora.company.com/api/v1/scans
```

### SSO Integration (Enterprise)

Configure OIDC/SAML:

```yaml
server:
  auth:
    provider: oidc
    oidc:
      issuer: https://auth.company.com
      client_id: pentora
      client_secret: ${OIDC_SECRET}
      redirect_url: https://pentora.company.com/auth/callback
```

## Monitoring

### Health Checks

```bash
# HTTP health check
curl http://localhost:8080/health

# Detailed status
curl http://localhost:8080/api/v1/health
```

Response:
```json
{
  "status": "healthy",
  "uptime": 432000,
  "version": "1.0.0",
  "workers": {
    "active": 2,
    "idle": 2,
    "total": 4
  },
  "queue": {
    "pending": 5,
    "running": 2,
    "failed": 0
  }
}
```

### Metrics (Enterprise)

Prometheus metrics at `/metrics`:

```bash
curl http://localhost:8080/metrics
```

Metrics include:
- `pentora_scans_total` - Total scans
- `pentora_scan_duration_seconds` - Scan duration histogram
- `pentora_queue_length` - Queue length gauge
- `pentora_worker_utilization` - Worker utilization

## Troubleshooting

### Server Won't Start

Check logs:
```bash
pentora server start --log-level debug
```

Common issues:
- Port already in use: Change `--bind` address
- Permission denied: Run with sufficient privileges or `sudo`
- Config error: Validate config with `pentora config validate`

### High Memory Usage

Reduce workers:
```yaml
server:
  workers: 2
```

Enable memory limits:
```yaml
engine:
  max_memory: 2GB
```

### Slow Response Times

Increase workers:
```yaml
server:
  workers: 8
```

Enable caching:
```yaml
server:
  cache:
    enabled: true
    ttl: 5m
```

## See Also

- [Server Deployment](/deployment/server-mode) - Deployment guide
- [REST API](/api/rest/scans) - API reference
- [Configuration](/configuration/overview) - Server configuration

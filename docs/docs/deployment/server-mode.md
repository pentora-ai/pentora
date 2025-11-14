# Server Mode Deployment

Deploy Vulntor as a persistent server daemon for centralized scan orchestration, API access, and scheduled scanning.

## Overview

Server mode deployment is ideal for:

- Centralized security scanning infrastructure
- REST API access for integrations
- Scheduled and recurring scans
- Web UI access (Enterprise)
- Multi-user environments
- Distributed scanning (Enterprise)

Server mode provides:

- REST API for scan submission and management
- Job queue and scheduler
- Worker pools for concurrent scanning
- Web portal for scan management (Enterprise)
- Multi-tenant storage isolation (Enterprise)

## Prerequisites

### System Requirements

#### Minimum

- **CPU**: 2 cores
- **RAM**: 4 GB
- **Disk**: 20 GB (including storage)
- **OS**: Linux (Ubuntu 20.04+, RHEL/CentOS 8+, Debian 11+)

#### Recommended for Production

- **CPU**: 4+ cores
- **RAM**: 8+ GB
- **Disk**: 100+ GB SSD
- **OS**: Linux with systemd
- **Network**: Static IP or DNS name

### Software Dependencies

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y curl systemd

# RHEL/CentOS/Fedora
sudo yum install -y curl systemd
```

## Installation

### Quick Server Setup

```bash
# Download and install Vulntor
curl -sSL https://vulntor.io/install.sh | bash

# Verify installation
vulntor version

# Test server mode
vulntor server start --bind 127.0.0.1:8080
```

### Dedicated User Setup

Create dedicated user for security:

```bash
# Create vulntor user
sudo useradd -r -s /bin/false -d /var/lib/vulntor vulntor

# Create directories
sudo mkdir -p /var/lib/vulntor
sudo mkdir -p /var/log/vulntor
sudo mkdir -p /etc/vulntor

# Set permissions
sudo chown -R vulntor:vulntor /var/lib/vulntor
sudo chown -R vulntor:vulntor /var/log/vulntor
sudo chown -R vulntor:vulntor /etc/vulntor
```

### Configuration

Create server configuration at `/etc/vulntor/config.yaml`:

```yaml
storage:
  dir: /var/lib/vulntor/storage
  enabled: true
  retention:
    enabled: true
    max_age: 90d
    max_scans: 5000
    min_free_space: 20GB
  scans:
    compress: true
    keep_artifacts: true

server:
  bind: 0.0.0.0:8080
  workers: 4
  api:
    enabled: true
    auth: true
    rate_limit: 100  # requests per minute
  ui:
    enabled: false  # Set true for Enterprise
    path: /ui
    static_dir: /usr/share/vulntor/ui
  tls:
    enabled: false
    cert_file: /etc/vulntor/tls/cert.pem
    key_file: /etc/vulntor/tls/key.pem
  cors:
    enabled: true
    origins: ["https://vulntor.company.com"]
  queue:
    max_jobs: 1000
    retention: 7d
  workers:
    min: 2
    max: 10
    auto_scale: true

logging:
  level: info
  format: json
  output: file
  file:
    enabled: true
    path: /var/log/vulntor/vulntor.log
    max_size: 100MB
    max_backups: 10
    max_age: 30d

scanner:
  default_profile: standard
  rate: 1000
  concurrency: 100
  timeout: 3s

fingerprint:
  cache:
    auto_sync: true
    ttl: 7d

notifications:
  default_channels: []
  slack:
    webhook_url: "${SLACK_WEBHOOK_URL}"
    channel: "#security-alerts"
  email:
    smtp_server: "smtp.company.com"
    smtp_port: 587
    from: "vulntor@company.com"
    to: ["security@company.com"]
```

Set file permissions:

```bash
sudo chmod 600 /etc/vulntor/config.yaml
sudo chown vulntor:vulntor /etc/vulntor/config.yaml
```

## Systemd Service Setup

### Create Service File

Create `/etc/systemd/system/vulntor.service`:

```ini
[Unit]
Description=Vulntor Security Scanner Server
Documentation=https://docs.vulntor.io
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=vulntor
Group=vulntor

# Working directory
WorkingDirectory=/var/lib/vulntor

# Environment
Environment="VULNTOR_CONFIG=/etc/vulntor/config.yaml"
Environment="VULNTOR_STORAGE_DIR=/var/lib/vulntor/storage"

# Start command
ExecStart=/usr/local/bin/vulntor server start --config /etc/vulntor/config.yaml

# Stop command
ExecStop=/usr/local/bin/vulntor server stop --timeout 30s

# Reload command
ExecReload=/usr/local/bin/vulntor server reload

# Restart policy
Restart=on-failure
RestartSec=5s

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/vulntor /var/log/vulntor
CapabilityBoundingSet=CAP_NET_RAW CAP_NET_ADMIN CAP_NET_BIND_SERVICE

# Resource limits
LimitNOFILE=65536
MemoryMax=8G
CPUQuota=400%

# Process management
TimeoutStartSec=30s
TimeoutStopSec=30s

[Install]
WantedBy=multi-user.target
```

### Set Capabilities

Allow Vulntor to perform privileged network operations:

```bash
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/vulntor
```

### Enable and Start Service

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable service (start on boot)
sudo systemctl enable vulntor

# Start service
sudo systemctl start vulntor

# Check status
sudo systemctl status vulntor

# View logs
sudo journalctl -u vulntor -f
```

### Service Management

```bash
# Start service
sudo systemctl start vulntor

# Stop service
sudo systemctl stop vulntor

# Restart service
sudo systemctl restart vulntor

# Reload configuration (no downtime)
sudo systemctl reload vulntor

# Check status
sudo systemctl status vulntor

# Enable on boot
sudo systemctl enable vulntor

# Disable from boot
sudo systemctl disable vulntor
```

## TLS/SSL Configuration

### Generate Self-Signed Certificate

For development/testing:

```bash
# Create TLS directory
sudo mkdir -p /etc/vulntor/tls
cd /etc/vulntor/tls

# Generate certificate
sudo openssl req -x509 -newkey rsa:4096 \
  -keyout key.pem \
  -out cert.pem \
  -days 365 \
  -nodes \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=vulntor.company.com"

# Set permissions
sudo chown vulntor:vulntor /etc/vulntor/tls/*.pem
sudo chmod 600 /etc/vulntor/tls/*.pem
```

### Use Let's Encrypt Certificate

For production:

```bash
# Install certbot
sudo apt install -y certbot

# Obtain certificate
sudo certbot certonly --standalone \
  -d vulntor.company.com \
  --email admin@company.com \
  --agree-tos

# Certificates will be at:
# /etc/letsencrypt/live/vulntor.company.com/fullchain.pem
# /etc/letsencrypt/live/vulntor.company.com/privkey.pem
```

Update `/etc/vulntor/config.yaml`:

```yaml
server:
  bind: 0.0.0.0:443
  tls:
    enabled: true
    cert_file: /etc/letsencrypt/live/vulntor.company.com/fullchain.pem
    key_file: /etc/letsencrypt/live/vulntor.company.com/privkey.pem
```

Allow certbot to access certificates:

```bash
# Add vulntor user to cert group
sudo usermod -a -G ssl-cert vulntor

# Set permissions
sudo chmod 640 /etc/letsencrypt/live/vulntor.company.com/*.pem
sudo chgrp ssl-cert /etc/letsencrypt/live/vulntor.company.com/*.pem
```

Restart service:

```bash
sudo systemctl restart vulntor
```

### Auto-Renewal Setup

```bash
# Create renewal hook
sudo tee /etc/letsencrypt/renewal-hooks/post/vulntor-reload.sh <<EOF
#!/bin/bash
systemctl reload vulntor
EOF

sudo chmod +x /etc/letsencrypt/renewal-hooks/post/vulntor-reload.sh

# Test renewal
sudo certbot renew --dry-run
```

## API Authentication

### Generate API Token

```bash
# Generate token
sudo -u vulntor vulntor server token create \
  --name "CI Pipeline" \
  --scopes scan:read,scan:write \
  --expiry 365d

# Example output:
# Token: vulntor_eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
# Save this token securely - it cannot be retrieved again
```

### Use API Token

```bash
# Set token as environment variable
export VULNTOR_API_TOKEN=vulntor_eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

# Make API request
curl -H "Authorization: Bearer $VULNTOR_API_TOKEN" \
  https://vulntor.company.com/api/v1/scans
```

### Token Management

```bash
# List tokens
vulntor server token list

# Revoke token
vulntor server token revoke <token-id>

# Rotate token
vulntor server token rotate <token-id>
```

## Reverse Proxy Configuration

### Nginx

Create `/etc/nginx/sites-available/vulntor`:

```nginx
upstream vulntor {
    server 127.0.0.1:8080;
}

server {
    listen 80;
    server_name vulntor.company.com;

    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name vulntor.company.com;

    # SSL configuration
    ssl_certificate /etc/letsencrypt/live/vulntor.company.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/vulntor.company.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Logging
    access_log /var/log/nginx/vulntor-access.log;
    error_log /var/log/nginx/vulntor-error.log;

    # Timeouts for long-running scans
    proxy_read_timeout 300s;
    proxy_connect_timeout 75s;

    location / {
        proxy_pass http://vulntor;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    # API rate limiting
    location /api/ {
        limit_req zone=api burst=20 nodelay;
        proxy_pass http://vulntor;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Health check endpoint
    location /health {
        proxy_pass http://vulntor;
        access_log off;
    }
}

# Rate limiting zone
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
```

Enable and restart:

```bash
sudo ln -s /etc/nginx/sites-available/vulntor /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

### Apache

Create `/etc/apache2/sites-available/vulntor.conf`:

```apache
<VirtualHost *:80>
    ServerName vulntor.company.com
    Redirect permanent / https://vulntor.company.com/
</VirtualHost>

<VirtualHost *:443>
    ServerName vulntor.company.com

    SSLEngine on
    SSLCertificateFile /etc/letsencrypt/live/vulntor.company.com/fullchain.pem
    SSLCertificateKeyFile /etc/letsencrypt/live/vulntor.company.com/privkey.pem

    # Logging
    ErrorLog ${APACHE_LOG_DIR}/vulntor-error.log
    CustomLog ${APACHE_LOG_DIR}/vulntor-access.log combined

    # Proxy configuration
    ProxyPreserveHost On
    ProxyTimeout 300

    ProxyPass / http://127.0.0.1:8080/
    ProxyPassReverse / http://127.0.0.1:8080/

    # WebSocket support
    RewriteEngine On
    RewriteCond %{HTTP:Upgrade} =websocket [NC]
    RewriteRule /(.*)           ws://127.0.0.1:8080/$1 [P,L]

    <Location />
        Require all granted
    </Location>
</VirtualHost>
```

Enable modules and site:

```bash
sudo a2enmod proxy proxy_http proxy_wstunnel ssl rewrite
sudo a2ensite vulntor
sudo apache2ctl configtest
sudo systemctl restart apache2
```

## Monitoring and Health Checks

### Health Check Endpoint

```bash
# Simple health check
curl http://localhost:8080/health

# Detailed health status
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
  },
  "storage": {
    "scans": 145,
    "size_mb": 2340,
    "free_space_mb": 87650
  }
}
```

### Systemd Watchdog

Add to `/etc/systemd/system/vulntor.service`:

```ini
[Service]
WatchdogSec=60s
```

### Monitoring with Prometheus (Enterprise)

Vulntor exposes Prometheus metrics at `/metrics`:

```bash
curl http://localhost:8080/metrics
```

Example metrics:

```
# HELP vulntor_scans_total Total number of scans
# TYPE vulntor_scans_total counter
vulntor_scans_total 1234

# HELP vulntor_scan_duration_seconds Scan duration histogram
# TYPE vulntor_scan_duration_seconds histogram
vulntor_scan_duration_seconds_bucket{le="60"} 450
vulntor_scan_duration_seconds_bucket{le="300"} 890
vulntor_scan_duration_seconds_bucket{le="900"} 1200

# HELP vulntor_queue_length Current queue length
# TYPE vulntor_queue_length gauge
vulntor_queue_length 5

# HELP vulntor_worker_utilization Worker utilization percentage
# TYPE vulntor_worker_utilization gauge
vulntor_worker_utilization 0.75
```

Configure Prometheus (`/etc/prometheus/prometheus.yml`):

```yaml
scrape_configs:
  - job_name: 'vulntor'
    scrape_interval: 30s
    static_configs:
      - targets: ['localhost:8080']
```

### Log Monitoring

```bash
# View live logs
sudo journalctl -u vulntor -f

# View logs since boot
sudo journalctl -u vulntor -b

# View last 100 lines
sudo journalctl -u vulntor -n 100

# View errors only
sudo journalctl -u vulntor -p err

# View logs for specific time
sudo journalctl -u vulntor --since "2024-10-06 10:00" --until "2024-10-06 11:00"

# Export logs
sudo journalctl -u vulntor > vulntor-logs.txt
```

### Alerting

Create `/etc/vulntor/alerts.yaml`:

```yaml
alerts:
  - name: high_error_rate
    condition: error_rate > 0.1
    action: slack
    message: 'Vulntor error rate exceeded threshold'

  - name: queue_backlog
    condition: queue_length > 100
    action: email
    message: 'Scan queue backlog detected'

  - name: disk_space_low
    condition: free_space_mb < 10000
    action: slack,email
    message: 'Storage disk space low'
```

## Backup and Recovery

### Backup Storage

```bash
# Create backup script
sudo tee /usr/local/bin/vulntor-backup.sh <<'EOF'
#!/bin/bash
set -euo pipefail

BACKUP_DIR="/var/backups/vulntor"
DATE=$(date +%Y%m%d-%H%M%S)
STORAGE_DIR="/var/lib/vulntor/storage"
CONFIG_DIR="/etc/vulntor"

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Backup storage
tar -czf "$BACKUP_DIR/storage-$DATE.tar.gz" -C "$(dirname "$STORAGE_DIR")" "$(basename "$STORAGE_DIR")"

# Backup configuration
tar -czf "$BACKUP_DIR/config-$DATE.tar.gz" "$CONFIG_DIR"

# Remove backups older than 30 days
find "$BACKUP_DIR" -name "*.tar.gz" -mtime +30 -delete

echo "Backup completed: $BACKUP_DIR"
EOF

sudo chmod +x /usr/local/bin/vulntor-backup.sh
```

Schedule daily backup:

```bash
# Add cron job
sudo crontab -e

# Add line:
0 3 * * * /usr/local/bin/vulntor-backup.sh
```

### Restore from Backup

```bash
# Stop service
sudo systemctl stop vulntor

# Restore storage
sudo tar -xzf /var/backups/vulntor/storage-20241006-030000.tar.gz -C /var/lib/vulntor/

# Restore configuration
sudo tar -xzf /var/backups/vulntor/config-20241006-030000.tar.gz -C /

# Fix permissions
sudo chown -R vulntor:vulntor /var/lib/vulntor
sudo chown -R vulntor:vulntor /etc/vulntor

# Start service
sudo systemctl start vulntor
```

## High Availability Setup

### Load Balancer Configuration

Deploy multiple Vulntor servers behind load balancer:

```
           ┌─────────────┐
           │Load Balancer│
           └──────┬──────┘
                  │
       ┌──────────┼──────────┐
       │          │          │
   ┌───▼───┐  ┌───▼───┐  ┌───▼───┐
   │Server1│  │Server2│  │Server3│
   └───┬───┘  └───┬───┘  └───┬───┘
       │          │          │
       └──────────┼──────────┘
                  │
           ┌──────▼──────┐
           │Shared Storage│
           └─────────────┘
```

### Shared Storage Setup

Use NFS for shared storage:

```bash
# On NFS server
sudo apt install nfs-kernel-server
sudo mkdir -p /export/vulntor-storage
sudo chown -R vulntor:vulntor /export/vulntor-storage

# Add to /etc/exports
echo "/export/vulntor-storage 192.168.1.0/24(rw,sync,no_subtree_check)" | sudo tee -a /etc/exports
sudo exportfs -ra

# On Vulntor servers
sudo apt install nfs-common
sudo mount -t nfs nfs-server:/export/vulntor-storage /var/lib/vulntor/storage
```

Add to `/etc/fstab`:

```
nfs-server:/export/vulntor-storage /var/lib/vulntor/storage nfs defaults 0 0
```

## Upgrading

### Backup Before Upgrade

```bash
# Backup storage and config
/usr/local/bin/vulntor-backup.sh

# Note current version
vulntor version > /tmp/vulntor-version-pre-upgrade.txt
```

### Upgrade Process

```bash
# Stop service
sudo systemctl stop vulntor

# Download new version
curl -LO https://github.com/vulntor-ai/vulntor/releases/latest/download/vulntor-linux-amd64.tar.gz

# Extract and install
tar -xzf vulntor-linux-amd64.tar.gz
sudo mv vulntor /usr/local/bin/vulntor
sudo chmod +x /usr/local/bin/vulntor

# Set capabilities
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/vulntor

# Start service
sudo systemctl start vulntor

# Verify
vulntor version
sudo systemctl status vulntor
```

### Rollback

```bash
# Stop service
sudo systemctl stop vulntor

# Restore previous binary
sudo cp /var/backups/vulntor/vulntor-backup /usr/local/bin/vulntor

# Start service
sudo systemctl start vulntor
```

## Troubleshooting

### Service Won't Start

```bash
# Check logs
sudo journalctl -u vulntor -n 50

# Test configuration
vulntor config validate --config /etc/vulntor/config.yaml

# Check port availability
sudo netstat -tlnp | grep 8080

# Check permissions
ls -l /usr/local/bin/vulntor
ls -l /etc/vulntor/config.yaml
```

### High Memory Usage

```bash
# Check memory usage
ps aux | grep vulntor

# Reduce workers in config
server:
  workers: 2

# Set memory limit in systemd
MemoryMax=4G
```

### Port Already in Use

```bash
# Find process using port
sudo lsof -i :8080

# Change bind address
server:
  bind: 0.0.0.0:9090
```

### Permission Errors

```bash
# Check capabilities
getcap /usr/local/bin/vulntor

# Set capabilities
sudo setcap cap_net_raw,cap_net_admin,cap_net_bind_service+eip /usr/local/bin/vulntor

# Fix file permissions
sudo chown -R vulntor:vulntor /var/lib/vulntor
sudo chown -R vulntor:vulntor /var/log/vulntor
sudo chown vulntor:vulntor /etc/vulntor/config.yaml
```

## Security Hardening

### Firewall Configuration

```bash
# UFW (Ubuntu)
sudo ufw allow 8080/tcp
sudo ufw enable

# Firewalld (RHEL/CentOS)
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --reload

# iptables
sudo iptables -A INPUT -p tcp --dport 8080 -j ACCEPT
```

### SELinux Configuration (RHEL/CentOS)

```bash
# Set SELinux context
sudo semanage fcontext -a -t bin_t /usr/local/bin/vulntor
sudo restorecon -v /usr/local/bin/vulntor

# Allow network binding
sudo setsebool -P vulntor_can_network_connect 1
```

### Audit Logging

Enable audit logging in `/etc/vulntor/config.yaml`:

```yaml
logging:
  audit:
    enabled: true
    file: /var/log/vulntor/audit.log
    events:
      - api_access
      - scan_start
      - scan_complete
      - config_change
      - user_login
```

## Next Steps

- [Docker Deployment](/deployment/docker) - Containerized deployment
- [Air-Gapped Deployment](/deployment/air-gapped) - Offline environments
- [REST API Reference](/api/rest/scans) - API documentation
- [Enterprise Features](/enterprise/overview) - Advanced capabilities
- [Distributed Scanning](/enterprise/distributed-scanning) - Scale horizontally

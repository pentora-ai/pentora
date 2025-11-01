# Configuration Overview

Pentora uses a hierarchical YAML-based configuration system that allows for flexible, environment-specific settings.

## Configuration Hierarchy

Configuration is loaded in order (later sources override earlier ones):

1. **Builtin defaults** - Compiled into binary
2. **System config** - `/etc/pentora/config.yaml` (Linux) or OS equivalent
3. **User config** - `~/.config/pentora/config.yaml`
4. **Storage config** - `<storage>/config/pentora.yaml`
5. **Custom config** - `--config /path/to/config.yaml`
6. **Environment variables** - `PENTORA_*`
7. **CLI flags** - Command-line arguments

## Configuration File Structure

```yaml
# ~/.config/pentora/config.yaml

storage:
  dir: ~/.local/share/pentora
  enabled: true
  auto_create: true
  retention:
    enabled: true
    max_age: 90d
    max_scans: 1000
    min_free_space: 10GB
  scans:
    compress: false
    keep_artifacts: true
    keep_pcaps: false

scanner:
  default_profile: standard
  rate: 1000
  timeout: 3s
  retry: 1
  ports:
    profile: standard
    custom: []
  concurrency: 100

discovery:
  profile: standard
  timeout: 2s
  retry: 2
  icmp:
    enabled: true
    count: 2
  arp:
    enabled: true
  tcp_probe:
    enabled: false
    ports: [80, 443, 22, 25]

fingerprint:
  enabled: true
  cache_dir: ${storage}/cache/fingerprints
  probe_timeout: 5s
  max_protocols: 3
  catalog:
    builtin: true
    remote_url: https://catalog.pentora.io/fingerprints.yaml
  cache:
    ttl: 7d
    auto_sync: true
  custom_rules: []

logging:
  level: info
  format: text
  output: stderr
  file:
    enabled: false
    path: /var/log/pentora/pentora.log
    max_size: 100MB
    max_backups: 10
    max_age: 30d

server:
  bind: 0.0.0.0:8080
  workers: 4
  api:
    enabled: true
    auth: true
    rate_limit: 100
  ui:
    enabled: true
    path: /ui
    static_dir: /usr/share/pentora/ui
  tls:
    enabled: false
    cert_file: ""
    key_file: ""
  queue:
    max_jobs: 1000
    retention: 7d

engine:
  fail_fast: false
  retry:
    enabled: true
    max_attempts: 3
    backoff: exponential
  global_timeout: 1h
  node_timeout: 10m
  max_parallel_nodes: 10
  data_context:
    max_size: 1GB

notifications:
  default_channels: []
  slack:
    webhook_url: ""
    channel: "#security"
  email:
    smtp_server: ""
    smtp_port: 587
    from: "pentora@company.com"
    to: []

# Enterprise-only sections
enterprise:
  license_file: ${storage}/config/license.key
  multi_tenant:
    enabled: false
  distributed:
    enabled: false
    queue_backend: redis
  integrations:
    siem: []
    ticketing: []

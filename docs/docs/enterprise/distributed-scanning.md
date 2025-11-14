# Distributed Scanning

Scale scanning across multiple worker nodes with centralized job orchestration.

## Architecture

```
[API/UI] → [Job Queue] → [Worker Pool] → [Shared Storage]
                              ↓
                          Worker 1
                          Worker 2
                          Worker 3
```

## Job Queue

Supported backends:
- Redis
- PostgreSQL
- Kafka

Configuration:
```yaml
enterprise:
  distributed:
    enabled: true
    queue_backend: redis
    redis:
      host: redis.company.com
      port: 6379
      db: 0
```

## Worker Configuration

```yaml
server:
  worker:
    mode: distributed
    queue_url: redis://redis.company.com:6379
    concurrency: 10
    heartbeat: 30s
```

Start worker:
```bash
vulntor worker start --queue redis://redis:6379
```

## Job Submission

```bash
# Submit distributed job
vulntor scan --targets 10.0.0.0/8 --distributed --workers 10
```

## Worker Pools

Organize workers by capability:
- **fast-pool**: Quick scans
- **deep-pool**: Comprehensive scans
- **compliance-pool**: CIS/PCI checks

```yaml
worker:
  pool: fast-pool
  tags: [fast, standard]
```

See [Server Mode Deployment](/deployment/server-mode) for setup.

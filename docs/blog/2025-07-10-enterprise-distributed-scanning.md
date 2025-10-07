---
slug: enterprise-distributed-scanning
title: "Enterprise Feature Spotlight: Distributed Scanning"
authors: [pentora_team]
tags: [enterprise, security, network-scanning]
---

Large organizations need to scan thousands of assets across multiple networks and regions. Learn how Pentora Enterprise enables distributed scanning at scale.

<!-- truncate -->

## The Scale Challenge

Enterprise networks are complex:

- **Geographic Distribution**: Assets in multiple data centers and cloud regions
- **Network Segmentation**: DMZs, internal networks, production/staging/dev environments
- **Firewall Constraints**: Limited connectivity between network segments
- **Compliance Requirements**: Data residency and sovereignty rules
- **Performance**: Scanning 100,000+ assets in reasonable time

A single scanner can't efficiently handle this scale.

## Pentora's Distributed Architecture

Pentora Enterprise uses a **coordinator-worker model**:

```
                    ┌─────────────────┐
                    │   Coordinator   │
                    │   (API Server)  │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              │              │              │
         ┌────▼───┐     ┌────▼───┐    ┌────▼───┐
         │Worker 1│     │Worker 2│    │Worker 3│
         │ (US)   │     │ (EU)   │    │ (APAC) │
         └────────┘     └────────┘    └────────┘
             │              │              │
        10.0.0.0/16   172.16.0.0/12  192.168.0.0/16
```

### Components

1. **Coordinator**: Central API server that:
   - Accepts scan requests
   - Distributes jobs to workers
   - Aggregates results
   - Manages job queue and scheduling

2. **Workers**: Lightweight scanning agents that:
   - Poll for jobs
   - Execute scans locally
   - Report results back
   - Handle network-specific access

3. **Job Queue**: Distributed queue (Redis/Kafka) for:
   - Job distribution
   - Priority handling
   - Retry logic
   - Status tracking

## Deployment Scenarios

### Scenario 1: Multi-Region Cloud

Scan AWS infrastructure across regions:

```yaml
# coordinator-config.yaml
server:
  mode: coordinator
  api:
    listen: 0.0.0.0:8080
    auth: jwt

workers:
  auto_register: true
  heartbeat_interval: 30s

queue:
  backend: redis
  url: redis://redis-cluster:6379
```

```yaml
# worker-us-east-1.yaml
worker:
  name: aws-us-east-1
  coordinator: https://pentora-coordinator.example.com
  token: ${WORKER_TOKEN}

  region: us-east-1
  tags:
    - aws
    - production
    - us-east-1

  capacity:
    max_concurrent_scans: 10
    max_targets_per_scan: 1000
```

Deploy workers:

```bash
# US East
docker run -d \
  --name pentora-worker-us-east-1 \
  -e WORKER_TOKEN=${US_WORKER_TOKEN} \
  --network aws-vpc-us-east-1 \
  pentora/pentora-enterprise:latest \
  worker start --config worker-us-east-1.yaml

# EU West
docker run -d \
  --name pentora-worker-eu-west-1 \
  -e WORKER_TOKEN=${EU_WORKER_TOKEN} \
  --network aws-vpc-eu-west-1 \
  pentora/pentora-enterprise:latest \
  worker start --config worker-eu-west-1.yaml
```

### Scenario 2: On-Premises + Cloud Hybrid

Scan both on-prem data center and AWS:

```
┌──────────────────────────────────────┐
│        Cloud (Coordinator)           │
│  pentora-server.example.com          │
└──────────────┬───────────────────────┘
               │
      ┌────────┴─────────┐
      │                  │
┌─────▼──────┐    ┌──────▼────────┐
│On-Prem DC  │    │ AWS VPC       │
│Worker      │    │ Worker        │
│(10.0.0.0/8)│    │(172.31.0.0/16)│
└────────────┘    └───────────────┘
```

Workers connect outbound to coordinator (no inbound firewall rules needed).

### Scenario 3: Air-Gapped Networks

For networks with no internet access:

```yaml
# air-gapped-worker.yaml
worker:
  mode: standalone
  offline: true

  local_queue: /var/pentora/queue
  local_results: /var/pentora/results

sync:
  export_interval: 1h
  export_path: /mnt/usb/pentora-exports
```

Results exported to USB/removable media, then imported to coordinator:

```bash
# On air-gapped worker
pentora worker export \
  --output /mnt/usb/results-2025-07-10.tar.gz

# On coordinator (after physically transferring USB)
pentora coordinator import \
  --file /mnt/usb/results-2025-07-10.tar.gz
```

## Job Distribution

### Target Assignment

Coordinator automatically assigns targets to optimal workers:

```go
func (c *Coordinator) AssignWorker(targets []string) Worker {
    for _, worker := range c.availableWorkers {
        // Check worker capacity
        if worker.CurrentJobs >= worker.MaxConcurrent {
            continue
        }

        // Check network reachability
        if !worker.CanReach(targets) {
            continue
        }

        // Check region/tag preferences
        if job.RequiresTags && !worker.HasTags(job.Tags) {
            continue
        }

        return worker
    }

    return nil // No available worker
}
```

### Priority Queues

Define job priorities:

```bash
# High-priority scan (production incident)
pentora scan 192.168.1.0/24 \
  --priority high \
  --worker-tag production

# Low-priority scan (routine audit)
pentora scan 10.0.0.0/8 \
  --priority low \
  --schedule "0 2 * * *"  # 2 AM daily
```

Queue structure:

```
Critical Priority (P0): [ Job 123 ]
High Priority (P1):     [ Job 124, Job 125 ]
Normal Priority (P2):   [ Job 126, Job 127, Job 128 ]
Low Priority (P3):      [ Job 129, ... ]
```

### Load Balancing

Distribute work evenly:

```yaml
coordinator:
  load_balancing:
    strategy: least_loaded  # Options: round_robin, least_loaded, tag_affinity

    thresholds:
      max_queue_depth: 100
      worker_max_load: 0.8
```

## Worker Management

### Registration

Workers auto-register on startup:

```json
POST /api/v1/workers/register
{
  "name": "worker-us-east-1",
  "version": "1.0.0",
  "capacity": {
    "max_concurrent_scans": 10,
    "max_targets_per_scan": 1000
  },
  "tags": ["aws", "us-east-1", "production"],
  "network_access": [
    "172.31.0.0/16",
    "10.0.0.0/24"
  ]
}
```

Response:

```json
{
  "worker_id": "wrk_a1b2c3d4",
  "token": "jwt_token_here",
  "heartbeat_interval": 30
}
```

### Health Monitoring

Coordinator tracks worker health:

```bash
# View worker status
pentora coordinator workers list

# Output
┌──────────────────┬────────┬───────────┬────────────┬──────────┐
│ Worker ID        │ Status │ Region    │ Load       │ Uptime   │
├──────────────────┼────────┼───────────┼────────────┼──────────┤
│ wrk_us_east_1    │ Active │ us-east-1 │ 3/10 (30%) │ 5d 3h    │
│ wrk_eu_west_1    │ Active │ eu-west-1 │ 7/10 (70%) │ 5d 3h    │
│ wrk_apac_1       │ Idle   │ ap-south  │ 0/10 (0%)  │ 2h 15m   │
│ wrk_onprem_dc1   │ Stale  │ on-prem   │ -          │ -        │
└──────────────────┴────────┴───────────┴────────────┴──────────┘
```

### Auto-Scaling

Integrate with Kubernetes HPA:

```yaml
# worker-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pentora-worker
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: pentora-worker
        image: pentora/pentora-enterprise:latest
        env:
        - name: COORDINATOR_URL
          value: "https://pentora-coordinator"
        - name: WORKER_TOKEN
          valueFrom:
            secretKeyRef:
              name: pentora-worker-token
              key: token

---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: pentora-worker-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: pentora-worker
  minReplicas: 2
  maxReplicas: 20
  metrics:
  - type: Pods
    pods:
      metric:
        name: pentora_queue_depth
      target:
        type: AverageValue
        averageValue: "5"
```

## Scan Execution

### Job Lifecycle

```
1. Request   → Scan submitted via API/CLI
2. Queue     → Job added to priority queue
3. Assign    → Coordinator selects worker
4. Execute   → Worker runs scan
5. Report    → Worker sends results
6. Aggregate → Coordinator combines results
7. Complete  → Results available via API
```

### Real-Time Status

Monitor scan progress:

```bash
pentora job status job-abc123

# Output
Job ID: job-abc123
Status: Running
Progress: 2,456 / 10,000 targets (24.56%)
Workers: 3 active

Worker Breakdown:
- wrk_us_east_1: 856 targets (8.56%) - Running
- wrk_us_east_2: 800 targets (8.00%) - Running
- wrk_us_west_1: 800 targets (8.00%) - Running

Elapsed: 12m 34s
ETA: 38m 16s
```

### Failure Handling

Automatic retry on worker failure:

```yaml
coordinator:
  job_retry:
    max_attempts: 3
    backoff: exponential
    timeout: 30m

  worker_failure:
    action: reassign
    grace_period: 2m
```

If worker crashes:

```
[12:00:00] Worker wrk_us_east_1 assigned job-abc123
[12:05:30] Worker wrk_us_east_1 heartbeat missed
[12:07:30] Worker wrk_us_east_1 marked as failed
[12:07:31] Reassigning job-abc123 to wrk_us_east_2
[12:08:00] Worker wrk_us_east_2 started job-abc123
```

## Result Aggregation

Combine results from multiple workers:

```json
{
  "job_id": "job-abc123",
  "status": "completed",
  "workers": 3,
  "targets_scanned": 10000,
  "duration": "45m 12s",

  "results": {
    "hosts_up": 7234,
    "open_ports": 12456,
    "services_detected": 8901,
    "vulnerabilities": 234
  },

  "worker_stats": [
    {
      "worker_id": "wrk_us_east_1",
      "targets": 3500,
      "duration": "40m 5s"
    },
    {
      "worker_id": "wrk_us_east_2",
      "targets": 3300,
      "duration": "42m 10s"
    },
    {
      "worker_id": "wrk_us_west_1",
      "targets": 3200,
      "duration": "45m 12s"
    }
  ]
}
```

## Performance Benchmarks

Scanning 100,000 assets:

| Setup | Workers | Time | Throughput |
|-------|---------|------|------------|
| Single Scanner | 1 | ~16 hours | 104 targets/min |
| Distributed (5 workers) | 5 | ~3.5 hours | 476 targets/min |
| Distributed (10 workers) | 10 | ~2 hours | 833 targets/min |
| Distributed (20 workers) | 20 | ~1.2 hours | 1,389 targets/min |

Linear scaling up to network bandwidth limits.

## Best Practices

### 1. Worker Placement

Place workers close to targets:

- Same VPC/VLAN
- Minimize network hops
- Respect firewall boundaries

### 2. Capacity Planning

```
Workers needed = (Total targets × Scan time per target) / (Time window × Worker capacity)

Example:
- 50,000 targets
- 2 seconds per target
- 1-hour scan window
- 10 concurrent scans per worker

Workers = (50,000 × 2s) / (3600s × 10) = 2.78 → 3 workers minimum
```

### 3. Network Segmentation

Tag workers by network:

```yaml
# prod-worker.yaml
worker:
  tags:
    - production
    - aws
    - sensitive

# dev-worker.yaml
worker:
  tags:
    - development
    - testing
```

Enforce separation:

```bash
# Only use prod workers
pentora scan prod-network.example.com \
  --require-worker-tag production
```

### 4. Cost Optimization

Use spot instances for non-critical scans:

```yaml
# kubernetes-spot-worker.yaml
nodeSelector:
  node-type: spot
tolerations:
- key: spot
  operator: Equal
  value: "true"
  effect: NoSchedule
```

## Getting Started

### 1. Deploy Coordinator

```bash
docker run -d \
  --name pentora-coordinator \
  -p 8080:8080 \
  -e LICENSE_KEY=${PENTORA_ENTERPRISE_LICENSE} \
  pentora/pentora-enterprise:latest \
  server start --mode coordinator
```

### 2. Deploy Workers

```bash
# Get worker token
WORKER_TOKEN=$(pentora coordinator worker-token create \
  --name "us-east-worker")

# Start worker
docker run -d \
  --name pentora-worker \
  -e COORDINATOR_URL=https://pentora-coordinator:8080 \
  -e WORKER_TOKEN=${WORKER_TOKEN} \
  pentora/pentora-enterprise:latest \
  worker start
```

### 3. Submit Scan

```bash
pentora scan 10.0.0.0/8 \
  --distributed \
  --workers 5
```

## Conclusion

Distributed scanning transforms enterprise security assessments:

- ✅ Scan 100,000+ assets in hours instead of days
- ✅ Respect network boundaries and firewalls
- ✅ Geographic distribution for global organizations
- ✅ Linear scaling with worker count
- ✅ High availability with automatic failover

Ready to scale your scanning? [Contact sales](https://pentora.io/contact) for Pentora Enterprise.

---

**Related:**
- [Enterprise Overview](/docs/enterprise/overview)
- [Distributed Architecture](/docs/enterprise/distributed-scanning)
- [Worker Deployment Guide](/docs/deployment/workers)

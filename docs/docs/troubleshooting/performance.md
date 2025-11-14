# Performance Troubleshooting

Optimize Vulntor for faster scans and lower resource usage.

## Slow Scans

### Symptoms
Scans take longer than expected.

### Diagnosis
```bash
# Enable verbose logging
vulntor scan --targets 192.168.1.0/24 -vv

# Check for bottlenecks in logs
```

### Solutions

#### 1. Increase Rate Limit
```bash
vulntor scan --targets 192.168.1.0/24 --rate 5000
```

#### 2. Increase Concurrency
```bash
vulntor scan --targets 192.168.1.0/24 --concurrency 200
```

#### 3. Use Quick Profile
```bash
vulntor scan --targets 192.168.1.0/24 --profile quick
```

#### 4. Disable Fingerprinting
```bash
vulntor scan --targets 192.168.1.0/24 --no-fingerprint
```

#### 5. Skip Vulnerability Checks
```bash
vulntor scan --targets 192.168.1.0/24 --no-vuln
```

## High Memory Usage

### Symptoms
```
vulntor process using > 2GB RAM
```

### Solutions

#### 1. Reduce Concurrency
```yaml
engine:
  max_parallel_nodes: 5
scanner:
  concurrency: 50
```

#### 2. Limit Context Size
```yaml
engine:
  data_context:
    max_size: 500MB
```

#### 3. Process in Batches
```bash
# Split large target list
split -l 1000 targets.txt batch-

# Scan batches sequentially
for batch in batch-*; do
    vulntor scan --target-file $batch
done
```

## High CPU Usage

### Solutions

#### 1. Reduce Workers
```yaml
server:
  workers: 2
```

#### 2. Limit Parallel Nodes
```yaml
engine:
  max_parallel_nodes: 4
```

## Network Bottlenecks

### Solutions

#### 1. Rate Limiting
```bash
vulntor scan --targets 192.168.1.0/24 --rate 1000
```

#### 2. Timeouts
```bash
vulntor scan --targets 192.168.1.0/24 --timeout 2s
```

## Disk I/O Issues

### Solutions

#### 1. Use Faster Storage
Move storage to SSD.

#### 2. Disable Artifacts
```yaml
storage:
  scans:
    keep_artifacts: false
    keep_pcaps: false
```

#### 3. Enable Compression
```yaml
storage:
  scans:
    compress: true
```

## Benchmarking

```bash
# Benchmark different profiles
time vulntor scan --targets test-network.txt --profile quick
time vulntor scan --targets test-network.txt --profile standard
time vulntor scan --targets test-network.txt --profile deep
```

See [Configuration Overview](/configuration/overview) for tuning options.

# Common Issues and Solutions

Solutions to frequently encountered problems.

## Installation Issues

### Binary Not Found
```
bash: pentora: command not found
```

**Solution**: Add to PATH
```bash
export PATH=$PATH:/usr/local/bin
# Or move binary
sudo mv pentora /usr/local/bin/
```

### Permission Denied
```
Error: permission denied
```

**Solution**: Make executable
```bash
chmod +x pentora
```

## Scanning Issues

### No Hosts Discovered
```
INFO Discovery completed: 0 hosts found
```

**Causes**:
- ICMP blocked by firewall
- Wrong network range
- Network connectivity issues

**Solutions**:
```bash
# Try TCP-based discovery
pentora scan --targets 192.168.1.0/24 --discover-profile tcp

# Skip discovery if hosts known live
pentora scan --targets 192.168.1.100 --no-discover

# Verify connectivity
ping 192.168.1.100
```

### SYN Scan Requires Root
```
Error: raw socket access denied (requires root)
```

**Solutions**:
```bash
# Run with sudo
sudo pentora scan --targets 192.168.1.0/24

# OR use connect scan (no root needed)
pentora scan --targets 192.168.1.0/24 --scan-type connect

# OR set capability
sudo setcap cap_net_raw+ep /usr/local/bin/pentora
```

### Scan Timeout
```
Error: scan timeout after 1h
```

**Solutions**:
```bash
# Increase timeout
pentora scan --targets large-network.txt --timeout 2h

# Reduce scan scope
pentora scan --targets 192.168.1.0/24 --profile quick

# Split into smaller batches
```

### Rate Limit Warnings
```
WARN Rate limit exceeded, throttling
```

**Solutions**:
```bash
# Reduce rate
pentora scan --targets 192.168.1.0/24 --rate 500

# Reduce concurrency
pentora scan --targets 192.168.1.0/24 --concurrency 50
```

## Storage Issues

### Disk Space Exhausted
```
Error: no space left on device
```

**Solutions**:
```bash
# Clean old scans
pentora storage gc --older-than 30d

# Check storage size
du -sh ~/.local/share/pentora

# Enable compression
# Add to config.yaml:
storage:
  scans:
    compress: true
```

### Corrupted Scan Data
```
Error: failed to read scan results
```

**Solution**:
```bash
# Check storage integrity
pentora storage check

# Attempt repair
pentora storage check --fix

# Delete corrupted scan
pentora storage delete <scan-id>
```

## Server Issues

### Port Already in Use
```
Error: bind: address already in use
```

**Solutions**:
```bash
# Use different port
pentora server start --bind 0.0.0.0:9090

# Find process using port
lsof -i :8080
sudo kill <PID>
```

### API Authentication Failed
```
Error: 401 Unauthorized
```

**Solutions**:
```bash
# Check API token
export PENTORA_API_TOKEN=your-token

# Verify token
pentora server token verify
```

## Configuration Issues

### Invalid Config
```
Error: invalid configuration file
```

**Solutions**:
```bash
# Validate config
pentora config validate

# Check YAML syntax
yamllint ~/.config/pentora/config.yaml

# Use default config
pentora scan --targets 192.168.1.100 --config ""
```

See [Performance Troubleshooting](/troubleshooting/performance) for optimization.

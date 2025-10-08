# Scan Profiles

Scan profiles are predefined configurations that bundle scanner settings, port lists, and module selections for common scenarios.

## Builtin Profiles

### quick
Fast reconnaissance scan.

**Characteristics**:
- Top 100 common ports
- Basic ICMP discovery
- Minimal fingerprinting
- No vulnerability checks

**Use cases**: Initial network mapping, time-constrained scans

**Usage**:
```bash
pentora scan --targets 192.168.1.0/24 --profile quick
```

### standard (default)
Balanced scan for general security assessment.

**Characteristics**:
- Top 1000 ports
- Standard discovery (ICMP + ARP)
- Full fingerprinting
- Optional vulnerability checks

**Usage**:
```bash
pentora scan --targets 192.168.1.100 --profile standard
```

### deep
Comprehensive scan for thorough assessment.

**Characteristics**:
- All 65535 ports
- Deep discovery (ICMP + ARP + TCP probes)
- Advanced fingerprinting with multiple protocols
- Vulnerability checks enabled

**Usage**:
```bash
pentora scan --targets 192.168.1.100 --profile deep
```

### webapp
Web application focused scan.

**Characteristics**:
- Web ports (80, 443, 8080, 8443, etc.)
- HTTP/HTTPS fingerprinting
- TLS analysis
- Web framework detection

**Usage**:
```bash
pentora scan --targets example.com --profile webapp
```

## Custom Profiles

Create custom profiles in `~/.config/pentora/profiles/`:

```yaml
# ~/.config/pentora/profiles/production.yaml
name: production
description: Production network scan with conservative settings

scanner:
  rate: 500                # Conservative rate
  timeout: 5s
  retry: 2
  ports:
    profile: standard
  concurrency: 50

discovery:
  profile: standard
  timeout: 3s

fingerprint:
  enabled: true
  max_protocols: 2

vulnerability:
  enabled: true
  severity_threshold: medium

notifications:
  channels:
    - slack://prod-security
```

**Usage**:
```bash
pentora scan --targets prod-network.txt --profile production
```

## Profile Reference

Profiles stored in:
- System: `/etc/pentora/profiles/`
- User: `~/.config/pentora/profiles/`
- Workspace: `<workspace>/config/profiles/`

See [Configuration Overview](/configuration/overview) for complete schema.

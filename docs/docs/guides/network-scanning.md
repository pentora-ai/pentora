# Network Scanning Best Practices

Guidelines for effective and responsible network scanning.

## Authorization

**Always obtain written authorization** before scanning:
- Internal networks: IT/Security approval
- External networks: Pentesting agreement
- Cloud environments: Account owner permission

Unauthorized scanning may violate laws (CFAA, Computer Misuse Act).

## Scan Planning

### 1. Define Scope
```bash
# Include targets
vulntor scan --targets 192.168.1.0/24

# Exclude sensitive hosts
vulntor scan --targets 10.0.0.0/16 --exclude-file sensitive.txt
```

### 2. Choose Profile
- **quick**: Initial reconnaissance
- **standard**: General assessment  
- **deep**: Comprehensive audit

### 3. Schedule
Avoid business hours for production networks:
```bash
# Schedule for 2 AM daily
vulntor scan --targets prod.txt --schedule "0 2 * * *" --server https://vulntor.company.com
```

## Rate Limiting

Prevent network disruption:

```bash
# Conservative rate (production)
vulntor scan --targets prod-network.txt --rate 100 --concurrency 10

# Standard rate (dev/test)
vulntor scan --targets dev-network.txt --rate 1000 --concurrency 50

# Aggressive rate (lab/offline)
vulntor scan --targets lab.txt --rate 5000 --concurrency 200
```

## Discovery Strategies

### ICMP Blocked
Use TCP-based discovery:
```bash
vulntor scan --targets 192.168.1.0/24 --discover-profile tcp
```

### Large Networks
Split into phases:
```bash
# Phase 1: Discovery
vulntor scan --targets 10.0.0.0/8 --only-discover -o live-hosts.txt

# Phase 2: Detailed scan
vulntor scan --target-file live-hosts.txt --no-discover --profile standard
```

## Handling False Positives

Review and refine:
```bash
# Compare scans
vulntor storage show scan-1 > scan-1.json
vulntor storage show scan-2 > scan-2.json
diff scan-1.json scan-2.json
```

## Legal and Ethical

- Obtain authorization
- Follow scope boundaries
- Respect rate limits
- Document findings
- Report responsibly

See [Vulnerability Assessment Guide](/guides/vulnerability-assessment) for CVE analysis.

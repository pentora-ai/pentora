# Scan Pipeline

Pentora's scan pipeline is a structured 9-stage process that transforms raw targets into actionable security intelligence. Each stage builds upon the previous, creating a data flow from initial target specification to final reporting.

## Pipeline Overview

```
┌─────────────────────┐
│ 1. Target Ingestion │
└──────────┬──────────┘
           │
┌──────────▼──────────┐
│ 2. Asset Discovery  │
└──────────┬──────────┘
           │
┌──────────▼──────────┐
│ 3. Port Scanning    │
└──────────┬──────────┘
           │
┌──────────▼────────────┐
│ 4. Service            │
│    Fingerprinting     │
└──────────┬────────────┘
           │
┌──────────▼──────────┐
│ 5. Asset Profiling  │
└──────────┬──────────┘
           │
┌──────────▼────────────┐
│ 6. Vulnerability      │
│    Evaluation         │
└──────────┬────────────┘
           │
┌──────────▼────────────┐
│ 7. Compliance &       │
│    Risk Scoring       │
└──────────┬────────────┘
           │
┌──────────▼────────────┐
│ 8. Reporting &        │
│    Notification       │
└──────────┬────────────┘
           │
┌──────────▼────────────┐
│ 9. Archival &         │
│    Analytics          │
└───────────────────────┘
```

## Stage 1: Target Ingestion

**Purpose**: Parse, validate, and prepare target specifications for scanning.

**Operations**:
1. **Parse input formats**:
   - Single IP: `192.168.1.100`
   - CIDR notation: `192.168.1.0/24`
   - IP ranges: `192.168.1.1-192.168.1.254`
   - Hostnames: `example.com`
   - From file: `--target-file targets.txt`

2. **Expand CIDR ranges**:
   - Convert `/24` → 256 individual IPs
   - Apply pagination for large ranges to avoid memory exhaustion

3. **Apply blocklists**:
   - Filter RFC1918 private ranges (configurable)
   - Exclude user-defined blocklists
   - Skip broadcast/network addresses

4. **Validate targets**:
   - DNS resolution for hostnames
   - Validate IP format
   - Check for duplicates

**Output**: Sanitized list of IP addresses ready for scanning.

**Configuration**:
```yaml
scanner:
  target_expansion:
    max_cidr_size: /16  # Largest allowed CIDR
    resolve_hostnames: true
  blocklists:
    - 127.0.0.0/8      # Loopback
    - 169.254.0.0/16   # Link-local
```

**CLI Control**: Target ingestion always runs; configure via `--targets` or `--target-file`.

## Stage 2: Asset Discovery

**Purpose**: Identify live hosts before performing expensive port scans.

**Methods**:
1. **ICMP Echo (Ping)**:
   - Send ICMP ECHO_REQUEST
   - Requires raw socket permissions (`CAP_NET_RAW` or root)
   - Fast but can be blocked by firewalls

2. **ARP Discovery** (local networks):
   - Layer 2 discovery for same-subnet targets
   - Cannot be blocked by host firewalls
   - Only works on local network segments

3. **TCP SYN Ping**:
   - Send TCP SYN to common ports (80, 443, 22)
   - Useful when ICMP is blocked
   - Requires raw sockets

**Discovery Profiles**:
- `fast`: ICMP only
- `standard`: ICMP + ARP (local nets)
- `deep`: ICMP + ARP + TCP SYN probes
- `tcp`: TCP SYN only (no ICMP)

**Output**: List of responsive hosts with response times.

**Configuration**:
```yaml
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
```

**CLI Control**:
```bash
# Discovery only
pentora scan --targets 192.168.1.0/24 --only-discover

# Skip discovery (targets known to be live)
pentora scan --targets 192.168.1.100 --no-discover

# Custom discovery profile
pentora scan --targets 192.168.1.0/24 --discover-profile deep
```

**Performance**: Discovers 1000 hosts in ~10-30 seconds depending on profile and network conditions.

## Stage 3: Port Scanning

**Purpose**: Identify open TCP/UDP ports on discovered hosts.

**Scanning Methods**:
1. **TCP SYN Scan** (default):
   - Send SYN packet, analyze SYN-ACK response
   - Stealthy (no full connection)
   - Requires raw sockets

2. **TCP Connect Scan**:
   - Full 3-way handshake
   - No special permissions needed
   - Leaves connection logs

3. **UDP Scan**:
   - Send UDP probes, check ICMP port unreachable
   - Slow due to rate limiting
   - Often requires protocol-specific payloads

**Port Selection**:
- **Quick profile**: Top 100 common ports
- **Standard profile**: Top 1000 ports (Nmap default)
- **Deep profile**: All 65,535 ports
- **Custom**: User-specified port list

**Concurrency & Rate Limiting**:
```yaml
scanner:
  rate: 1000            # Packets per second
  concurrency: 100      # Parallel targets
  timeout: 3s
  retry: 1
  ports:
    profile: standard
    custom: [80, 443, 8080, 8443]
```

**Dark Subnet Detection**:
- Identifies networks with no responses (all packets dropped)
- Triggers timeout backoff to avoid wasting time
- Logged for operator awareness

**Output**: List of open ports per host with state (open/closed/filtered).

**CLI Control**:
```bash
# Use predefined profile
pentora scan --targets 192.168.1.100 --profile quick

# Scan specific ports
pentora scan --targets 192.168.1.100 --ports 80,443,8080

# Scan port range
pentora scan --targets 192.168.1.100 --ports 1-1024

# Adjust rate limit
pentora scan --targets 192.168.1.0/24 --rate 500
```

**Performance**: Scans 1000 ports on a single host in ~5-10 seconds at default rate.

## Stage 4: Service Fingerprinting

**Purpose**: Identify the service, application, version, and OS running on open ports.

**Layered Detection**:

### Layer 1: Initial Heuristics
- Port number → service guess (80=HTTP, 22=SSH)
- Initial banner capture (connect and read)
- Basic pattern matching

### Layer 2: Protocol-Specific Probes
Targeted probes based on Layer 1 results:

**HTTP/HTTPS**:
- `GET / HTTP/1.1` with headers
- Parse `Server:`, `X-Powered-By:`, `X-AspNet-Version:`
- Detect frameworks (Laravel, Django, Express)

**TLS/SSL**:
- TLS handshake
- Extract certificate details (CN, SAN, issuer)
- Identify cipher suites and protocol versions

**SMTP/IMAP/POP3**:
- Read greeting banner
- Send EHLO/CAPABILITY commands
- Parse extension lists

**FTP**:
- Read welcome banner
- Send SYST for OS detection
- Check for anonymous access

**Redis/Memcached**:
- Send INFO command
- Parse version and configuration

### Layer 3: Confidence Scoring
Aggregate evidence from multiple sources:
```json
{
  "fingerprints": [
    {
      "match": "nginx",
      "version": "1.18.0",
      "confidence": 95,
      "source": "http_header",
      "evidence": "Server: nginx/1.18.0"
    },
    {
      "match": "ubuntu",
      "confidence": 80,
      "source": "banner",
      "evidence": "Ubuntu Linux"
    }
  ]
}
```

**Fingerprint Database**:
- Builtin rules compiled into binary
- Cached catalogs in workspace: `<workspace>/cache/fingerprints/`
- Sync remote catalogs: `pentora fingerprint sync`

**Output**: Service records with application, version, OS, and confidence scores.

**Configuration**:
```yaml
fingerprint:
  cache_dir: ${workspace}/cache/fingerprints
  probe_timeout: 5s
  max_protocols: 3  # Max protocols to probe per port
  catalog:
    builtin: true
    remote_url: https://catalog.pentora.io/fingerprints.yaml
```

**CLI Control**:
```bash
# Use cached fingerprints
pentora scan --targets 192.168.1.100 --fingerprint-cache

# Update fingerprint catalog
pentora fingerprint sync
```

See [Fingerprinting System](/docs/concepts/fingerprinting) for detailed probe specifications.

## Stage 5: Asset Profiling

**Purpose**: Fuse signals from discovery, ports, and fingerprints into a comprehensive asset profile.

**Profile Components**:
1. **Device Classification**:
   - Server, workstation, network device, IoT, mobile
   - Based on open ports, services, OS detection

2. **Operating System**:
   - OS family (Linux, Windows, BSD, macOS)
   - Distribution (Ubuntu, CentOS, Windows Server 2019)
   - Version and kernel

3. **Application Stack**:
   - Web server (nginx, Apache, IIS)
   - Application server (Tomcat, Node.js, Gunicorn)
   - Frameworks (Django, Rails, ASP.NET)
   - Databases (MySQL, PostgreSQL, MongoDB)

4. **Network Function**:
   - Web server, mail server, DNS server, database server
   - Multi-function hosts (e.g., web + database)

**Profile Confidence**:
- High confidence: Multiple corroborating signals
- Medium confidence: Single strong signal
- Low confidence: Weak heuristics only

**Output**: Asset inventory records suitable for CMDB integration.

Example profile:
```json
{
  "host": "192.168.1.100",
  "device_type": "server",
  "os": {
    "family": "linux",
    "distribution": "ubuntu",
    "version": "20.04",
    "confidence": 90
  },
  "applications": [
    {"name": "nginx", "version": "1.18.0", "role": "web_server"},
    {"name": "php", "version": "7.4", "role": "runtime"},
    {"name": "mysql", "version": "8.0", "role": "database"}
  ],
  "functions": ["web_server", "database_server"]
}
```

## Stage 6: Vulnerability Evaluation

**Purpose**: Identify known vulnerabilities (CVEs) and common misconfigurations.

**Detection Methods**:

### CVE Matching
- Match service versions against CVE database
- Consider version ranges and patch levels
- Filter by exploitability and severity (CVSS score)

### Misconfiguration Checks
- Default credentials (admin/admin, root/toor)
- Weak protocols (SSLv3, TLSv1.0, FTP)
- Anonymous access (FTP, SMB, Redis)
- Missing security headers (HTTP)
- Exposed admin interfaces

### Heuristic Checks
- Outdated software versions
- End-of-life products
- Known vulnerable services (ElasticSearch, MongoDB)

**Output**: Vulnerability records with severity, CVSS, and remediation.

```json
{
  "host": "192.168.1.100",
  "port": 80,
  "vulnerability": {
    "id": "CVE-2021-44228",
    "title": "Log4Shell Remote Code Execution",
    "severity": "critical",
    "cvss": 10.0,
    "affected": "Apache Log4j 2.0-2.14.1",
    "detected": "2.14.0",
    "remediation": "Upgrade to 2.15.0 or set log4j2.formatMsgNoLookups=true"
  }
}
```

**CLI Control**:
```bash
# Enable vulnerability checks
pentora scan --targets 192.168.1.100 --vuln

# Disable vulnerability checks (faster)
pentora scan --targets 192.168.1.100 --no-vuln
```

## Stage 7: Compliance & Risk Scoring

**Purpose**: Evaluate findings against regulatory frameworks and assign risk scores.

**Compliance Frameworks** (Enterprise):
- **CIS Benchmarks**: Center for Internet Security baselines
- **PCI DSS**: Payment Card Industry Data Security Standard
- **NIST 800-53**: National Institute of Standards and Technology controls
- **HIPAA**: Health Insurance Portability and Accountability Act
- **ISO 27001**: Information security management

**Risk Scoring**:
Calculate risk based on:
1. **Vulnerability severity**: Critical > High > Medium > Low
2. **Asset value**: Critical systems weighted higher
3. **Exploitability**: Public exploits increase risk
4. **Exposure**: Internet-facing vs internal

**Risk Formula**:
```
Risk Score = (Severity × Exploitability × Exposure) / Mitigations
```

**Output**: Compliance violations and aggregated risk scores.

```json
{
  "host": "192.168.1.100",
  "compliance": {
    "framework": "PCI-DSS",
    "violations": [
      {
        "control": "2.2.4",
        "description": "Configure system security parameters",
        "finding": "Weak SSL/TLS configuration detected"
      }
    ]
  },
  "risk_score": 8.5,
  "risk_level": "high"
}
```

**CLI Control** (Enterprise):
```bash
# Run compliance checks
pentora scan --targets cardholder-env.txt --compliance pci-dss

# Multiple frameworks
pentora scan --targets dmz.txt --compliance cis-level1,nist-800-53
```

## Stage 8: Reporting & Notification

**Purpose**: Generate structured reports and trigger external integrations.

**Report Formats**:
- **JSON**: Machine-readable, suitable for SIEM ingestion
- **JSONL**: Line-delimited for streaming and large datasets
- **CSV**: Spreadsheet import, executive summaries
- **PDF**: Executive reports with charts and remediation (Enterprise)
- **HTML**: Interactive dashboards

**Notification Channels**:
- **Slack**: Post scan summaries to channels
- **Email**: Send reports to stakeholders
- **Webhooks**: POST results to external systems
- **Ticketing**: Auto-create Jira/ServiceNow tickets (Enterprise)
- **SIEM**: Forward to Splunk/QRadar/Elastic (Enterprise)

**Notification Rules**:
```yaml
notifications:
  - name: critical_vulns
    channels: [slack, email]
    conditions:
      severity: [critical]
      asset_tags: [production]
  - name: compliance_violations
    channels: [jira]
    conditions:
      compliance_failed: true
```

**Output**: Reports written to workspace and external systems notified.

**CLI Control**:
```bash
# Specify output format
pentora scan --targets 192.168.1.100 --output json

# Export to file
pentora scan --targets 192.168.1.100 -o results.csv

# Trigger notifications (server mode)
curl -X POST /api/scans -d '{"targets": [...], "notify": ["slack://security"]}'
```

## Stage 9: Archival & Analytics

**Purpose**: Store scan results for historical analysis and trend detection.

**Workspace Storage**:
Results saved to: `<workspace>/scans/<scan-id>/`
```
scans/20231006-143022-a1b2c3/
├── request.json       # Original scan parameters
├── status.json        # Execution metadata
├── results.jsonl      # Main results (line-delimited JSON)
├── artifacts/
│   ├── banners/       # Raw banner captures
│   ├── screenshots/   # Web screenshots (if enabled)
│   └── pcaps/         # Packet captures (if enabled)
└── reports/
    ├── summary.json
    ├── vulnerabilities.csv
    └── executive.pdf
```

**Retention Policies**:
```yaml
workspace:
  retention:
    enabled: true
    max_age: 90d         # Delete scans older than 90 days
    max_scans: 1000      # Keep at most 1000 scans
    min_free_space: 10GB # Delete oldest when space low
```

**Analytics** (Enterprise):
- **Trend analysis**: Compare scans over time
- **Diff detection**: New/resolved vulnerabilities
- **Asset changes**: Added/removed services
- **Risk trends**: Organizational risk over time
- **Compliance posture**: Historical compliance scores

**CLI Control**:
```bash
# Clean old scans
pentora workspace gc --older-than 30d

# Disable workspace (stateless)
pentora scan --targets 192.168.1.100 --no-workspace

# Custom workspace location
pentora scan --targets 192.168.1.100 --workspace-dir /data/pentora
```

## Pipeline Control

### Phase Flags

Control which stages execute:

```bash
# Discovery only (stages 1-2)
pentora scan --targets 192.168.1.0/24 --only-discover

# Skip discovery (stages 1, 3-9)
pentora scan --targets 192.168.1.100 --no-discover

# Disable vulnerability checks (stages 1-5, 8-9)
pentora scan --targets 192.168.1.100 --no-vuln

# Full pipeline with all stages
pentora scan --targets 192.168.1.100 --vuln
```

### Profiles

Predefined profiles configure multiple stages:

```bash
# Quick: Fast discovery, top 100 ports, no vuln
pentora scan --targets 192.168.1.0/24 --profile quick

# Standard: Standard discovery, top 1000 ports, basic fingerprint
pentora scan --targets 192.168.1.0/24 --profile standard

# Deep: Thorough discovery, all ports, advanced fingerprint, vuln checks
pentora scan --targets 192.168.1.0/24 --profile deep
```

See [Scan Profiles](/docs/configuration/scan-profiles) for custom profile creation.

## Performance Characteristics

Typical scan times (single host, standard profile):

| Stage                   | Time      | Bottleneck       |
|------------------------|-----------|------------------|
| Target Ingestion       | ~1s       | CPU              |
| Asset Discovery        | 2-5s      | Network latency  |
| Port Scanning          | 5-10s     | Rate limiting    |
| Service Fingerprinting | 10-30s    | Protocol probes  |
| Asset Profiling        | ~1s       | CPU              |
| Vulnerability Eval     | 5-15s     | Database lookups |
| Compliance Scoring     | ~1s       | CPU              |
| Reporting              | 1-5s      | I/O              |
| Archival               | ~1s       | I/O              |

**Total**: ~25-70 seconds per host depending on open ports and enabled checks.

**Large Networks**: Parallelism allows scanning 1000 hosts in 10-20 minutes with proper rate limiting.

## Error Handling

Each stage can fail independently:

1. **Fail-fast mode**: Stop pipeline on first error
2. **Continue-on-error**: Skip failed stage, continue with available data
3. **Retry logic**: Transient failures retried with exponential backoff

**Configuration**:
```yaml
engine:
  fail_fast: false         # Continue on errors
  retry:
    enabled: true
    max_attempts: 3
    backoff: exponential
```

**Dependent Stages**:
If a stage fails, dependent stages are skipped:
```
Port Scan FAILED
  ↓
Banner Grab SKIPPED (no ports)
  ↓
Fingerprint SKIPPED (no banners)
```

Reporting stage always runs to capture partial results.

## Next Steps

- [DAG Engine](/docs/concepts/dag-engine) - How stages are orchestrated
- [Fingerprinting](/docs/concepts/fingerprinting) - Deep dive into Layer 4
- [Workspace](/docs/concepts/workspace) - Where results are stored
- [Scan Profiles](/docs/configuration/scan-profiles) - Customizing pipeline behavior

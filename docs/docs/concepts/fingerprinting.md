# Fingerprinting System

Vulntor's fingerprinting system identifies services, applications, versions, and operating systems through a layered detection approach that combines heuristics, protocol-specific probes, and confidence scoring.

## Overview

Service fingerprinting goes beyond simple banner matching. Vulntor employs a multi-stage approach:

1. **Initial heuristics** - Port number and basic banner
2. **Protocol-specific probes** - Targeted requests per protocol
3. **Confidence scoring** - Aggregate evidence from multiple sources
4. **Multiple match support** - Surface all detected technologies

## Layered Detection

### Layer 1: Initial Heuristics

First-pass identification using readily available information:

**Port-Based Heuristics**:
```
Port 22   → Likely SSH
Port 80   → Likely HTTP
Port 443  → Likely HTTPS
Port 3306 → Likely MySQL
```

**Banner Matching**:
```
"SSH-2.0-OpenSSH_8.2p1" → OpenSSH 8.2p1
"220 mail.example.com ESMTP Postfix" → Postfix SMTP
```

**Confidence**: Low to Medium (30-60%)
- Port heuristics alone: 30-40% confidence
- Simple banner match: 50-60% confidence

### Layer 2: Protocol-Specific Probes

Targeted probes confirm and refine Layer 1 guesses:

#### HTTP/HTTPS Probes

```http
GET / HTTP/1.1
Host: target.com
User-Agent: Vulntor/1.0
Accept: */*
Connection: close
```

**Analyzed Headers**:
- `Server`: Web server identification (e.g., `nginx/1.18.0`)
- `X-Powered-By`: Application framework (e.g., `PHP/7.4.3`)
- `X-AspNet-Version`: ASP.NET version
- `X-Generator`: CMS identification (e.g., `WordPress 5.8`)

**Content Analysis**:
- HTML comments: `<!-- Built with Django -->`
- Meta tags: `<meta name="generator" content="Drupal 9">`
- JavaScript frameworks: Detect React, Angular, Vue.js
- CSS framework signatures: Bootstrap, Tailwind

**Confidence**: Medium to High (60-90%)

#### HTTPS/TLS Probes

```
TLS ClientHello → ServerHello + Certificate
```

**Analyzed Fields**:
- Certificate Common Name (CN) and Subject Alternative Names (SAN)
- Issuer information
- TLS version (TLS 1.2, TLS 1.3)
- Cipher suites offered and selected
- Extensions (SNI, ALPN, Session tickets)

**Identification**:
- JA3/JA3S fingerprints for TLS client/server
- Certificate issuer patterns (Let's Encrypt, DigiCert)
- Self-signed detection

**Confidence**: Medium to High (60-85%)

#### SMTP/IMAP/POP3 Probes

**SMTP**:
```
EHLO vulntor.scanner
```

Response:
```
250-mail.example.com
250-PIPELINING
250-SIZE 52428800
250-STARTTLS
250 ENHANCEDSTATUSCODES
```

Identifies:
- SMTP server (Postfix, Exim, Sendmail)
- Supported extensions
- TLS support

**IMAP**:
```
A001 CAPABILITY
```

Response:
```
* CAPABILITY IMAP4rev1 LITERAL+ SASL-IR LOGIN-REFERRALS ID ENABLE IDLE AUTH=PLAIN AUTH=LOGIN
A001 OK Capability completed.
```

**Confidence**: High (75-90%)

#### FTP Probes

```
Connect → Read Banner
SYST → Get System Type
```

Banner:
```
220 ProFTPD 1.3.6 Server (Debian)
```

SYST Response:
```
215 UNIX Type: L8
```

Identifies:
- FTP server (ProFTPD, vsftpd, Pure-FTPd)
- Operating system
- Anonymous access availability

**Confidence**: High (80-95%)

#### Redis Probes

```
INFO SERVER
```

Response:
```
# Server
redis_version:6.2.6
redis_mode:standalone
os:Linux 5.10.0-8-amd64 x86_64
```

**Confidence**: Very High (90-95%)

#### SSH Probes

```
Connect → Read SSH banner
```

Banner:
```
SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.3
```

**Key Exchange Analysis**:
- Supported algorithms
- Encryption methods
- Compression

**Confidence**: High (80-90%)

### Layer 3: Confidence Scoring

Aggregate evidence from multiple sources:

```json
{
  "host": "192.168.1.100",
  "port": 80,
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
      "version": "20.04",
      "confidence": 80,
      "source": "http_header",
      "evidence": "Server: nginx/1.18.0 (Ubuntu)"
    },
    {
      "match": "php",
      "version": "7.4.3",
      "confidence": 90,
      "source": "http_header",
      "evidence": "X-Powered-By: PHP/7.4.3"
    },
    {
      "match": "wordpress",
      "version": "5.8",
      "confidence": 85,
      "source": "http_content",
      "evidence": "<meta name=\"generator\" content=\"WordPress 5.8\">"
    }
  ]
}
```

**Confidence Levels**:
- **0-30%**: Weak guess (port heuristic only)
- **30-60%**: Low confidence (single weak signal)
- **60-80%**: Medium confidence (single strong signal)
- **80-95%**: High confidence (multiple corroborating signals)
- **95-100%**: Very high confidence (explicit version strings)

### Layer 4: Multiple Match Support

Vulntor surfaces all detected technologies, not just the primary service:

**Web Server Stack Example**:
```
Port 443/tcp open
├── nginx 1.18.0 (Web Server, 95% confidence)
├── PHP 7.4.3 (Runtime, 90% confidence)
├── WordPress 5.8 (CMS, 85% confidence)
├── MySQL 8.0 (Database, inferred from PHP/WordPress, 70% confidence)
└── Ubuntu 20.04 (OS, 80% confidence)
```

**Benefits**:
- Complete technology stack visibility
- Better vulnerability correlation
- Comprehensive asset inventory

## Fingerprint Database

### Builtin Rules

Compiled into Vulntor binary:

```yaml
# builtin fingerprints
fingerprints:
  - name: openssh
    category: ssh
    patterns:
      - type: banner
        regex: 'SSH-2\.0-OpenSSH_([0-9.]+)'
        version_group: 1
      - type: banner
        regex: 'SSH-2\.0-OpenSSH_([0-9.]+p[0-9]+) Ubuntu-([0-9.]+)'
        version_group: 1
        os: ubuntu
        os_version_group: 2

  - name: nginx
    category: http
    patterns:
      - type: http_header
        header: Server
        regex: 'nginx/([0-9.]+)'
        version_group: 1
        confidence: 95

  - name: apache
    category: http
    patterns:
      - type: http_header
        header: Server
        regex: 'Apache/([0-9.]+)'
        version_group: 1
        confidence: 95
```

### Remote Catalogs

Sync updated fingerprints from remote repository:

```bash
# Sync from default catalog
vulntor fingerprint sync

# Sync from custom URL
vulntor fingerprint sync --url https://custom.repo/fingerprints.yaml

# Show available catalogs
vulntor fingerprint list-catalogs
```

**Cached Location**: `<storage>/cache/fingerprints/`

**Update Frequency**: Configurable TTL (default: 7 days)

```yaml
fingerprint:
  cache:
    ttl: 7d
    auto_sync: true
  catalog:
    remote_url: https://catalog.vulntor.io/fingerprints.yaml
```

### Custom Fingerprints

Add organization-specific rules:

```yaml
# ~/.config/vulntor/fingerprints/custom.yaml
fingerprints:
  - name: internal_webapp
    category: http
    patterns:
      - type: http_header
        header: X-App-Name
        regex: 'InternalApp/([0-9.]+)'
        version_group: 1
        confidence: 95

  - name: custom_ssh_banner
    category: ssh
    patterns:
      - type: banner
        regex: 'SSH-2\.0-CustomSSH_([0-9.]+)'
        version_group: 1
        confidence: 90
```

Load custom rules:

```bash
vulntor scan --targets 192.168.1.100 --fingerprint-rules custom.yaml
```

See [Custom Fingerprints Guide](/advanced/custom-fingerprints) for rule syntax.

## Probe Execution

### Probe Definitions

Probes defined in YAML catalog:

```yaml
# pkg/fingerprint/data/probes.yaml
probes:
  - name: http_get
    protocol: http
    trigger:
      - port: 80
      - port: 8080
      - service_hint: http
    request: |
      GET / HTTP/1.1
      Host: {target}
      User-Agent: Vulntor/1.0
      Accept: */*
      Connection: close

    timeout: 5s
    max_size: 1MB

  - name: https_get
    protocol: https
    trigger:
      - port: 443
      - port: 8443
      - service_hint: https
    tls: true
    request: |
      GET / HTTP/1.1
      Host: {target}
      User-Agent: Vulntor/1.0
      Accept: */*
      Connection: close

    timeout: 10s

  - name: smtp_ehlo
    protocol: smtp
    trigger:
      - port: 25
      - port: 587
      - service_hint: smtp
    request: "EHLO vulntor.scanner\r\n"
    timeout: 5s

  - name: imap_capability
    protocol: imap
    trigger:
      - port: 143
      - port: 993
      - service_hint: imap
    request: "A001 CAPABILITY\r\n"
    timeout: 5s

  - name: redis_info
    protocol: redis
    trigger:
      - port: 6379
      - service_hint: redis
    request: "INFO SERVER\r\n"
    timeout: 3s
```

### Trigger Logic

Probes execute based on:

1. **Port number**: Standard ports trigger specific probes
2. **Service hints**: Layer 1 guesses influence probe selection
3. **Explicit requests**: User specifies protocols to probe

**Example Flow**:
```
Port 443 detected open
  ↓
Layer 1: Port heuristic → Likely HTTPS (40% confidence)
  ↓
Trigger: https_get, tls_fingerprint probes
  ↓
Execute probes → Collect evidence
  ↓
Layer 2: Parse HTTP headers, TLS certificate
  ↓
Fingerprint match: nginx 1.18.0 (95% confidence)
```

### Probe Priority

When multiple protocols possible, probe in order:

1. **TLS/SSL**: Always probe first on common TLS ports
2. **HTTP/HTTPS**: High priority for web services
3. **Email protocols**: SMTP, IMAP, POP3
4. **Databases**: Redis, MySQL, PostgreSQL, MongoDB
5. **Other services**: FTP, SSH, Telnet

**Max protocols per port**: Configurable (default: 3)

```yaml
fingerprint:
  max_protocols: 3  # Stop after 3 successful identifications
```

### Response Handling

Each probe captures:

- **Raw response**: Complete protocol output
- **Timing**: Response latency
- **Status**: Success, timeout, error
- **Evidence fields**: Extracted data (headers, banners, etc.)

Stored in `artifacts/banners/`:
```
192.168.1.100-80-http.txt
192.168.1.100-443-https.txt
192.168.1.100-22-ssh.txt
```

## Fingerprint Matching

### Rule Processing

For each captured response:

1. **Select applicable rules**: Match protocol and category
2. **Apply patterns**: Test regex against response
3. **Extract version**: Capture groups for version/OS
4. **Score confidence**: Based on pattern specificity
5. **Deduplicate**: Merge redundant matches

### Regex Patterns

Named capture groups extract version information:

```yaml
patterns:
  - type: banner
    regex: 'Apache/(?P<version>[0-9.]+) \((?P<os>[^)]+)\)'
    confidence: 90
```

Match: `Apache/2.4.41 (Ubuntu)`

Extracted:
- `version`: `2.4.41`
- `os`: `Ubuntu`
- `confidence`: 90

### Confidence Calculation

Base confidence from pattern, adjusted by:

**+10%**: Multiple corroborating signals
**+5%**: Explicit version string (not just product name)
**-10%**: Ambiguous match (many possible products)
**-20%**: Port heuristic only

**Example**:
```
Base pattern confidence: 85
+ Version string present: +5
+ HTTP header match: +10
= Final confidence: 100 (capped at 100)
```

### Deduplication

Merge redundant matches:

```
Before:
- nginx 1.18.0 (http_header, 95%)
- nginx 1.18.0 (http_content, 80%)

After:
- nginx 1.18.0 (http_header, http_content, 95%)
```

Highest confidence retained, sources combined.

## CLI Integration

### Basic Fingerprinting

Enabled by default in standard scans:

```bash
vulntor scan --targets 192.168.1.100
```

Output includes fingerprints:
```
192.168.1.100:80 open
  Service: nginx 1.18.0 (95% confidence)
  OS: Ubuntu 20.04 (80% confidence)
  Stack: PHP 7.4.3 (90% confidence)
```

### Disable Fingerprinting

Skip fingerprinting for faster scans:

```bash
vulntor scan --targets 192.168.1.100 --no-fingerprint
```

Only port states reported, no service identification.

### Fingerprint Cache

Use cached fingerprint database:

```bash
# Enable caching (faster, may be outdated)
vulntor scan --targets 192.168.1.100 --fingerprint-cache

# Force refresh
vulntor fingerprint sync --force
```

### Custom Rules

Load additional rules:

```bash
vulntor scan --targets 192.168.1.100 --fingerprint-rules /path/to/custom.yaml
```

### Verbose Output

Show all fingerprint matches and confidence scores:

```bash
vulntor scan --targets 192.168.1.100 --verbose
```

## Performance Considerations

### Probe Overhead

Each protocol probe adds latency:

- **Simple probe** (Redis INFO): ~10-50ms
- **HTTP GET**: ~50-200ms
- **HTTPS with TLS handshake**: ~100-500ms
- **Complex multi-stage probe**: ~200-1000ms

**Total fingerprinting time**: 5-30 seconds per host depending on open ports.

### Optimization Strategies

#### 1. Limit Probe Count

```yaml
fingerprint:
  max_protocols: 2  # Stop after 2 successful IDs per port
```

#### 2. Parallel Probing

Probe multiple ports simultaneously:

```yaml
fingerprint:
  probe_concurrency: 10  # Probe up to 10 ports in parallel
```

#### 3. Cache Results

Reuse fingerprints for known hosts:

```yaml
fingerprint:
  cache:
    enabled: true
    ttl: 24h
```

#### 4. Skip Low-Priority Ports

Focus on interesting services:

```yaml
fingerprint:
  skip_ports:
    - 1-1023  # Skip well-known ports if time-constrained
```

### Memory Usage

Fingerprint catalog loaded into memory:

- **Builtin rules**: ~1-5 MB
- **Remote catalog**: ~5-20 MB
- **Custom rules**: Variable

**Per-scan memory**: ~10-100 KB per host depending on open ports and responses.

## Integration with Asset Profiling

Fingerprints feed into asset profiling:

```
Fingerprints:
  - nginx 1.18.0
  - PHP 7.4.3
  - WordPress 5.8
  - Ubuntu 20.04

Asset Profile:
  Device Type: Server
  OS: Linux (Ubuntu 20.04)
  Primary Function: Web Server
  Applications:
    - nginx (Web Server)
    - PHP (Runtime)
    - WordPress (CMS)
  Risk Factors:
    - Publicly accessible
    - CMS detected (attack surface)
    - PHP version (check for CVEs)
```

See [Asset Profiling](/concepts/scan-pipeline#stage-5-asset-profiling) for details.

## Troubleshooting

### No Fingerprints Detected

```
Port 80 open, but service unknown
```

**Causes**:
- Custom/obscure service
- Banner stripped for security
- Probe timeout

**Solutions**:
1. Increase timeout: `fingerprint.timeout: 10s`
2. Add custom rule for service
3. Use manual banner grab: `nc target 80`

### Incorrect Fingerprint

```
Port 8080 identified as Tomcat, but actually Jetty
```

**Solutions**:
1. Check probe output: Review `artifacts/banners/`
2. Add higher-confidence rule for Jetty
3. Report false positive to Vulntor team

### Probe Timeouts

```
WARN Fingerprint probe timeout on 192.168.1.100:443
```

**Causes**:
- Slow network
- Rate limiting
- Firewall interference

**Solutions**:
1. Increase timeout: `fingerprint.timeout: 15s`
2. Reduce concurrency: `fingerprint.probe_concurrency: 5`
3. Retry failed probes: `fingerprint.retry: 2`

## Next Steps

- [Scan Pipeline](/concepts/scan-pipeline) - How fingerprinting fits in the pipeline
- [Custom Fingerprints](/advanced/custom-fingerprints) - Writing custom rules
- [Module System](/concepts/modules) - Fingerprint module internals
- [Vulnerability Assessment](/guides/vulnerability-assessment) - Using fingerprints for CVE matching

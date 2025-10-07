---
slug: compliance-automation
title: "Automating Security Compliance with Pentora"
authors: [pentora_team]
tags: [compliance, security, best-practices, tutorial]
---

Continuous compliance monitoring is essential for modern organizations. Learn how Pentora automates compliance checks for CIS, PCI-DSS, NIST, and custom frameworks.

<!-- truncate -->

## The Compliance Challenge

Security compliance is more than a checkbox exercise. Organizations face:

- **Multiple Frameworks**: PCI-DSS for payments, HIPAA for healthcare, SOC 2 for SaaS
- **Continuous Monitoring**: Point-in-time audits aren't enough
- **Evidence Collection**: Auditors need proof of controls
- **Remediation Tracking**: Finding issues is only the first step

Traditional approaches rely on manual checklists, spreadsheets, and ad-hoc scans. This doesn't scale.

## Pentora's Compliance Engine

Pentora provides **built-in compliance frameworks** with automated scanning and reporting:

```bash
# Scan against PCI-DSS requirements
pentora scan 192.168.1.0/24 --compliance pci-dss

# Multiple frameworks
pentora scan 192.168.1.0/24 --compliance pci-dss,cis,nist

# Generate compliance report
pentora scan 192.168.1.0/24 \
  --compliance pci-dss \
  --format pdf \
  -o pci-compliance-2025-08.pdf
```

## Supported Frameworks

### 1. PCI-DSS (Payment Card Industry)

**Key Requirements Automated:**

| Requirement | Check | Pentora Detection |
|-------------|-------|-------------------|
| 2.2.4 | Configure system security parameters | Checks for default passwords, unnecessary services |
| 2.3 | Encrypt non-console admin access | Detects unencrypted admin protocols (Telnet, HTTP) |
| 6.2 | Ensure all systems protected from known vulnerabilities | CVE matching against detected services |
| 11.1 | Test for presence of wireless access points | Detects rogue access points |

Example scan:

```bash
pentora scan cardholder-env.example.com \
  --compliance pci-dss \
  --ports all \
  --vuln
```

Output:

```
=== PCI-DSS Compliance Report ===

Requirement 2.2.4: System Security Parameters
  ✓ No default passwords detected
  ✗ FAIL: Unnecessary services detected
    - Host 192.168.1.10: Telnet (port 23)
    - Host 192.168.1.15: FTP (port 21)

Requirement 2.3: Encrypt Admin Access
  ✗ FAIL: Unencrypted admin access detected
    - Host 192.168.1.20: HTTP admin panel (port 8080)
    - Host 192.168.1.22: Telnet (port 23)

Requirement 6.2: Vulnerability Protection
  ✗ FAIL: 12 known vulnerabilities detected
    - CVE-2021-44228 (Log4Shell) on 192.168.1.30
    - CVE-2022-22965 (Spring4Shell) on 192.168.1.31

Overall: 18/25 checks passed (72%)
Status: NON-COMPLIANT
```

### 2. CIS Benchmarks

**Coverage:**

- CIS Critical Security Controls
- OS-specific benchmarks (Ubuntu, RHEL, Windows Server)
- Application benchmarks (Apache, nginx, MySQL)

Example:

```yaml
# cis-profile.yaml
compliance:
  framework: cis-controls-v8
  controls:
    - id: "4.1"
      name: "Establish and Maintain Secure Configuration"
      checks:
        - no_default_passwords
        - tls_enabled
        - weak_ciphers_disabled
```

Run scan:

```bash
pentora scan web-servers.example.com \
  --compliance-profile ./cis-profile.yaml
```

### 3. NIST Cybersecurity Framework

Maps findings to NIST CSF categories:

```
Identify (ID):
  ✓ ID.AM-1: Physical devices and systems cataloged
  ✓ ID.AM-2: Software platforms and applications cataloged

Protect (PR):
  ✗ PR.AC-3: Remote access is managed (unencrypted found)
  ✗ PR.DS-2: Data-in-transit is protected (HTTP detected)

Detect (DE):
  ✓ DE.CM-7: Monitoring for unauthorized activity

Respond (RS):
  ⚠ RS.AN-1: Notifications from detection systems
```

### 4. HIPAA (Healthcare)

**Security Rule Requirements:**

```bash
pentora scan healthcare-network.example.com \
  --compliance hipaa \
  --include-phi-systems
```

Checks:

- §164.312(a)(1): Access Controls
- §164.312(e)(1): Transmission Security (encryption)
- §164.312(b): Audit Controls (logging capabilities)

## Custom Compliance Policies

Define organization-specific policies:

```yaml
# custom-compliance.yaml
name: "ACME Corp Security Policy v2.0"
version: "2.0"
description: "Internal security requirements"

policies:
  - id: "ACME-001"
    name: "No Legacy Protocols"
    severity: high
    checks:
      - type: port
        condition: not_open
        ports: [21, 23, 25, 110, 143]
        message: "Legacy protocols prohibited"

  - id: "ACME-002"
    name: "TLS 1.3 Required"
    severity: high
    checks:
      - type: tls_version
        condition: minimum
        version: "1.3"
        message: "TLS 1.3 or higher required"

  - id: "ACME-003"
    name: "No Expired Certificates"
    severity: critical
    checks:
      - type: certificate
        condition: not_expired
        warning_days: 30
        message: "SSL certificate expired or expiring soon"

  - id: "ACME-004"
    name: "Database Encryption Required"
    severity: critical
    checks:
      - type: service
        services: [mysql, postgresql, mongodb]
        require_encryption: true
        message: "Database must use encrypted connections"
```

Run custom policy:

```bash
pentora scan 192.168.1.0/24 \
  --compliance-policy ./custom-compliance.yaml \
  --format json \
  -o compliance-results.json
```

## CI/CD Integration

### GitHub Actions

```yaml
# .github/workflows/compliance-scan.yml
name: Weekly Compliance Scan

on:
  schedule:
    - cron: '0 2 * * 1'  # Every Monday at 2 AM

jobs:
  compliance:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Install Pentora
        run: |
          curl -sSL https://pentora.io/install.sh | bash

      - name: Run PCI-DSS Scan
        run: |
          pentora scan ${{ secrets.PRODUCTION_NETWORK }} \
            --compliance pci-dss \
            --format json \
            -o compliance-report.json

      - name: Check Compliance Status
        run: |
          status=$(jq -r '.compliance.status' compliance-report.json)
          if [ "$status" != "COMPLIANT" ]; then
            echo "::error::Compliance check failed"
            exit 1
          fi

      - name: Upload Report
        uses: actions/upload-artifact@v3
        with:
          name: compliance-report
          path: compliance-report.json
```

### GitLab CI

```yaml
# .gitlab-ci.yml
compliance_scan:
  stage: security
  image: pentora/pentora:latest
  script:
    - pentora scan $PRODUCTION_NETWORK
        --compliance pci-dss,hipaa
        --format pdf
        -o compliance-report-$CI_COMMIT_SHORT_SHA.pdf
  artifacts:
    paths:
      - compliance-report-*.pdf
    expire_in: 1 year
  only:
    - schedules
```

## Continuous Monitoring

Set up scheduled scans:

```bash
# crontab -e
# Daily compliance scan at 2 AM
0 2 * * * pentora scan 192.168.1.0/24 \
  --compliance pci-dss \
  --format json \
  -o /var/compliance/daily-$(date +\%Y\%m\%d).json

# Weekly comprehensive scan on Sundays
0 3 * * 0 pentora scan 192.168.1.0/24 \
  --compliance pci-dss,cis,nist \
  --ports all \
  --vuln \
  --format pdf \
  -o /var/compliance/weekly-$(date +\%Y\%m\%d).pdf
```

## Trend Analysis

Track compliance posture over time:

```bash
# Compare with baseline
pentora workspace compare \
  --baseline scan-2025-01-01 \
  --current scan-2025-08-20 \
  --compliance pci-dss
```

Output:

```
=== Compliance Trend Analysis ===

Overall Score:
  Baseline (2025-01-01): 68%
  Current (2025-08-20):  85%
  Change: +17% ✓

New Issues:
  - 2 expired SSL certificates
  - 1 new vulnerable service (CVE-2025-XXXX)

Resolved Issues:
  - 8 legacy protocols removed
  - 5 vulnerabilities patched
  - TLS 1.3 enabled on all web servers

Recommendations:
  1. Renew SSL certificates on 192.168.1.10, 192.168.1.15
  2. Patch CVE-2025-XXXX on 192.168.1.30
```

## Integration with Ticketing Systems

Auto-create tickets for compliance failures:

```bash
# Scan and create Jira tickets
pentora scan 192.168.1.0/24 \
  --compliance pci-dss \
  --on-fail create-jira-ticket \
  --jira-project SECURITY \
  --jira-labels compliance,pci-dss
```

Created ticket:

```
Title: [PCI-DSS] Requirement 2.3 Failure - Unencrypted Admin Access

Description:
Compliance scan detected unencrypted administrative access:

Host: 192.168.1.20
Port: 8080
Service: HTTP (Tomcat Manager)
Requirement: PCI-DSS 2.3 - Encrypt all non-console administrative access

Remediation:
1. Enable HTTPS on Tomcat Manager
2. Disable HTTP access on port 8080
3. Verify with: pentora scan 192.168.1.20 --ports 8080,8443

Scan ID: scan-2025-08-20-001
Detected: 2025-08-20 02:15:30 UTC
```

## Audit Evidence Collection

Generate audit-ready reports:

```bash
pentora scan production-network.example.com \
  --compliance pci-dss \
  --format pdf \
  --include-evidence \
  --include-screenshots \
  -o PCI-DSS-Q3-2025-Audit.pdf
```

Report includes:

- Executive summary with pass/fail counts
- Detailed findings with evidence
- Screenshots of vulnerable services
- Remediation recommendations
- Historical comparison charts
- Scan metadata and timestamps

## Best Practices

### 1. Schedule Regular Scans

```bash
# Development: Daily
# Staging: Weekly
# Production: Weekly + Pre-deployment
```

### 2. Use Compliance Profiles

```yaml
# prod-compliance.yaml
scan:
  targets: "{{ prod_network }}"
  compliance:
    - pci-dss
    - soc2
  vuln: true
  ports: all

notifications:
  on_failure:
    - slack: "#security-alerts"
    - email: "security-team@example.com"
    - jira:
        project: SECURITY
        assignee: "security-lead"
```

### 3. Track Remediation

```bash
# Scan → Fix → Verify cycle
pentora scan 192.168.1.0/24 --compliance pci-dss -o scan1.json
# Fix issues...
pentora scan 192.168.1.0/24 --compliance pci-dss -o scan2.json
pentora workspace diff scan1.json scan2.json --compliance-only
```

### 4. Automate Evidence Collection

Store reports in version control or S3 for audit trails:

```bash
#!/bin/bash
DATE=$(date +%Y-%m-%d)
pentora scan production.example.com \
  --compliance pci-dss \
  --format json \
  -o compliance-$DATE.json

# Upload to S3 for audit trail
aws s3 cp compliance-$DATE.json \
  s3://audit-evidence/pentora/compliance/$DATE/

# Commit to git
git add compliance-$DATE.json
git commit -m "Compliance scan: $DATE"
git push
```

## Conclusion

Compliance automation isn't optional—it's a necessity. Pentora provides:

- ✅ Built-in frameworks (PCI-DSS, CIS, NIST, HIPAA)
- ✅ Custom policy support
- ✅ CI/CD integration
- ✅ Automated reporting and evidence collection
- ✅ Trend analysis and remediation tracking

Start automating today:

```bash
pentora scan <your-network> --compliance pci-dss
```

---

**Resources:**
- [Compliance Guide](/docs/guides/compliance-checks)
- [Custom Policies](/docs/advanced/compliance-policies)
- [CI/CD Examples](/docs/guides/cicd-integration)

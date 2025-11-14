# Standalone CLI Deployment

Deploy Vulntor as a standalone CLI tool for direct command-line usage and ad-hoc scanning.

## Overview

Standalone deployment is ideal for:

- Security professionals performing manual assessments
- Automated scripts and cron jobs
- CI/CD pipeline integration
- Local workstation installations
- Quick one-off scans without infrastructure

The standalone mode operates without a persistent server daemon, executing scans directly from the command line.

## Installation

### Quick Install

#### Linux / macOS

```bash
# Download and install
curl -sSL https://vulntor.io/install.sh | bash

# Verify installation
vulntor version
```

#### Windows

```powershell
# Download installer
Invoke-WebRequest -Uri https://vulntor.io/install.ps1 -OutFile install.ps1

# Run installer
.\install.ps1

# Verify installation
vulntor version
```

### Manual Binary Installation

#### Linux (amd64)

```bash
# Download latest release
curl -LO https://github.com/vulntor/vulntor/releases/latest/download/vulntor-linux-amd64.tar.gz

# Extract
tar -xzf vulntor-linux-amd64.tar.gz

# Install to system path
sudo mv vulntor /usr/local/bin/

# Set executable permissions
sudo chmod +x /usr/local/bin/vulntor

# Verify
vulntor version
```

#### macOS (amd64)

```bash
# Download
curl -LO https://github.com/vulntor/vulntor/releases/latest/download/vulntor-darwin-amd64.tar.gz

# Extract and install
tar -xzf vulntor-darwin-amd64.tar.gz
sudo mv vulntor /usr/local/bin/
sudo chmod +x /usr/local/bin/vulntor

# macOS may require security approval
sudo xattr -d com.apple.quarantine /usr/local/bin/vulntor

# Verify
vulntor version
```

#### macOS (arm64 - Apple Silicon)

```bash
# Download ARM64 version
curl -LO https://github.com/vulntor/vulntor/releases/latest/download/vulntor-darwin-arm64.tar.gz

# Extract and install
tar -xzf vulntor-darwin-arm64.tar.gz
sudo mv vulntor /usr/local/bin/
sudo chmod +x /usr/local/bin/vulntor
sudo xattr -d com.apple.quarantine /usr/local/bin/vulntor

# Verify
vulntor version
```

#### Windows (Manual)

```powershell
# Download
Invoke-WebRequest -Uri https://github.com/vulntor/vulntor/releases/latest/download/vulntor-windows-amd64.zip -OutFile vulntor.zip

# Extract
Expand-Archive vulntor.zip -DestinationPath "C:\Program Files\Vulntor"

# Add to PATH (requires Administrator)
[Environment]::SetEnvironmentVariable(
    "Path",
    $env:Path + ";C:\Program Files\Vulntor",
    "Machine"
)

# Verify (restart terminal)
vulntor version
```

### Package Manager Installation

#### Debian / Ubuntu (APT)

```bash
# Add repository
curl -fsSL https://vulntor.io/gpg.key | sudo gpg --dearmor -o /usr/share/keyrings/vulntor-archive-keyring.gpg

echo "deb [signed-by=/usr/share/keyrings/vulntor-archive-keyring.gpg] https://apt.vulntor.io stable main" | \
  sudo tee /etc/apt/sources.list.d/vulntor.list

# Install
sudo apt update
sudo apt install vulntor

# Verify
vulntor version
```

#### RHEL / CentOS / Fedora (YUM/DNF)

```bash
# Add repository
sudo tee /etc/yum.repos.d/vulntor.repo <<EOF
[vulntor]
name=Vulntor Repository
baseurl=https://yum.vulntor.io/stable
enabled=1
gpgcheck=1
gpgkey=https://vulntor.io/gpg.key
EOF

# Install
sudo dnf install vulntor
# or
sudo yum install vulntor

# Verify
vulntor version
```

#### Homebrew (macOS)

```bash
# Add tap
brew tap vulntor/tap

# Install
brew install vulntor

# Verify
vulntor version
```

## Initial Configuration

### Storage Setup

Vulntor uses a storage directory to store scan results:

```bash
# Default storage locations:
# Linux: ~/.local/share/vulntor
# macOS: ~/Library/Application Support/Vulntor
# Windows: %AppData%\Vulntor

# Custom storage location
export VULNTOR_STORAGE_DIR=/data/vulntor-scans
```

### Configuration File

Create user configuration:

```bash
# Create config directory
mkdir -p ~/.config/vulntor

# Generate default config
vulntor config init > ~/.config/vulntor/config.yaml
```

Edit `~/.config/vulntor/config.yaml`:

```yaml
storage:
  dir: ~/.local/share/vulntor
  enabled: true
  retention:
    max_age: 90d
    max_scans: 1000

scanner:
  default_profile: standard
  rate: 1000
  concurrency: 100
  timeout: 3s

logging:
  level: info
  format: text
  output: stderr

fingerprint:
  cache:
    auto_sync: true
    ttl: 7d
```

### Permissions Setup

#### Linux: Set Capabilities (Recommended)

Allow raw socket access without sudo:

```bash
# Set capabilities
sudo setcap cap_net_raw,cap_net_admin+eip /usr/local/bin/vulntor

# Verify
getcap /usr/local/bin/vulntor
```

This allows:

- SYN scanning without root
- ICMP ping without root
- ARP discovery without root

#### Alternative: Use Sudo

```bash
# Run scans with sudo
sudo vulntor scan 192.168.1.0/24
```

#### Windows: Administrator Access

Run PowerShell/CMD as Administrator for full scanning capabilities.

## Basic Usage

### Simple Scan

```bash
# Scan single host
vulntor scan 192.168.1.100

# Scan network range
vulntor scan 192.168.1.0/24

# Scan with specific ports
vulntor scan 192.168.1.100 --ports 22,80,443,8080

# Scan from file
vulntor scan --target-file targets.txt
```

### Scan Profiles

```bash
# Quick scan (fast, top 100 ports)
vulntor scan 192.168.1.0/24 --profile quick

# Standard scan (balanced, top 1000 ports)
vulntor scan 192.168.1.0/24 --profile standard

# Deep scan (comprehensive, all ports)
vulntor scan 192.168.1.0/24 --profile deep
```

### Discovery-Only Mode

```bash
# Only discover live hosts
vulntor scan 10.0.0.0/16 --only-discover -o live-hosts.txt

# Skip discovery for known hosts
vulntor scan --target-file live-hosts.txt --no-discover
```

### Vulnerability Assessment

```bash
# Scan with vulnerability detection
vulntor scan 192.168.1.100 --vuln

# Vulnerability scan with severity filter
vulntor scan 192.168.1.100 --vuln --min-severity high
```

### Output Options

```bash
# JSON output
vulntor scan 192.168.1.100 -o results.json --format json

# CSV output
vulntor scan 192.168.1.100 -o results.csv --format csv

# PDF report
vulntor scan 192.168.1.100 -o report.pdf --format pdf

# Multiple formats
vulntor scan 192.168.1.100 -o results --format json,csv,pdf
```

## Advanced Configuration

### Custom Scan Profile

Create `~/.config/vulntor/profiles/custom.yaml`:

```yaml
name: custom
discovery:
  timeout: 2s
  retry: 2
  icmp:
    enabled: true
    count: 2

scanner:
  rate: 2000
  timeout: 5s
  concurrency: 200
  ports:
    - 22
    - 80
    - 443
    - 3306
    - 5432
    - 8080
    - 8443

fingerprint:
  enabled: true
  probe_timeout: 5s
  max_protocols: 5

vulnerability:
  enabled: true
  min_severity: medium
```

Use custom profile:

```bash
vulntor scan 192.168.1.0/24 --profile custom
```

### Rate Limiting

```bash
# Conservative rate (production networks)
vulntor scan 192.168.1.0/24 --rate 100 --concurrency 10

# Aggressive rate (lab environments)
vulntor scan 192.168.1.0/24 --rate 5000 --concurrency 200

# Timeout configuration
vulntor scan 192.168.1.0/24 --timeout 5s --retry 2
```

### Exclusions

```bash
# Exclude specific hosts
vulntor scan 192.168.1.0/24 --exclude 192.168.1.1,192.168.1.2

# Exclude from file
vulntor scan 192.168.1.0/24 --exclude-file sensitive-hosts.txt

# Exclude ports
vulntor scan 192.168.1.0/24 --exclude-ports 25,465,587
```

## Storage Management

### View Scans

```bash
# List all scans
vulntor storage list

# List recent scans
vulntor storage list --limit 10

# Show specific scan
vulntor storage show <scan-id>

# Export scan results
vulntor storage export <scan-id> -o results.json
```

### Cleanup

```bash
# Remove old scans
vulntor storage gc --older-than 30d

# Remove specific scan
vulntor storage delete <scan-id>

# Check storage size
vulntor storage info

# Validate storage integrity
vulntor storage check
```

### Statistics

```bash
# Show storage statistics
vulntor storage stats

# Example output:
# Total scans: 145
# Total targets: 5,234
# Total findings: 1,823
# Storage size: 2.3 GB
# Oldest scan: 2024-01-15
# Newest scan: 2024-10-06
```

## Automation

### Cron Jobs

Create `/etc/cron.d/vulntor`:

```bash
# Daily network scan at 2 AM
0 2 * * * vulntor vulntor scan --target-file /etc/vulntor/targets.txt --profile standard -o /var/log/vulntor/scan-$(date +\%Y\%m\%d).json

# Weekly full scan on Sunday at 1 AM
0 1 * * 0 vulntor vulntor scan --target-file /etc/vulntor/all-hosts.txt --profile deep --vuln
```

Or use crontab:

```bash
crontab -e

# Add:
0 2 * * * /usr/local/bin/vulntor scan 192.168.1.0/24 -o ~/scans/daily-$(date +\%Y\%m\%d).json
```

### Shell Scripts

Create `scan-network.sh`:

```bash
#!/bin/bash
set -euo pipefail

TARGETS="/etc/vulntor/targets.txt"
OUTPUT_DIR="/var/vulntor/scans"
DATE=$(date +%Y%m%d-%H%M%S)

echo "Starting Vulntor scan at $(date)"

# Run scan
vulntor scan --target-file "$TARGETS" \
    --profile standard \
    --vuln \
    -o "$OUTPUT_DIR/scan-$DATE.json" \
    --format json

# Check for critical vulnerabilities
CRITICAL=$(jq '[.findings[] | select(.severity == "critical")] | length' "$OUTPUT_DIR/scan-$DATE.json")

if [ "$CRITICAL" -gt 0 ]; then
    echo "ALERT: $CRITICAL critical vulnerabilities found!"
    # Send alert
    mail -s "Vulntor: Critical Vulnerabilities Detected" security@company.com < "$OUTPUT_DIR/scan-$DATE.json"
fi

echo "Scan completed at $(date)"
```

Make executable and run:

```bash
chmod +x scan-network.sh
./scan-network.sh
```

### CI/CD Integration

#### GitHub Actions

Create `.github/workflows/security-scan.yml`:

```yaml
name: Security Scan

on:
  schedule:
    - cron: '0 2 * * *' # Daily at 2 AM
  workflow_dispatch:

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - name: Install Vulntor
        run: |
          curl -sSL https://vulntor.io/install.sh | bash
          vulntor version

      - name: Run Security Scan
        run: |
          vulntor scan ${{ secrets.SCAN_TARGETS }} \
            --profile standard \
            --vuln \
            -o scan-results.json \
            --format json

      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: scan-results
          path: scan-results.json

      - name: Check for Critical Vulnerabilities
        run: |
          CRITICAL=$(jq '[.findings[] | select(.severity == "critical")] | length' scan-results.json)
          if [ "$CRITICAL" -gt 0 ]; then
            echo "::error::Found $CRITICAL critical vulnerabilities"
            exit 1
          fi
```

#### GitLab CI

Create `.gitlab-ci.yml`:

```yaml
security_scan:
  stage: test
  image: ubuntu:latest
  before_script:
    - curl -sSL https://vulntor.io/install.sh | bash
  script:
    - vulntor scan $SCAN_TARGETS --profile standard --vuln -o results.json
    - jq . results.json
  artifacts:
    reports:
      junit: results.json
    paths:
      - results.json
  only:
    - schedules
```

#### Jenkins Pipeline

Create `Jenkinsfile`:

```groovy
pipeline {
    agent any

    stages {
        stage('Install Vulntor') {
            steps {
                sh 'curl -sSL https://vulntor.io/install.sh | bash'
            }
        }

        stage('Security Scan') {
            steps {
                sh '''
                    vulntor scan ${SCAN_TARGETS} \
                        --profile standard \
                        --vuln \
                        -o scan-results.json \
                        --format json
                '''
            }
        }

        stage('Analyze Results') {
            steps {
                script {
                    def results = readJSON file: 'scan-results.json'
                    def critical = results.findings.findAll { it.severity == 'critical' }.size()

                    if (critical > 0) {
                        error("Found ${critical} critical vulnerabilities")
                    }
                }
            }
        }
    }

    post {
        always {
            archiveArtifacts artifacts: 'scan-results.json', fingerprint: true
        }
    }
}
```

## Environment Variables

Configure Vulntor via environment variables:

```bash
# Storage directory
export VULNTOR_STORAGE_DIR=/data/vulntor

# Configuration file
export VULNTOR_CONFIG=/etc/vulntor/config.yaml

# Log level
export VULNTOR_LOG_LEVEL=debug

# Log format
export VULNTOR_LOG_FORMAT=json

# API token (for server integration)
export VULNTOR_API_TOKEN=your-token-here

# Default scan profile
export VULNTOR_PROFILE=standard

# Rate limiting
export VULNTOR_RATE=1000
export VULNTOR_CONCURRENCY=100

# Timeout
export VULNTOR_TIMEOUT=5s
```

Add to `~/.bashrc` or `~/.zshrc`:

```bash
# Vulntor configuration
export VULNTOR_STORAGE_DIR=~/vulntor-storage
export VULNTOR_LOG_LEVEL=info
export VULNTOR_PROFILE=standard
```

## Troubleshooting

### Permission Denied

```bash
# Solution 1: Set capabilities
sudo setcap cap_net_raw,cap_net_admin+eip /usr/local/bin/vulntor

# Solution 2: Use sudo
sudo vulntor scan 192.168.1.0/24

# Solution 3: Use connect scan (no raw sockets)
vulntor scan 192.168.1.0/24 --scan-type connect
```

### Command Not Found

```bash
# Add to PATH
export PATH=$PATH:/usr/local/bin

# Verify binary location
which vulntor
ls -l /usr/local/bin/vulntor

# Make executable
chmod +x /usr/local/bin/vulntor
```

### Slow Scans

```bash
# Increase rate and concurrency
vulntor scan 192.168.1.0/24 --rate 2000 --concurrency 200

# Use quick profile
vulntor scan 192.168.1.0/24 --profile quick

# Skip unnecessary phases
vulntor scan 192.168.1.0/24 --no-vuln --no-fingerprint
```

### No Hosts Discovered

```bash
# Use TCP-based discovery
vulntor scan 192.168.1.0/24 --discover-profile tcp

# Skip discovery for known hosts
vulntor scan 192.168.1.100 --no-discover

# Verify network connectivity
ping 192.168.1.100
```

### Storage Issues

```bash
# Check storage integrity
vulntor storage check

# Fix corrupted storage
vulntor storage check --fix

# Clean old scans
vulntor storage gc --older-than 7d

# Check disk space
df -h ~/.local/share/vulntor
```

## Upgrading

### Package Manager

```bash
# APT
sudo apt update && sudo apt upgrade vulntor

# YUM/DNF
sudo yum update vulntor

# Homebrew
brew upgrade vulntor
```

### Manual Upgrade

```bash
# Download latest version
curl -sSL https://vulntor.io/install.sh | bash

# Verify upgrade
vulntor version

# Check for updates
vulntor version --check-updates
```

### Backup Before Upgrade

```bash
# Backup storage
tar -czf vulntor-backup-$(date +%Y%m%d).tar.gz ~/.local/share/vulntor

# Backup configuration
cp -r ~/.config/vulntor ~/vulntor-config-backup
```

## Uninstallation

### Package Manager

```bash
# APT
sudo apt remove vulntor

# YUM/DNF
sudo yum remove vulntor

# Homebrew
brew uninstall vulntor
```

### Manual Removal

```bash
# Remove binary
sudo rm /usr/local/bin/vulntor

# Remove configuration
rm -rf ~/.config/vulntor

# Remove storage (optional - contains scan results)
rm -rf ~/.local/share/vulntor  # Linux
rm -rf ~/Library/Application\ Support/Vulntor  # macOS
rm -rf %AppData%\Vulntor  # Windows
```

## Security Considerations

### Privileged Operations

- SYN scanning requires raw socket access (root or CAP_NET_RAW)
- ICMP discovery requires ICMP socket access (root or CAP_NET_RAW)
- ARP discovery requires raw socket access (root or CAP_NET_RAW)
- Connect scanning works without privileges but is slower

### Network Security

```bash
# Rate limit to avoid detection/disruption
vulntor scan 192.168.1.0/24 --rate 500 --concurrency 50

# Scan during maintenance windows
vulntor scan prod-network.txt --schedule "0 2 * * *"

# Use TCP discovery in strict environments
vulntor scan 192.168.1.0/24 --discover-profile tcp
```

### Data Security

```bash
# Encrypt sensitive scan results
gpg --encrypt --recipient security@company.com results.json

# Secure storage permissions
chmod 700 ~/.local/share/vulntor

# Disable storage for stateless scanning
vulntor scan 192.168.1.0/24 --no-storage -o results.json
```

## Next Steps

- [Server Mode Deployment](/deployment/server-mode) - Deploy as persistent service
- [Docker Deployment](/deployment/docker) - Containerized deployment
- [Scan Profiles](/configuration/scan-profiles) - Customize scan behavior
- [Network Scanning Guide](/guides/network-scanning) - Best practices
- [CLI Reference](/cli/scan) - Complete command reference

# Standalone CLI Deployment

Deploy Pentora as a standalone CLI tool for direct command-line usage and ad-hoc scanning.

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
curl -sSL https://pentora.io/install.sh | bash

# Verify installation
pentora version
```

#### Windows

```powershell
# Download installer
Invoke-WebRequest -Uri https://pentora.io/install.ps1 -OutFile install.ps1

# Run installer
.\install.ps1

# Verify installation
pentora version
```

### Manual Binary Installation

#### Linux (amd64)

```bash
# Download latest release
curl -LO https://github.com/pentora-ai/pentora/releases/latest/download/pentora-linux-amd64.tar.gz

# Extract
tar -xzf pentora-linux-amd64.tar.gz

# Install to system path
sudo mv pentora /usr/local/bin/

# Set executable permissions
sudo chmod +x /usr/local/bin/pentora

# Verify
pentora version
```

#### macOS (amd64)

```bash
# Download
curl -LO https://github.com/pentora-ai/pentora/releases/latest/download/pentora-darwin-amd64.tar.gz

# Extract and install
tar -xzf pentora-darwin-amd64.tar.gz
sudo mv pentora /usr/local/bin/
sudo chmod +x /usr/local/bin/pentora

# macOS may require security approval
sudo xattr -d com.apple.quarantine /usr/local/bin/pentora

# Verify
pentora version
```

#### macOS (arm64 - Apple Silicon)

```bash
# Download ARM64 version
curl -LO https://github.com/pentora-ai/pentora/releases/latest/download/pentora-darwin-arm64.tar.gz

# Extract and install
tar -xzf pentora-darwin-arm64.tar.gz
sudo mv pentora /usr/local/bin/
sudo chmod +x /usr/local/bin/pentora
sudo xattr -d com.apple.quarantine /usr/local/bin/pentora

# Verify
pentora version
```

#### Windows (Manual)

```powershell
# Download
Invoke-WebRequest -Uri https://github.com/pentora-ai/pentora/releases/latest/download/pentora-windows-amd64.zip -OutFile pentora.zip

# Extract
Expand-Archive pentora.zip -DestinationPath "C:\Program Files\Pentora"

# Add to PATH (requires Administrator)
[Environment]::SetEnvironmentVariable(
    "Path",
    $env:Path + ";C:\Program Files\Pentora",
    "Machine"
)

# Verify (restart terminal)
pentora version
```

### Package Manager Installation

#### Debian / Ubuntu (APT)

```bash
# Add repository
curl -fsSL https://pentora.io/gpg.key | sudo gpg --dearmor -o /usr/share/keyrings/pentora-archive-keyring.gpg

echo "deb [signed-by=/usr/share/keyrings/pentora-archive-keyring.gpg] https://apt.pentora.io stable main" | \
  sudo tee /etc/apt/sources.list.d/pentora.list

# Install
sudo apt update
sudo apt install pentora

# Verify
pentora version
```

#### RHEL / CentOS / Fedora (YUM/DNF)

```bash
# Add repository
sudo tee /etc/yum.repos.d/pentora.repo <<EOF
[pentora]
name=Pentora Repository
baseurl=https://yum.pentora.io/stable
enabled=1
gpgcheck=1
gpgkey=https://pentora.io/gpg.key
EOF

# Install
sudo dnf install pentora
# or
sudo yum install pentora

# Verify
pentora version
```

#### Homebrew (macOS)

```bash
# Add tap
brew tap pentora/tap

# Install
brew install pentora

# Verify
pentora version
```

## Initial Configuration

### Workspace Setup

Pentora uses a workspace directory to store scan results:

```bash
# Initialize workspace (default location)
pentora workspace init

# Default locations:
# Linux: ~/.local/share/pentora
# macOS: ~/Library/Application Support/Pentora
# Windows: %AppData%\Pentora

# Custom workspace location
export PENTORA_WORKSPACE_DIR=/data/pentora-scans
pentora workspace init
```

### Configuration File

Create user configuration:

```bash
# Create config directory
mkdir -p ~/.config/pentora

# Generate default config
pentora config init > ~/.config/pentora/config.yaml
```

Edit `~/.config/pentora/config.yaml`:

```yaml
workspace:
  dir: ~/.local/share/pentora
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
sudo setcap cap_net_raw,cap_net_admin+eip /usr/local/bin/pentora

# Verify
getcap /usr/local/bin/pentora
```

This allows:

- SYN scanning without root
- ICMP ping without root
- ARP discovery without root

#### Alternative: Use Sudo

```bash
# Run scans with sudo
sudo pentora scan 192.168.1.0/24
```

#### Windows: Administrator Access

Run PowerShell/CMD as Administrator for full scanning capabilities.

## Basic Usage

### Simple Scan

```bash
# Scan single host
pentora scan 192.168.1.100

# Scan network range
pentora scan 192.168.1.0/24

# Scan with specific ports
pentora scan 192.168.1.100 --ports 22,80,443,8080

# Scan from file
pentora scan --target-file targets.txt
```

### Scan Profiles

```bash
# Quick scan (fast, top 100 ports)
pentora scan 192.168.1.0/24 --profile quick

# Standard scan (balanced, top 1000 ports)
pentora scan 192.168.1.0/24 --profile standard

# Deep scan (comprehensive, all ports)
pentora scan 192.168.1.0/24 --profile deep
```

### Discovery-Only Mode

```bash
# Only discover live hosts
pentora scan 10.0.0.0/16 --only-discover -o live-hosts.txt

# Skip discovery for known hosts
pentora scan --target-file live-hosts.txt --no-discover
```

### Vulnerability Assessment

```bash
# Scan with vulnerability detection
pentora scan 192.168.1.100 --vuln

# Vulnerability scan with severity filter
pentora scan 192.168.1.100 --vuln --min-severity high
```

### Output Options

```bash
# JSON output
pentora scan 192.168.1.100 -o results.json --format json

# CSV output
pentora scan 192.168.1.100 -o results.csv --format csv

# PDF report
pentora scan 192.168.1.100 -o report.pdf --format pdf

# Multiple formats
pentora scan 192.168.1.100 -o results --format json,csv,pdf
```

## Advanced Configuration

### Custom Scan Profile

Create `~/.config/pentora/profiles/custom.yaml`:

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
pentora scan 192.168.1.0/24 --profile custom
```

### Rate Limiting

```bash
# Conservative rate (production networks)
pentora scan 192.168.1.0/24 --rate 100 --concurrency 10

# Aggressive rate (lab environments)
pentora scan 192.168.1.0/24 --rate 5000 --concurrency 200

# Timeout configuration
pentora scan 192.168.1.0/24 --timeout 5s --retry 2
```

### Exclusions

```bash
# Exclude specific hosts
pentora scan 192.168.1.0/24 --exclude 192.168.1.1,192.168.1.2

# Exclude from file
pentora scan 192.168.1.0/24 --exclude-file sensitive-hosts.txt

# Exclude ports
pentora scan 192.168.1.0/24 --exclude-ports 25,465,587
```

## Workspace Management

### View Scans

```bash
# List all scans
pentora workspace list

# List recent scans
pentora workspace list --limit 10

# Show specific scan
pentora workspace show <scan-id>

# Export scan results
pentora workspace export <scan-id> -o results.json
```

### Cleanup

```bash
# Remove old scans
pentora workspace gc --older-than 30d

# Remove specific scan
pentora workspace delete <scan-id>

# Check workspace size
pentora workspace info

# Validate workspace integrity
pentora workspace check
```

### Statistics

```bash
# Show workspace statistics
pentora workspace stats

# Example output:
# Total scans: 145
# Total targets: 5,234
# Total findings: 1,823
# Workspace size: 2.3 GB
# Oldest scan: 2024-01-15
# Newest scan: 2024-10-06
```

## Automation

### Cron Jobs

Create `/etc/cron.d/pentora`:

```bash
# Daily network scan at 2 AM
0 2 * * * pentora pentora scan --target-file /etc/pentora/targets.txt --profile standard -o /var/log/pentora/scan-$(date +\%Y\%m\%d).json

# Weekly full scan on Sunday at 1 AM
0 1 * * 0 pentora pentora scan --target-file /etc/pentora/all-hosts.txt --profile deep --vuln
```

Or use crontab:

```bash
crontab -e

# Add:
0 2 * * * /usr/local/bin/pentora scan 192.168.1.0/24 -o ~/scans/daily-$(date +\%Y\%m\%d).json
```

### Shell Scripts

Create `scan-network.sh`:

```bash
#!/bin/bash
set -euo pipefail

TARGETS="/etc/pentora/targets.txt"
OUTPUT_DIR="/var/pentora/scans"
DATE=$(date +%Y%m%d-%H%M%S)

echo "Starting Pentora scan at $(date)"

# Run scan
pentora scan --target-file "$TARGETS" \
    --profile standard \
    --vuln \
    -o "$OUTPUT_DIR/scan-$DATE.json" \
    --format json

# Check for critical vulnerabilities
CRITICAL=$(jq '[.findings[] | select(.severity == "critical")] | length' "$OUTPUT_DIR/scan-$DATE.json")

if [ "$CRITICAL" -gt 0 ]; then
    echo "ALERT: $CRITICAL critical vulnerabilities found!"
    # Send alert
    mail -s "Pentora: Critical Vulnerabilities Detected" security@company.com < "$OUTPUT_DIR/scan-$DATE.json"
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
      - name: Install Pentora
        run: |
          curl -sSL https://pentora.io/install.sh | bash
          pentora version

      - name: Run Security Scan
        run: |
          pentora scan ${{ secrets.SCAN_TARGETS }} \
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
    - curl -sSL https://pentora.io/install.sh | bash
  script:
    - pentora scan $SCAN_TARGETS --profile standard --vuln -o results.json
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
        stage('Install Pentora') {
            steps {
                sh 'curl -sSL https://pentora.io/install.sh | bash'
            }
        }

        stage('Security Scan') {
            steps {
                sh '''
                    pentora scan ${SCAN_TARGETS} \
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

Configure Pentora via environment variables:

```bash
# Workspace directory
export PENTORA_WORKSPACE_DIR=/data/pentora

# Configuration file
export PENTORA_CONFIG=/etc/pentora/config.yaml

# Log level
export PENTORA_LOG_LEVEL=debug

# Log format
export PENTORA_LOG_FORMAT=json

# API token (for server integration)
export PENTORA_API_TOKEN=your-token-here

# Default scan profile
export PENTORA_PROFILE=standard

# Rate limiting
export PENTORA_RATE=1000
export PENTORA_CONCURRENCY=100

# Timeout
export PENTORA_TIMEOUT=5s
```

Add to `~/.bashrc` or `~/.zshrc`:

```bash
# Pentora configuration
export PENTORA_WORKSPACE_DIR=~/pentora-workspace
export PENTORA_LOG_LEVEL=info
export PENTORA_PROFILE=standard
```

## Troubleshooting

### Permission Denied

```bash
# Solution 1: Set capabilities
sudo setcap cap_net_raw,cap_net_admin+eip /usr/local/bin/pentora

# Solution 2: Use sudo
sudo pentora scan 192.168.1.0/24

# Solution 3: Use connect scan (no raw sockets)
pentora scan 192.168.1.0/24 --scan-type connect
```

### Command Not Found

```bash
# Add to PATH
export PATH=$PATH:/usr/local/bin

# Verify binary location
which pentora
ls -l /usr/local/bin/pentora

# Make executable
chmod +x /usr/local/bin/pentora
```

### Slow Scans

```bash
# Increase rate and concurrency
pentora scan 192.168.1.0/24 --rate 2000 --concurrency 200

# Use quick profile
pentora scan 192.168.1.0/24 --profile quick

# Skip unnecessary phases
pentora scan 192.168.1.0/24 --no-vuln --no-fingerprint
```

### No Hosts Discovered

```bash
# Use TCP-based discovery
pentora scan 192.168.1.0/24 --discover-profile tcp

# Skip discovery for known hosts
pentora scan 192.168.1.100 --no-discover

# Verify network connectivity
ping 192.168.1.100
```

### Workspace Issues

```bash
# Check workspace integrity
pentora workspace check

# Fix corrupted workspace
pentora workspace check --fix

# Clean old scans
pentora workspace gc --older-than 7d

# Check disk space
df -h ~/.local/share/pentora
```

## Upgrading

### Package Manager

```bash
# APT
sudo apt update && sudo apt upgrade pentora

# YUM/DNF
sudo yum update pentora

# Homebrew
brew upgrade pentora
```

### Manual Upgrade

```bash
# Download latest version
curl -sSL https://pentora.io/install.sh | bash

# Verify upgrade
pentora version

# Check for updates
pentora version --check-updates
```

### Backup Before Upgrade

```bash
# Backup workspace
tar -czf pentora-backup-$(date +%Y%m%d).tar.gz ~/.local/share/pentora

# Backup configuration
cp -r ~/.config/pentora ~/pentora-config-backup
```

## Uninstallation

### Package Manager

```bash
# APT
sudo apt remove pentora

# YUM/DNF
sudo yum remove pentora

# Homebrew
brew uninstall pentora
```

### Manual Removal

```bash
# Remove binary
sudo rm /usr/local/bin/pentora

# Remove configuration
rm -rf ~/.config/pentora

# Remove workspace (optional - contains scan results)
rm -rf ~/.local/share/pentora  # Linux
rm -rf ~/Library/Application\ Support/Pentora  # macOS
rm -rf %AppData%\Pentora  # Windows
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
pentora scan 192.168.1.0/24 --rate 500 --concurrency 50

# Scan during maintenance windows
pentora scan prod-network.txt --schedule "0 2 * * *"

# Use TCP discovery in strict environments
pentora scan 192.168.1.0/24 --discover-profile tcp
```

### Data Security

```bash
# Encrypt sensitive scan results
gpg --encrypt --recipient security@company.com results.json

# Secure workspace permissions
chmod 700 ~/.local/share/pentora

# Disable workspace for stateless scanning
pentora scan 192.168.1.0/24 --no-workspace -o results.json
```

## Next Steps

- [Server Mode Deployment](/docs/deployment/server-mode) - Deploy as persistent service
- [Docker Deployment](/docs/deployment/docker) - Containerized deployment
- [Scan Profiles](/docs/configuration/scan-profiles) - Customize scan behavior
- [Network Scanning Guide](/docs/guides/network-scanning) - Best practices
- [CLI Reference](/docs/cli/scan) - Complete command reference

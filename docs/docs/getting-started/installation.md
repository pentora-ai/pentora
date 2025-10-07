---
sidebar_position: 1
---

# Installation

Pentora supports multiple installation methods across macOS, Linux, and Windows platforms.

## System Requirements

### Minimum Requirements

- **CPU**: 2 cores
- **RAM**: 2 GB
- **Disk**: 100 MB for binaries, additional space for workspace
- **OS**: macOS 10.15+, Linux (kernel 3.10+), Windows 10+

### Recommended for Large Scans

- **CPU**: 4+ cores
- **RAM**: 8+ GB
- **Disk**: 10 GB+ for workspace and scan results
- **Network**: Low-latency connection for optimal scanning

## Quick Install (Recommended)

### Linux / macOS

Use the official install script:

```bash
curl -sSL https://pentora.io/install.sh | bash
```

This script will:

1. Detect your platform and architecture
2. Download the latest release binary
3. Install to `/usr/local/bin/pentora`
4. Set appropriate permissions

Verify installation:

```bash
pentora version
```

### Manual Install Script

If you prefer to review the script first:

```bash
curl -sSL https://pentora.io/install.sh -o install.sh
chmod +x install.sh
./install.sh
```

## Platform-Specific Installation

### macOS

#### Homebrew (Coming Soon)

```bash
brew tap pentora/tap
brew install pentora
```

#### Download Binary

```bash
# Download latest release
curl -LO https://github.com/pentora-ai/pentora/releases/latest/download/pentora-darwin-amd64.tar.gz

# Extract
tar -xzf pentora-darwin-amd64.tar.gz

# Move to PATH
sudo mv pentora /usr/local/bin/

# Verify
pentora version
```

#### DMG Installer

1. Download `pentora-installer.dmg` from [releases page](https://github.com/pentora-ai/pentora/releases)
2. Open the DMG file
3. Drag Pentora to Applications folder
4. Run from Terminal or add to PATH

### Linux

#### Debian / Ubuntu (APT)

```bash
# Add Pentora repository
curl -fsSL https://pentora.io/gpg.key | sudo gpg --dearmor -o /usr/share/keyrings/pentora-archive-keyring.gpg

echo "deb [signed-by=/usr/share/keyrings/pentora-archive-keyring.gpg] https://apt.pentora.io stable main" | \
  sudo tee /etc/apt/sources.list.d/pentora.list

# Install
sudo apt update
sudo apt install pentora
```

#### RHEL / CentOS / Fedora (YUM/DNF)

```bash
# Add Pentora repository
sudo tee /etc/yum.repos.d/pentora.repo <<EOF
[pentora]
name=Pentora Repository
baseurl=https://yum.pentora.io/stable
enabled=1
gpgcheck=1
gpgkey=https://pentora.io/gpg.key
EOF

# Install
sudo yum install pentora
# or
sudo dnf install pentora
```

#### Arch Linux (AUR)

```bash
yay -S pentora-bin
# or
paru -S pentora-bin
```

#### Generic Linux Binary

```bash
# Download
curl -LO https://github.com/pentora-ai/pentora/releases/latest/download/pentora-linux-amd64.tar.gz

# Extract
tar -xzf pentora-linux-amd64.tar.gz

# Install
sudo mv pentora /usr/local/bin/

# Verify
pentora version
```

### Windows

#### Installer (MSI)

1. Download `pentora-installer-x64.msi` from [releases page](https://github.com/pentora-ai/pentora/releases)
2. Run the installer
3. Follow the installation wizard
4. Pentora will be added to PATH automatically

#### Chocolatey (Coming Soon)

```powershell
choco install pentora
```

#### Scoop

```powershell
scoop bucket add pentora https://github.com/pentora/scoop-bucket
scoop install pentora
```

#### Manual Binary

```powershell
# Download latest release
Invoke-WebRequest -Uri https://github.com/pentora-ai/pentora/releases/latest/download/pentora-windows-amd64.zip -OutFile pentora.zip

# Extract
Expand-Archive pentora.zip -DestinationPath C:\Program Files\Pentora

# Add to PATH (run as Administrator)
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\Program Files\Pentora", "Machine")

# Verify
pentora version
```

## Build from Source

### Prerequisites

- Go 1.21 or higher
- Make (optional, for convenience)
- Git

### Clone and Build

```bash
# Clone repository
git clone https://github.com/pentora-ai/pentora.git
cd pentora

# Build
go build -o pentora ./cmd/pentora

# Install
sudo mv pentora /usr/local/bin/

# Verify
pentora version
```

### Build with Make

```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Build and run tests
make test

# Build packages (DMG, DEB, RPM, etc.)
make pkg
```

### Development Build

```bash
# Build with debug symbols
go build -tags debug -o pentora ./cmd/pentora

# Run tests
make test

# Run linters
make lint
```

## Docker

### Pull Official Image

```bash
docker pull pentora/pentora:latest
```

### Run Container

```bash
# Basic scan
docker run --rm pentora/pentora scan 192.168.1.0/24

# With workspace persistence
docker run --rm -v $(pwd)/workspace:/workspace pentora/pentora scan 192.168.1.0/24

# Interactive mode
docker run -it --rm pentora/pentora
```

### Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'
services:
  pentora:
    image: pentora/pentora:latest
    volumes:
      - ./workspace:/workspace
      - ./config:/config
    environment:
      - PENTORA_WORKSPACE_DIR=/workspace
    command: ['server', 'start']
    ports:
      - '8080:8080'
```

Run:

```bash
docker-compose up -d
```

## Verification

After installation, verify Pentora is working:

```bash
# Check version
pentora version

# Run help
pentora --help

# Test with a simple scan (local interface)
pentora scan 127.0.0.1 --ports 22,80,443
```

## Configuration

Pentora looks for configuration in the following locations (in order):

1. `./pentora.yaml` (current directory)
2. `~/.config/pentora/config.yaml` (Linux/macOS)
3. `%AppData%\Pentora\config.yaml` (Windows)
4. `/etc/pentora/config.yaml` (system-wide, Linux)

Create a basic config file:

```bash
# Create config directory
mkdir -p ~/.config/pentora

# Generate default config
pentora config init > ~/.config/pentora/config.yaml
```

See the [Configuration Guide](../configuration/overview.md) for detailed options.

## Workspace Setup

Pentora uses a workspace directory to store scan results:

```bash
# Default workspace locations:
# Linux: ~/.local/share/pentora or $XDG_DATA_HOME/pentora
# macOS: ~/Library/Application Support/Pentora
# Windows: %AppData%\Pentora

# Initialize workspace
pentora workspace init

# Set custom workspace
export PENTORA_WORKSPACE_DIR=/path/to/workspace
pentora workspace init
```

## Updating Pentora

### Via Package Manager

```bash
# APT
sudo apt update && sudo apt upgrade pentora

# YUM/DNF
sudo yum update pentora

# Homebrew
brew upgrade pentora
```

### Manual Update

```bash
# Download latest release
curl -sSL https://pentora.io/install.sh | bash

# Verify new version
pentora version
```

### Check for Updates

```bash
# Check if newer version is available
pentora version --check-updates
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

# Remove workspace (optional)
rm -rf ~/.local/share/pentora  # Linux
rm -rf ~/Library/Application\ Support/Pentora  # macOS
```

## Troubleshooting

### Permission Denied

If you encounter permission errors:

```bash
# Ensure binary is executable
chmod +x /usr/local/bin/pentora

# Or run with sudo for privileged operations
sudo pentora scan 192.168.1.0/24
```

### Command Not Found

If `pentora` command is not found after installation:

```bash
# Check if binary exists
ls -l /usr/local/bin/pentora

# Add to PATH manually (add to ~/.bashrc or ~/.zshrc)
export PATH=$PATH:/usr/local/bin

# Reload shell
source ~/.bashrc  # or ~/.zshrc
```

### Network Scanning Requires Root

Raw socket operations require elevated privileges:

```bash
# Run with sudo
sudo pentora scan 192.168.1.0/24

# Or set capabilities (Linux only)
sudo setcap cap_net_raw,cap_net_admin+eip /usr/local/bin/pentora
```

## Next Steps

- 🚀 [Quick Start Guide](./quick-start.md) - Run your first scan
- 📖 [First Scan Tutorial](./first-scan.md) - Detailed walkthrough
- ⚙️ [Configuration Guide](../configuration/overview.md) - Customize Pentora
- 🔧 [CLI Reference](../cli/overview.md) - Explore all commands

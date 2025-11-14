---
sidebar_position: 1
---

# Installation

Vulntor supports multiple installation methods across macOS, Linux, and Windows platforms.

## System Requirements

### Minimum Requirements

- **CPU**: 2 cores
- **RAM**: 2 GB
- **Disk**: 100 MB for binaries, additional space for storage
- **OS**: macOS 10.15+, Linux (kernel 3.10+), Windows 10+

### Recommended for Large Scans

- **CPU**: 4+ cores
- **RAM**: 8+ GB
- **Disk**: 10 GB+ for storage and scan results
- **Network**: Low-latency connection for optimal scanning

## Quick Install (Recommended)

### Linux / macOS

Use the official install script:

```bash
curl -sSL https://vulntor.io/install.sh | bash
```

This script will:

1. Detect your platform and architecture
2. Download the latest release binary
3. Install to `/usr/local/bin/vulntor`
4. Set appropriate permissions

Verify installation:

```bash
vulntor version
```

### Manual Install Script

If you prefer to review the script first:

```bash
curl -sSL https://vulntor.io/install.sh -o install.sh
chmod +x install.sh
./install.sh
```

## Platform-Specific Installation

### macOS

#### Homebrew (Coming Soon)

```bash
brew tap vulntor/tap
brew install vulntor
```

#### Download Binary

```bash
# Download latest release
curl -LO https://github.com/vulntor-ai/vulntor/releases/latest/download/vulntor-darwin-amd64.tar.gz

# Extract
tar -xzf vulntor-darwin-amd64.tar.gz

# Move to PATH
sudo mv vulntor /usr/local/bin/

# Verify
vulntor version
```

#### DMG Installer

1. Download `vulntor-installer.dmg` from [releases page](https://github.com/vulntor-ai/vulntor/releases)
2. Open the DMG file
3. Drag Vulntor to Applications folder
4. Run from Terminal or add to PATH

### Linux

#### Debian / Ubuntu (APT)

```bash
# Add Vulntor repository
curl -fsSL https://vulntor.io/gpg.key | sudo gpg --dearmor -o /usr/share/keyrings/vulntor-archive-keyring.gpg

echo "deb [signed-by=/usr/share/keyrings/vulntor-archive-keyring.gpg] https://apt.vulntor.io stable main" | \
  sudo tee /etc/apt/sources.list.d/vulntor.list

# Install
sudo apt update
sudo apt install vulntor
```

#### RHEL / CentOS / Fedora (YUM/DNF)

```bash
# Add Vulntor repository
sudo tee /etc/yum.repos.d/vulntor.repo <<EOF
[vulntor]
name=Vulntor Repository
baseurl=https://yum.vulntor.io/stable
enabled=1
gpgcheck=1
gpgkey=https://vulntor.io/gpg.key
EOF

# Install
sudo yum install vulntor
# or
sudo dnf install vulntor
```

#### Arch Linux (AUR)

```bash
yay -S vulntor-bin
# or
paru -S vulntor-bin
```

#### Generic Linux Binary

```bash
# Download
curl -LO https://github.com/vulntor-ai/vulntor/releases/latest/download/vulntor-linux-amd64.tar.gz

# Extract
tar -xzf vulntor-linux-amd64.tar.gz

# Install
sudo mv vulntor /usr/local/bin/

# Verify
vulntor version
```

### Windows

#### Installer (MSI)

1. Download `vulntor-installer-x64.msi` from [releases page](https://github.com/vulntor-ai/vulntor/releases)
2. Run the installer
3. Follow the installation wizard
4. Vulntor will be added to PATH automatically

#### Chocolatey (Coming Soon)

```powershell
choco install vulntor
```

#### Scoop

```powershell
scoop bucket add vulntor https://github.com/vulntor/scoop-bucket
scoop install vulntor
```

#### Manual Binary

```powershell
# Download latest release
Invoke-WebRequest -Uri https://github.com/vulntor-ai/vulntor/releases/latest/download/vulntor-windows-amd64.zip -OutFile vulntor.zip

# Extract
Expand-Archive vulntor.zip -DestinationPath C:\Program Files\Vulntor

# Add to PATH (run as Administrator)
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\Program Files\Vulntor", "Machine")

# Verify
vulntor version
```

## Build from Source

### Prerequisites

- Go 1.21 or higher
- Make (optional, for convenience)
- Git

### Clone and Build

```bash
# Clone repository
git clone https://github.com/vulntor-ai/vulntor.git
cd vulntor

# Build
go build -o vulntor ./cmd/vulntor

# Install
sudo mv vulntor /usr/local/bin/

# Verify
vulntor version
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
go build -tags debug -o vulntor ./cmd/vulntor

# Run tests
make test

# Run linters
make lint
```

## Docker

### Pull Official Image

```bash
docker pull vulntor/vulntor:latest
```

### Run Container

```bash
# Basic scan
docker run --rm vulntor/vulntor scan 192.168.1.0/24

# With storage persistence
docker run --rm -v $(pwd)/storage:/storage vulntor/vulntor scan 192.168.1.0/24

# Interactive mode
docker run -it --rm vulntor/vulntor
```

### Docker Compose

Create `docker-compose.yml`:

```yaml
version: '3.8'
services:
  vulntor:
    image: vulntor/vulntor:latest
    volumes:
      - ./storage:/storage
      - ./config:/config
    environment:
      - VULNTOR_STORAGE_DIR=/storage
    command: ['server', 'start']
    ports:
      - '8080:8080'
```

Run:

```bash
docker-compose up -d
```

## Verification

After installation, verify Vulntor is working:

```bash
# Check version
vulntor version

# Run help
vulntor --help

# Test with a simple scan (local interface)
vulntor scan 127.0.0.1 --ports 22,80,443
```

## Configuration

Vulntor looks for configuration in the following locations (in order):

1. `./vulntor.yaml` (current directory)
2. `~/.config/vulntor/config.yaml` (Linux/macOS)
3. `%AppData%\Vulntor\config.yaml` (Windows)
4. `/etc/vulntor/config.yaml` (system-wide, Linux)

Create a basic config file:

```bash
# Create config directory
mkdir -p ~/.config/vulntor

# Generate default config
vulntor config init > ~/.config/vulntor/config.yaml
```

See the [Configuration Guide](../configuration/overview.md) for detailed options.

## Storage Setup

Vulntor uses a storage directory to store scan results:

```bash
# Default storage locations:
# Linux: ~/.local/share/vulntor or $XDG_DATA_HOME/vulntor
# macOS: ~/Library/Application Support/Vulntor
# Windows: %AppData%\Vulntor

# Initialize storage
vulntor storage init

# Set custom storage directory
export VULNTOR_STORAGE_DIR=/path/to/storage
vulntor storage init
```

## Updating Vulntor

### Via Package Manager

```bash
# APT
sudo apt update && sudo apt upgrade vulntor

# YUM/DNF
sudo yum update vulntor

# Homebrew
brew upgrade vulntor
```

### Manual Update

```bash
# Download latest release
curl -sSL https://vulntor.io/install.sh | bash

# Verify new version
vulntor version
```

### Check for Updates

```bash
# Check if newer version is available
vulntor version --check-updates
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

# Remove storage (optional)
rm -rf ~/.local/share/vulntor  # Linux
rm -rf ~/Library/Application\ Support/Vulntor  # macOS
```

## Troubleshooting

### Permission Denied

If you encounter permission errors:

```bash
# Ensure binary is executable
chmod +x /usr/local/bin/vulntor

# Or run with sudo for privileged operations
sudo vulntor scan 192.168.1.0/24
```

### Command Not Found

If `vulntor` command is not found after installation:

```bash
# Check if binary exists
ls -l /usr/local/bin/vulntor

# Add to PATH manually (add to ~/.bashrc or ~/.zshrc)
export PATH=$PATH:/usr/local/bin

# Reload shell
source ~/.bashrc  # or ~/.zshrc
```

### Network Scanning Requires Root

Raw socket operations require elevated privileges:

```bash
# Run with sudo
sudo vulntor scan 192.168.1.0/24

# Or set capabilities (Linux only)
sudo setcap cap_net_raw,cap_net_admin+eip /usr/local/bin/vulntor
```

## Next Steps

- 🚀 [Quick Start Guide](./quick-start.md) - Run your first scan
- 📖 [First Scan Tutorial](./first-scan.md) - Detailed walkthrough
- ⚙️ [Configuration Guide](../configuration/overview.md) - Customize Vulntor
- 🔧 [CLI Reference](../cli/overview.md) - Explore all commands

# Air-Gapped Deployment

Deploy Vulntor in environments without internet access.

## Offline Installation

1. **Download on internet-connected system**:

```bash
# Download binary
curl -LO https://github.com/vulntor-ai/vulntor/releases/latest/download/vulntor-linux-amd64

# Download fingerprint catalog
curl -LO https://catalog.vulntor.io/fingerprints.yaml
```

2. **Transfer to air-gapped system**:

```bash
scp vulntor-linux-amd64 fingerprints.yaml user@airgapped-host:/tmp/
```

3. **Install on air-gapped system**:

```bash
sudo mv /tmp/vulntor-linux-amd64 /usr/local/bin/vulntor
sudo chmod +x /usr/local/bin/vulntor

# Install fingerprint catalog
mkdir -p ~/.local/share/vulntor/cache/fingerprints/
cp /tmp/fingerprints.yaml ~/.local/share/vulntor/cache/fingerprints/
```

## Configuration

Disable remote features:

```yaml
fingerprint:
  cache:
    auto_sync: false # Disable automatic catalog updates

server:
  external_access: false
```

## Enterprise License

Copy license file:

```bash
cp license.key ~/.local/share/vulntor/config/
```

Offline grace period: 7 days

## Updates

Manual update process:

1. Download new version on internet-connected system
2. Transfer to air-gapped environment
3. Replace binary
4. Restart service

See [Enterprise Overview](/enterprise/overview) for air-gapped features.

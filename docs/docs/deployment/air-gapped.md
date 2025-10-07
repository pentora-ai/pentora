# Air-Gapped Deployment

Deploy Pentora in environments without internet access.

## Offline Installation

1. **Download on internet-connected system**:

```bash
# Download binary
curl -LO https://github.com/pentora-ai/pentora/releases/latest/download/pentora-linux-amd64

# Download fingerprint catalog
curl -LO https://catalog.pentora.io/fingerprints.yaml
```

2. **Transfer to air-gapped system**:

```bash
scp pentora-linux-amd64 fingerprints.yaml user@airgapped-host:/tmp/
```

3. **Install on air-gapped system**:

```bash
sudo mv /tmp/pentora-linux-amd64 /usr/local/bin/pentora
sudo chmod +x /usr/local/bin/pentora

# Install fingerprint catalog
mkdir -p ~/.local/share/pentora/cache/fingerprints/
cp /tmp/fingerprints.yaml ~/.local/share/pentora/cache/fingerprints/
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
cp license.key ~/.local/share/pentora/config/
```

Offline grace period: 7 days

## Updates

Manual update process:

1. Download new version on internet-connected system
2. Transfer to air-gapped environment
3. Replace binary
4. Restart service

See [Enterprise Overview](/docs/enterprise/overview) for air-gapped features.

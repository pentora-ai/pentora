# JWT-Based Licensing

Pentora Enterprise uses JWT tokens for license management and feature gating.

## License Structure

JWT payload:
```json
{
  "customer_id": "company-123",
  "license_id": "lic-456",
  "plan": "enterprise",
  "features": [
    "distributed",
    "multi_tenant",
    "compliance_packs",
    "siem_integrations"
  ],
  "allowed_agents": 50,
  "issued_at": 1696608000,
  "expiry": 1728230400
}
```

Signed with Ed25519/ECDSA key.

## License File

Location: `<storage>/config/license.key`

```bash
# Install license
cp license.key ~/.local/share/pentora/config/

# Verify license
pentora license verify

# View license details
pentora license show
```

## Feature Gating

Code checks features at runtime:

```go
if feature.Check("distributed") {
    // Enable distributed scanning
} else {
    return errors.New("distributed scanning requires Enterprise license")
}
```

## Offline Grace Period

License allows 7-day offline operation before requiring refresh.

## License Renewal

```bash
# Check expiry
pentora license show

# Refresh from server
pentora license refresh

# Manual update
cp new-license.key ~/.local/share/pentora/config/license.key
pentora server restart
```

Expiry warnings: 30 days, 7 days, 1 day before expiration.

See [Server Commands](/cli/server) for management.

# REST API Authentication

Secure API access with token-based authentication.

## API Tokens

### Create Token

```bash
vulntor server token create \
  --name "Production API" \
  --scopes "scan:read,scan:write,storage:read" \
  --expires 90d
```

### List Tokens

```bash
vulntor server token list
```

### Revoke Token

```bash
vulntor server token revoke <token-id>
```

## Using Tokens

### Bearer Authentication

```bash
curl -H "Authorization: Bearer eyJhbGc..." \
     https://vulntor.company.com/api/v1/scans
```

### Environment Variable

```bash
export VULNTOR_API_TOKEN=eyJhbGc...
curl -H "Authorization: Bearer $VULNTOR_API_TOKEN" \
     https://vulntor.company.com/api/v1/scans
```

## Token Scopes

- `scan:read` - View scans
- `scan:write` - Create/delete scans
- `scan:execute` - Execute scans
- `storage:read` - View storage
- `storage:write` - Modify storage
- `admin` - Full access

## SSO Integration (Enterprise)

### OIDC

```yaml
server:
  auth:
    provider: oidc
    oidc:
      issuer: https://auth.company.com
      client_id: vulntor
      client_secret: ${OIDC_SECRET}
```

### SAML

```yaml
server:
  auth:
    provider: saml
    saml:
      idp_metadata_url: https://idp.company.com/metadata
      sp_entity_id: vulntor
```

See [Enterprise Multi-Tenant](/enterprise/multi-tenant) for details.

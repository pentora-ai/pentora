# Multi-Tenant Storage

Isolate scan data and access between multiple tenants or customers.

## Directory Structure

```
storage/
├── tenants/
│   ├── customer-a/
│   │   ├── scans/
│   │   ├── reports/
│   │   └── config/
│   ├── customer-b/
│   │   └── ...
```

## Tenant Configuration

```yaml
enterprise:
  multi_tenant:
    enabled: true
    default_tenant: default
    isolation: strict
```

## RBAC

Role-based access control per tenant:

**Roles**:
- **Viewer**: Read-only access to scan results
- **Operator**: Run scans, view results
- **Admin**: Full access, manage users

**Permissions**:
```yaml
rbac:
  roles:
    - name: operator
      permissions:
        - scans:read
        - scans:create
        - reports:read
    - name: admin
      permissions:
        - "*"
```

## SSO Integration

OIDC/SAML configuration:
```yaml
server:
  auth:
    provider: oidc
    oidc:
      issuer: https://auth.company.com
      client_id: vulntor
      client_secret: ${OIDC_SECRET}
```

## Tenant Switching

Via UI: Tenant selector dropdown
Via API: `X-Tenant-ID` header

```bash
curl -H "X-Tenant-ID: customer-a" \
     -H "Authorization: Bearer token" \
     https://vulntor.company.com/api/v1/scans
```

See [Deployment Guide](/deployment/server-mode) for setup.

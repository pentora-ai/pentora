# API Overview

Vulntor provides REST and gRPC APIs for programmatic access and integration.

## Base URL

```
https://vulntor.company.com/api/v1
```

## Authentication

All API requests require authentication via Bearer token:

```bash
curl -H "Authorization: Bearer <token>" \
     https://vulntor.company.com/api/v1/scans
```

## Generate API Token

```bash
vulntor server token create --name "CI Pipeline" --scopes scan:read,scan:write
```

## API Versioning

Current version: **v1**

URL format: `/api/v1/<resource>`

## Rate Limiting

Default limits:
- **Free/Starter**: 60 requests/minute
- **Team**: 100 requests/minute
- **Business**: 500 requests/minute
- **Enterprise**: Unlimited (configurable)

## Common Headers

```
Authorization: Bearer <token>
Content-Type: application/json
X-Tenant-ID: <tenant-id>  (multi-tenant only)
```

## Response Format

Success (200):
```json
{
  "data": { ... },
  "meta": {
    "timestamp": "2023-10-06T14:30:22Z"
  }
}
```

Error (4xx/5xx):
```json
{
  "error": {
    "code": "invalid_request",
    "message": "Target validation failed",
    "details": { ... }
  }
}
```

## Available APIs

- [REST API](/api/rest/scans) - Scan management
- [UI Portal](/api/ui/overview) - Web interface
- [Module API](/api/modules/interface) - Custom modules

See sections for detailed endpoints.

# Workspace API

Manage workspace and scan storage via REST API.

## Get Workspace Info

**GET** `/api/v1/workspace/info`

```bash
curl https://pentora.company.com/api/v1/workspace/info \
  -H "Authorization: Bearer $TOKEN"
```

**Response**:
```json
{
  "data": {
    "path": "/var/lib/pentora",
    "size_bytes": 2147483648,
    "scan_count": 145,
    "oldest_scan": "2023-09-01T00:00:00Z",
    "newest_scan": "2023-10-06T14:30:22Z"
  }
}
```

## Garbage Collection

**POST** `/api/v1/workspace/gc`

```bash
curl -X POST https://pentora.company.com/api/v1/workspace/gc \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "older_than": "30d",
    "keep_last": 100,
    "dry_run": false
  }'
```

## Export Scan

**GET** `/api/v1/workspace/scans/{scan_id}/export`

```bash
curl https://pentora.company.com/api/v1/workspace/scans/20231006-143022-a1b2c3/export?format=csv \
  -H "Authorization: Bearer $TOKEN" \
  -o export.csv
```

See [REST Scans API](/docs/api/rest/scans) for scan operations.

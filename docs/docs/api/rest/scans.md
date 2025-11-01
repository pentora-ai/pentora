# Scan Management API

REST endpoints for scan submission and management.

## Submit Scan

**POST** `/api/v1/scans`

```bash
curl -X POST https://pentora.company.com/api/v1/scans \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "targets": ["192.168.1.0/24"],
    "profile": "standard",
    "vuln": true,
    "notifications": ["slack://security-alerts"]
  }'
```

**Response**:
```json
{
  "data": {
    "scan_id": "20231006-143022-a1b2c3",
    "status": "queued",
    "created_at": "2023-10-06T14:30:22Z"
  }
}
```

## List Scans

**GET** `/api/v1/scans`

```bash
curl https://pentora.company.com/api/v1/scans?limit=20&status=completed \
  -H "Authorization: Bearer $TOKEN"
```

**Query Parameters**:
- `limit`: Results per page (default: 50)
- `offset`: Pagination offset
- `status`: Filter by status (pending, running, completed, failed)
- `since`: ISO 8601 timestamp

## Get Scan Details

**GET** `/api/v1/scans/{scan_id}`

```bash
curl https://pentora.company.com/api/v1/scans/20231006-143022-a1b2c3 \
  -H "Authorization: Bearer $TOKEN"
```

**Response**:
```json
{
  "data": {
    "scan_id": "20231006-143022-a1b2c3",
    "status": "completed",
    "targets": ["192.168.1.0/24"],
    "started_at": "2023-10-06T14:30:22Z",
    "completed_at": "2023-10-06T14:35:45Z",
    "results": {
      "live_hosts": 15,
      "open_ports": 73,
      "vulnerabilities": 12
    }
  }
}
```

## Delete Scan

**DELETE** `/api/v1/scans/{scan_id}`

```bash
curl -X DELETE https://pentora.company.com/api/v1/scans/20231006-143022-a1b2c3 \
  -H "Authorization: Bearer $TOKEN"
```

## Download Results

**GET** `/api/v1/scans/{scan_id}/results`

```bash
curl https://pentora.company.com/api/v1/scans/20231006-143022-a1b2c3/results?format=json \
  -H "Authorization: Bearer $TOKEN" \
  -o results.json
```

**Query Parameters**:
- `format`: json, jsonl, csv (default: json)

See [Scans API](/api/rest/scans) for scan and storage management.

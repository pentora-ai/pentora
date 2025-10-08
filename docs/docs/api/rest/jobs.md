# Job Management API (Enterprise)

Manage distributed job queue and scheduling.

## Submit Job

**POST** `/api/v1/jobs`

```bash
curl -X POST https://pentora.company.com/api/v1/jobs \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "scan",
    "params": {
      "targets": ["10.0.0.0/16"],
      "profile": "deep"
    },
    "schedule": "0 2 * * *",
    "pool": "fast-pool"
  }'
```

## List Jobs

**GET** `/api/v1/jobs`

```bash
curl https://pentora.company.com/api/v1/jobs?status=running \
  -H "Authorization: Bearer $TOKEN"
```

## Get Job Status

**GET** `/api/v1/jobs/{job_id}`

```bash
curl https://pentora.company.com/api/v1/jobs/job-123 \
  -H "Authorization: Bearer $TOKEN"
```

## Cancel Job

**POST** `/api/v1/jobs/{job_id}/cancel`

```bash
curl -X POST https://pentora.company.com/api/v1/jobs/job-123/cancel \
  -H "Authorization: Bearer $TOKEN"
```

Requires Enterprise license. See [Distributed Scanning](/enterprise/distributed-scanning).

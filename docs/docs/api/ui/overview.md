# UI Portal Overview

Pentora UI provides a clean, approachable interface for non-technical stakeholders.

## Purpose

The UI targets:
- Security managers and executives
- Compliance auditors
- Non-technical stakeholders
- Scheduled scan management

While technical operators use the CLI, the UI empowers the rest of the organization.

## Access

Navigate to server address:
```
https://pentora.company.com/ui
```

Default credentials (change immediately):
- Username: `admin`
- Password: `changeme`

## Key Features

### 1. Dashboard
- Scan summary widgets
- Trend charts (ports, vulnerabilities, scans)
- System health (uptime, license, workers)
- Quick actions (start scan, schedule, invite)

### 2. Scan Management
- Scan list with filters
- Scan creation wizard
- Real-time status
- Export options (JSON, CSV, PDF)

### 3. Workspace Explorer
- Directory view of scans
- File inspector for results
- Search and filter
- Retention controls

### 4. Scheduler
- Calendar view of scheduled scans
- Recurring jobs configuration
- Maintenance windows

### 5. Notifications
- Slack, email, webhook configuration
- Test delivery
- Alert severity thresholds

## Architecture

```
[Browser] → [Pentora Server] → [REST API] → [Workspace]
                ↓
         [Static UI Files]
```

UI never reads workspace files directly; server abstracts storage.

See [Dashboard Features](/api/ui/dashboard) for details.

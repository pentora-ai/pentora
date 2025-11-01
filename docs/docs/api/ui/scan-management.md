# Scan Management UI

Create, monitor, and manage scans through the web interface.

## Scan List

**Path**: `/ui/scans`

Table columns:
- Scan name
- Target scope
- Profile
- Schedule (one-off / recurring)
- Status (queued, running, completed, failed)
- Last run / Next run
- Owner

**Actions**:
- Filter by status, profile, owner
- Sort by date, status, name
- Bulk operations (pause, delete)

## Scan Creation Wizard

**Path**: `/ui/scans/new`

### Step 1: Select Targets
- Enter IPs, hostnames, CIDR ranges
- Upload target file
- Select from saved target groups

### Step 2: Choose Profile
Predefined profiles:
- Standard Discovery
- Web App Focus
- Infrastructure Audit
- Custom

### Step 3: Advanced Options (Optional)
Hidden by default:
- Concurrency settings
- Enable vulnerability matching
- Scan window (start time, duration)

### Step 4: Confirmation
- Summary of configuration
- Storage path
- Notification recipients
- **Start Scan** button

## Scan Detail Page

**Path**: `/ui/scans/{scan_id}`

### Status Timeline
Visual timeline:
```
Queued → Running → Completed → Archived
```

### Target Breakdown
Hierarchical view:
```
Host: 192.168.1.100
  ├─ Port 22/tcp (open)
  │   ├─ Banner: SSH-2.0-OpenSSH_8.2p1
  │   └─ Service: OpenSSH 8.2p1
  ├─ Port 80/tcp (open)
  │   ├─ Banner: HTTP/1.1 200 OK
  │   ├─ Service: nginx 1.18.0
  │   └─ Vulnerability: CVE-2021-23017 (critical)
```

### Filter Chips
- Severity (critical, high, medium, low)
- Protocol (tcp, udp)
- Asset tags

### Manual Notes
Add comments per host or finding:
- Timestamp
- Author
- Markdown support

### Actions
- **Re-run Scan**: Repeat with same config
- **Export Results**: JSON, CSV, PDF
- **Create Ticket**: Jira, ServiceNow (Enterprise)
- **Acknowledge**: Mark finding as reviewed
- **Resolve**: Mark vulnerability as fixed

## Export Options

1. **JSON**: Complete data
2. **CSV**: Spreadsheet-friendly
3. **PDF**: Executive report (Enterprise)
4. **Send to Integration**: Direct to Jira/ServiceNow

See [Scan API](/api/rest/scans) for programmatic access.

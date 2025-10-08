# Notifications and Integrations UI

Configure alert channels and external integrations.

## Notification Rules

**Path**: `/ui/settings/notifications`

### Create Rule

1. **Name**: Rule identifier
2. **Channels**: Select Slack, email, webhook
3. **Conditions**:
   - Severity threshold (critical, high, medium)
   - Asset tags
   - Scan profiles
4. **Test**: Send test notification
5. **Save**

### Example Rule

```
Name: Critical Production Alerts
Channels: Slack (#prod-security), Email (oncall@company.com)
Conditions:
  - Severity: Critical
  - Asset tags: production
  - Scan profile: Any
```

## Integration Catalog

**Path**: `/ui/integrations`

Grid/list of available integrations:
- **Slack**: Team communication
- **Jira**: Ticketing
- **ServiceNow**: Incident management
- **Splunk**: SIEM
- **QRadar**: SIEM
- **Webhook**: Custom endpoints

### Connection Status
- ✓ Connected (green)
- ✗ Disconnected (red)
- ⚠ Warning (yellow)

## Slack Configuration

1. Navigate to **Integrations** → **Slack**
2. Click **Connect**
3. Authorize Pentora app
4. Select channel
5. Test notification

## Webhook Editor

**Path**: `/ui/integrations/webhook`

Fields:
- **Endpoint URL**: Destination
- **Auth Token**: Bearer token
- **Payload Template**: JSON template with variables

**Preview**: Test payload rendering

**Example Template**:
```json
{
  "text": "Scan {{ .ScanID }} found {{ .VulnCount }} vulnerabilities",
  "severity": "{{ .MaxSeverity }}",
  "targets": {{ .Targets }}
}
```

## Incident Automation (Enterprise)

**Path**: `/ui/integrations/automation`

Rule builder:
```
WHEN: Critical vulnerability found
  AND: Asset group = Production Servers
THEN: Create Jira ticket
  IN: Project SEC
  WITH: Priority = Highest
```

See [Enterprise Integrations](/enterprise/integrations) for details.

# Enterprise Integrations

Connect Pentora with SIEM, ticketing, and collaboration platforms.

## SIEM Integrations

### Splunk
```yaml
enterprise:
  integrations:
    siem:
      - type: splunk
        url: https://splunk.company.com:8088
        token: ${SPLUNK_HEC_TOKEN}
        index: security
```

### QRadar
```yaml
enterprise:
  integrations:
    siem:
      - type: qradar
        url: https://qradar.company.com
        api_token: ${QRADAR_TOKEN}
```

### Elastic
```yaml
enterprise:
  integrations:
    siem:
      - type: elasticsearch
        url: https://elastic.company.com:9200
        index: pentora-scans
        api_key: ${ELASTIC_API_KEY}
```

## Ticketing Systems

### Jira
```yaml
enterprise:
  integrations:
    ticketing:
      - type: jira
        url: https://company.atlassian.net
        project: SEC
        api_token: ${JIRA_TOKEN}
        automation:
          create_on_critical: true
```

### ServiceNow
```yaml
enterprise:
  integrations:
    ticketing:
      - type: servicenow
        instance: company.service-now.com
        username: pentora
        password: ${SNOW_PASSWORD}
```

## Collaboration

### Slack
```yaml
notifications:
  slack:
    webhook_url: ${SLACK_WEBHOOK}
    channel: "#security-alerts"
    severity_threshold: high
```

### Microsoft Teams
```yaml
notifications:
  teams:
    webhook_url: ${TEAMS_WEBHOOK}
```

## CMDB Sync

Sync asset inventory to CMDB:
```yaml
enterprise:
  integrations:
    cmdb:
      - type: servicenow
        table: cmdb_ci_server
        sync_interval: 24h
```

See [Notification Configuration](/docs/configuration/overview) for setup.

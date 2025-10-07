# Hook System and Events

Subscribe to scan lifecycle events for custom workflows.

## Event Types

- `scan.started`
- `scan.completed`
- `scan.failed`
- `node.started`
- `node.completed`
- `node.failed`
- `discovery.host_found`
- `scanner.port_open`
- `vulnerability.found`

## Event Subscription

```go
orchestrator.OnNodeComplete(func(node Node, result Result) {
    log.Info().
        Str("node", node.ID).
        Dur("duration", result.Duration).
        Msg("Node completed")
})

orchestrator.OnEvent("vulnerability.found", func(event Event) {
    vuln := event.Data.(Vulnerability)
    if vuln.Severity == "critical" {
        alertSecurityTeam(vuln)
    }
})
```

## Webhook Notifications

Configure webhooks for external integrations:

```yaml
notifications:
  webhooks:
    - url: https://company.com/security/webhook
      events: [scan.completed, vulnerability.found]
      headers:
        Authorization: "Bearer token"
```

See [Reporting & Notification](/docs/concepts/scan-pipeline#stage-8-reporting--notification) for configuration.

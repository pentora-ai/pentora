# Custom Fingerprint Rules

Create custom fingerprint rules for proprietary or internal services.

## Rule Format

```yaml
fingerprints:
  - name: custom_app
    category: http
    description: Internal application detection
    patterns:
      - type: http_header
        header: Server
        regex: 'CustomApp/([0-9.]+)'
        version_group: 1
        confidence: 95
      - type: http_content
        regex: '<title>CustomApp v([0-9.]+)</title>'
        version_group: 1
        confidence: 90
```

## Pattern Types

- `banner`: Raw TCP banner
- `http_header`: HTTP response header
- `http_content`: HTTP response body
- `tls_cert`: TLS certificate fields
- `ssh_banner`: SSH protocol banner

## Advanced Patterns

```yaml
fingerprints:
  - name: complex_app
    category: custom
    patterns:
      - type: http_header
        header: X-App-Version
        regex: '([0-9.]+)'
        version_group: 1
        os_hint: linux
        confidence: 95
        requires:
          - type: http_header
            header: X-App-Name
            value: "InternalApp"
```

## Testing Rules

```bash
pentora fingerprint validate custom-rules.yaml
pentora fingerprint test custom-rules.yaml sample-banner.txt
```

See [Fingerprint Command](/docs/cli/fingerprint) for management tools.

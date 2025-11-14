# Report Generation and Customization

Generate and customize scan reports for different audiences.

## Built-in Formats

### JSON
Machine-readable, complete data:
```bash
vulntor scan --targets 192.168.1.100 --output results.json --format json
```

### CSV
Spreadsheet import:
```bash
vulntor scan --targets 192.168.1.100 --output report.csv --format csv
```

### JSONL
Streaming/log-friendly:
```bash
vulntor scan --targets 192.168.1.100 --output results.jsonl --format jsonl
```

### PDF (Enterprise)
Executive reports:
```bash
vulntor storage export scan-id --format pdf -o executive-report.pdf
```

## Custom Templates

Use Go templates for custom output:

```bash
# Create template
cat > custom-report.tmpl <<'TMPL'
# Scan Report

**Date**: {{ .Timestamp }}
**Targets**: {{ .TargetCount }}

## Summary
- Live hosts: {{ .LiveHosts }}
- Open ports: {{ .OpenPorts }}
- Vulnerabilities: {{ .VulnCount }}

## Critical Findings
{{ range .Vulnerabilities }}
{{ if eq .Severity "critical" }}
- {{ .CVE }}: {{ .Title }}
{{ end }}
{{ end }}
TMPL

# Use template
vulntor scan --targets 192.168.1.100 --template custom-report.tmpl -o report.md
```

## Report Sections

### Executive Summary
- High-level overview
- Risk summary
- Key findings
- Recommendations

### Technical Details
- Host inventory
- Service fingerprints
- Vulnerability details
- Remediation steps

### Compliance (Enterprise)
- Framework mapping
- Pass/fail status
- Control violations

## Scheduled Reports (Enterprise)

```bash
# Weekly executive report
vulntor report schedule \
  --name "Weekly Security Posture" \
  --schedule "0 9 * * 1" \
  --format pdf \
  --email exec-team@company.com
```

## Custom Dashboards (Enterprise)

Create custom dashboards in UI:
1. Navigate to **Reports** → **Dashboards**
2. Add widgets (scan summary, vulnerability trends)
3. Save and share

See [UI Dashboard](/api/ui/dashboard) for details.

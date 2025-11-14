# Compliance Scanning

Assess compliance against regulatory frameworks (Enterprise).

## Supported Frameworks

- **CIS Benchmarks**: Level 1 and Level 2
- **PCI DSS**: Payment Card Industry
- **NIST 800-53**: US Federal security controls
- **HIPAA**: Healthcare security
- **ISO 27001**: Information security management

## Running Compliance Scans

```bash
# CIS Level 1
vulntor scan --targets prod-servers.txt --compliance cis-level1

# PCI DSS
vulntor scan --targets cardholder-env.txt --compliance pci-dss

# Multiple frameworks
vulntor scan --targets critical.txt --compliance "cis-level1,nist-800-53"
```

## Compliance Reports

Generate compliance reports:
```bash
vulntor storage export scan-id --format pdf --compliance-report pci-dss
```

## Common Controls

### CIS Benchmark Checks
- Disable unused services
- Strong password policies
- Firewall configuration
- Patch management
- Logging and monitoring

### PCI DSS Requirements
- Network segmentation
- Encryption in transit (TLS)
- Access control (RBAC)
- Regular vulnerability scanning
- Audit logging

### NIST 800-53
- Access control (AC family)
- Security assessment (CA family)
- Configuration management (CM family)
- System monitoring (SI family)

## Remediation Tracking

```bash
# Initial assessment
vulntor scan --targets servers.txt --compliance pci-dss -o baseline.json

# After remediation
vulntor scan --targets servers.txt --compliance pci-dss -o remediated.json

# Compare
diff baseline.json remediated.json
```

## Continuous Compliance

Schedule regular checks:
```bash
vulntor scan --targets prod.txt \
  --compliance cis-level1 \
  --schedule "0 2 * * 0" \
  --notify email://compliance-team@company.com
```

Requires Enterprise license. See [Enterprise Overview](/enterprise/overview).

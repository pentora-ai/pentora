# Fingerprint Validation Dataset

This directory contains test cases for validating the fingerprint resolver's accuracy and performance.

## Files

- **validation_dataset.yaml**: 130 labeled test cases for measuring detection accuracy
  - 90 true positives (services that should be detected)
  - 40 true negatives (services that should NOT be detected)
  - 10 edge cases (challenging scenarios)

## Schema

### Test Case Structure

```yaml
true_positives:
  - protocol: string          # Required: Protocol name (http, ssh, ftp, mysql, etc.)
    port: int                 # Required: Port number (80, 22, 3306, etc.)
    banner: string            # Required: Raw banner/response text
    expected_product: string  # Required: Expected product name (Apache, OpenSSH, etc.)
    expected_vendor: string   # Optional: Expected vendor name (Apache, OpenBSD, etc.)
    expected_version: string  # Optional: Expected version string (2.4.41, 8.2p1, etc.)
    description: string       # Required: Human-readable description

true_negatives:
  - protocol: string          # Required: Protocol name
    port: int                 # Required: Port number
    banner: string            # Required: Raw banner/response text
    expected_match: false     # Required: Must be false (indicates should NOT match)
    description: string       # Required: Human-readable description (explain why no match)

edge_cases:
  - protocol: string          # Required: Protocol name
    port: int                 # Required: Port number
    banner: string            # Required: Raw banner/response text
    expected_product: string  # Optional: Expected product (if should match)
    expected_version: string  # Optional: Expected version (empty string if no version)
    description: string       # Required: Human-readable description (explain edge case)
```

### Field Descriptions

**Required Fields**:
- `protocol`: Protocol identifier matching fingerprint rules (e.g., "http", "ssh", "ftp", "mysql")
- `port`: TCP/UDP port number (must be valid: 1-65535)
- `banner`: Raw service banner or response text (preserve exact casing, whitespace)
- `description`: Clear explanation of what this test case validates

**Optional Fields**:
- `expected_product`: Product name to match (e.g., "Apache", "Nginx", "OpenSSH")
- `expected_vendor`: Vendor/organization name (e.g., "Apache", "OpenBSD", "Microsoft")
- `expected_version`: Version string to extract (e.g., "2.4.41", "8.2p1", "10.0.19041")
- `expected_match`: Boolean flag for true negatives (must be `false` when present)

### Banner Normalization

**Important**: Banners are used AS-IS without normalization. Preserve:
- Exact casing (e.g., "Server:" vs "server:")
- Leading/trailing whitespace
- Line breaks (`\r\n` or `\n`)
- Special characters

**Example**:
```yaml
# Correct (preserves exact banner)
banner: "Server: Apache/2.4.41 (Ubuntu)\r\n"

# Incorrect (normalized)
banner: "apache/2.4.41"
```

## Adding New Test Cases

### True Positives (Should Match)

Add cases for services that SHOULD be detected:

```yaml
true_positives:
  - protocol: http
    port: 8080
    banner: "Server: Caddy v2.6.4"
    expected_product: "Caddy"
    expected_vendor: "Caddy"
    expected_version: "2.6.4"
    description: "Caddy web server with version"
```

**Guidelines**:
- Use real-world banners from actual services (use `nmap`, `curl`, `nc`)
- Include common services (Apache, Nginx, OpenSSH, MySQL, etc.)
- Include modern services (Caddy, Traefik, Express.js, etc.)
- Include enterprise services (Exchange, IIS, Cisco devices)
- Cover multiple versions per product (e.g., Apache 2.2, 2.4, 2.5)
- Test version extraction edge cases (truncated, non-standard formats)

### True Negatives (Should NOT Match)

Add cases for services that should be REJECTED:

```yaml
true_negatives:
  - protocol: mysql
    port: 3306
    banner: "Server: Apache/2.4.41 (Ubuntu)"
    expected_match: false
    description: "HTTP banner on MySQL port (anti-pattern: wrong protocol)"
```

**Guidelines**:
- **Anti-patterns**: Correct banner on wrong port (HTTP on MySQL port)
- **Malformed**: Corrupted/incomplete banners (truncated, encoding issues)
- **Unknown**: Custom protocols, obfuscated banners, non-standard services
- **Cross-protocol**: Services masquerading as other protocols
- **Noise**: Error messages, HTML pages on binary ports

### Edge Cases (Challenging Scenarios)

Add cases that test resolver resilience:

```yaml
edge_cases:
  - protocol: http
    port: 80
    banner: "Server: Apache"
    expected_product: "Apache"
    expected_version: ""
    description: "Apache without version (should match product, version optional)"
```

**Guidelines**:
- **Version-less**: Services without version strings
- **Multi-protocol**: Services supporting multiple protocols (HTTP + HTTPS)
- **Obfuscated**: Modified banners (e.g., "ApacheServer" vs "Apache")
- **Case variations**: Mixed casing ("APACHE", "apache", "Apache")
- **Whitespace**: Extra spaces, tabs, line breaks
- **Encoding**: UTF-8, Latin-1, special characters

## Contribution Process

### 1. Add Test Case

1. Identify a gap in test coverage (missing protocol, service, edge case)
2. Collect real-world banner from actual service:
   ```bash
   # HTTP
   curl -I http://example.com | head -1

   # SSH
   nc example.com 22 | head -1

   # MySQL
   mysql -h example.com -e "SELECT VERSION();"
   ```
3. Add test case to appropriate section (true_positives, true_negatives, edge_cases)
4. Ensure all required fields are present
5. Write clear, descriptive `description` field

### 2. Validate Syntax

```bash
# Verify YAML syntax
yamllint pkg/fingerprint/testdata/validation_dataset.yaml

# Or use Python
python3 -c "import yaml; yaml.safe_load(open('pkg/fingerprint/testdata/validation_dataset.yaml'))"
```

### 3. Run Validation Tests

```bash
# Run validation tests
go test -v ./pkg/fingerprint -run TestValidationRunner

# Check metrics
go test -v ./pkg/fingerprint -run TestValidationRunner 2>&1 | grep "Metrics Passed"
```

### 4. Update Metrics Expectations

If adding test cases changes metrics significantly:

1. Document the impact in your commit message
2. Update VALIDATION_REPORT.md if needed
3. Explain why metrics changed (e.g., "Added 10 TN cases for anti-patterns, FPR decreased")

## Quality Guidelines

### Banner Quality

**Do**:
- ✅ Use real banners from production services
- ✅ Preserve exact formatting (spaces, line breaks, casing)
- ✅ Include full banner (not truncated)
- ✅ Test against current resolver rules

**Don't**:
- ❌ Invent synthetic banners
- ❌ Modify banners to "make them easier"
- ❌ Use outdated/obsolete banners
- ❌ Duplicate test cases unnecessarily

### Description Quality

**Good descriptions**:
```yaml
description: "Apache 2.4 on Ubuntu with full version extraction"
description: "HTTP banner on MySQL port (anti-pattern test)"
description: "OpenSSH without version (product-only match)"
```

**Bad descriptions**:
```yaml
description: "test case"
description: "apache"
description: "should work"
```

### Coverage Balance

Target distribution:
- **True Positives**: 60-70% (90/130 currently)
- **True Negatives**: 25-35% (40/130 currently)
- **Edge Cases**: 5-10% (10/130 currently)

Protocol distribution:
- **Common protocols** (HTTP, SSH, DB): 50-60%
- **Enterprise protocols** (LDAP, Kerberos, SMB): 20-30%
- **Modern protocols** (MQTT, CoAP, gRPC): 10-20%

## Running Validation Suite

### Basic Validation

```bash
# Run all validation tests
go test -v ./pkg/fingerprint -run TestValidationRunner

# Run with metrics output
go test -v ./pkg/fingerprint -run TestValidationRunner 2>&1 | tee validation.log
```

### Performance Benchmarks

```bash
# Run all benchmarks
go test -bench=. -benchmem ./pkg/fingerprint -run=^$

# Run specific benchmark
go test -bench=BenchmarkValidationRunner -benchmem ./pkg/fingerprint

# Run with longer duration for accuracy
go test -bench=. -benchmem ./pkg/fingerprint -benchtime=5s
```

### Coverage Analysis

```bash
# Run with coverage
go test -cover ./pkg/fingerprint -run TestValidationRunner

# Generate coverage report
go test -coverprofile=coverage.out ./pkg/fingerprint
go tool cover -html=coverage.out
```

### Full Validation Report

```bash
# Run validation and generate report
go test -v ./pkg/fingerprint -run TestValidationRunner 2>&1 | \
  grep -A 100 "Validation Metrics:" | \
  tee validation_metrics.txt
```

## Metrics Interpretation

### Confusion Matrix

```
                    Predicted Positive    Predicted Negative
Actual Positive     TP (correct match)    FN (missed detection)
Actual Negative     FP (false alarm)      TN (correct rejection)
```

### Metric Formulas

- **False Positive Rate (FPR)** = FP / (FP + TN) — Lower is better (<10%)
- **True Positive Rate (TPR)** = TP / (TP + FN) — Higher is better (>80%)
- **Precision** = TP / (TP + FP) — Higher is better (>85%)
- **Recall** = TPR — Same as True Positive Rate
- **F1 Score** = 2 × (Precision × Recall) / (Precision + Recall) — Higher is better (>0.82)

### Current Targets

| Metric | Target | Current | Status |
|--------|--------|---------|--------|
| False Positive Rate | <10% | 23.68% | ❌ Needs improvement |
| True Positive Rate | >80% | 51.09% | ❌ Needs improvement |
| Precision | >85% | 83.93% | 🟡 Close (1.07% gap) |
| F1 Score | >0.82 | 0.6351 | ❌ Needs improvement |
| Protocol Coverage | 20+ | 16 | ❌ Add 4+ protocols |
| Version Extraction | >70% | 54.67% | ❌ Needs improvement |
| Performance | <50ms | 0.95µs | ✅ Excellent |

## Troubleshooting

### Test Case Not Matching

1. **Check protocol name**: Must match fingerprint rule protocol
2. **Check banner format**: Must be exact (spaces, casing, line breaks)
3. **Check rule exists**: `grep -i "product_name" pkg/fingerprint/data/fingerprint_db.yaml`
4. **Check pattern strength**: Rule may have low confidence threshold
5. **Check anti-patterns**: Rule may have exclude patterns blocking match

### False Positive in Tests

1. **Add to true_negatives**: Create test case with `expected_match: false`
2. **Add anti-pattern**: Update rule with `ExcludePatterns` or `SoftExcludePatterns`
3. **Tighten regex**: Make pattern more specific (add anchors, context)
4. **Add port constraint**: Restrict rule to specific ports

### Version Not Extracted

1. **Check VersionExtraction field**: Rule must have version regex
2. **Check regex pattern**: Must have capture group `(...)`
3. **Check banner format**: Version must match regex exactly
4. **Test regex**: Use online regex tester with actual banner

## References

- **Validation Framework**: [../validation.go](../validation.go)
- **Validation Tests**: [../validation_test.go](../validation_test.go)
- **Validation Report**: [../VALIDATION_REPORT.md](../VALIDATION_REPORT.md)
- **Fingerprint Rules**: [../data/fingerprint_db.yaml](../data/fingerprint_db.yaml)
- **Resolver Implementation**: [../resolver_rulebased.go](../resolver_rulebased.go)

## Questions?

For questions about validation dataset:
- Review [VALIDATION_REPORT.md](../VALIDATION_REPORT.md) for current metrics
- Check [validation_test.go](../validation_test.go) for test examples
- See [data/fingerprint_db.yaml](../data/fingerprint_db.yaml) for rule syntax

---

**Dataset Version**: 1.0
**Last Updated**: 2025-11-04
**Total Test Cases**: 130 (90 TP, 40 TN, 10 edge)
**Protocols Covered**: 16 (http, ssh, ftp, mysql, postgresql, redis, mongodb, memcached, elasticsearch, smtp, rdp, vnc, telnet, snmp, smb, dns)

---

## CI/CD Integration

The fingerprint validation workflow automatically runs on every PR that touches `pkg/fingerprint/**`. It compares validation metrics between the PR branch and main baseline, failing the workflow if regressions >5% are detected. Metrics output is visible in the workflow logs.

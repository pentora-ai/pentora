# Example YAML Plugins

This directory contains example YAML plugins that demonstrate the plugin system's capabilities.

## Plugins

### 1. SSH CVE-2024-TEST Detection
**File**: `ssh-cve-2024-test.yaml`
**Severity**: High
**Purpose**: Detects vulnerable SSH versions below 8.5

Demonstrates:
- Trigger based on data key existence
- Version comparison operators (`version_lt`)
- AND logic for multiple rules
- CVE metadata

### 2. HTTP Weak Security Headers
**File**: `http-weak-security-headers.yaml`
**Severity**: Medium
**Purpose**: Detects missing HTTP security headers

Demonstrates:
- Multiple OR rules
- Checking for missing/empty headers
- OWASP metadata
- Medium severity findings

### 3. TLS Weak Cipher Detection
**File**: `tls-weak-cipher.yaml`
**Severity**: High
**Purpose**: Detects weak or insecure TLS ciphers

Demonstrates:
- Multiple OR rules
- String contains operator
- Cipher detection patterns
- Cryptography-related findings

### 4. Default Credentials Detection
**File**: `default-credentials.yaml`
**Severity**: Critical
**Purpose**: Detects services with default or weak credentials

Demonstrates:
- Trigger with `in` operator
- Multiple authentication checks
- Critical severity
- CWE and OWASP references

### 5. Service Version Detection (Informational)
**File**: `version-detection-info.yaml`
**Severity**: Info
**Purpose**: Informational detection of service versions

Demonstrates:
- No triggers (runs on all data)
- Regex pattern matching
- Non-vulnerability findings
- Information disclosure category

## Usage

Load these plugins with the Loader:

```go
loader := plugin.NewLoader("./testdata/plugins")
plugins, err := loader.LoadAll("./testdata/plugins")
if err != nil {
    log.Fatal(err)
}

// Evaluate plugins against context
evaluator := plugin.NewEvaluator()
context := map[string]any{
    "ssh.version": "7.4.0",
    "ssh.banner":  "OpenSSH_7.4p1",
}

results, err := evaluator.EvaluateMatched(plugins, context)
```

## Creating Custom Plugins

See the plugin architecture design document at `.claude/features/plugin-architecture/README.md` for details on:
- Plugin schema structure
- Available operators
- Trigger conditions
- Match logic options

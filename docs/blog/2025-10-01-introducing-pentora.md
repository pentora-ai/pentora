---
slug: introducing-pentora
title: Introducing Pentora - Modern Network Security Scanner
authors: [pentora_team]
tags: [security, release, network-scanning]
image: /img/blog/pentora-launch.png
---

We're excited to announce the initial release of **Pentora**, a modern, high-performance network security scanner built for today's cloud-native infrastructure.

<!-- truncate -->

## Why We Built Pentora

Traditional network scanning tools were designed for a different era. Today's security teams face challenges that existing tools struggle to address:

- **Cloud-Native Infrastructure**: Dynamic environments with ephemeral workloads
- **Microservices Architecture**: Hundreds or thousands of services to track
- **Compliance Requirements**: Continuous auditing against multiple frameworks
- **DevSecOps Integration**: Security scanning in CI/CD pipelines

Pentora was built from the ground up to solve these modern challenges.

## Core Design Principles

### 1. Modular Architecture

At the heart of Pentora is a **DAG-based execution engine**. Every scan phase—from discovery to vulnerability assessment—is a composable module in a directed acyclic graph. This architecture provides:

- **Parallel Execution**: Independent modules run concurrently
- **Deterministic Results**: No race conditions or unpredictable behavior
- **Extensibility**: Add custom modules without modifying core code

```go
// Example module registration
registry.Register(&Module{
  ID:          "custom_scanner",
  Type:        "scan",
  DependsOn:   []string{"port_scan"},
  ExecuteFunc: customScanLogic,
})
```

### 2. Layered Fingerprinting

Accurate service identification is critical for vulnerability assessment. Pentora uses **layered fingerprinting** with protocol-specific probes:

1. **Initial Probe**: Quick banner grab
2. **Protocol Detection**: Identify service type
3. **Targeted Probes**: Protocol-specific queries (HTTP headers, TLS handshake, etc.)
4. **Confidence Scoring**: Multiple signals combined for accuracy

This approach achieves >95% accuracy in service identification compared to ~70% with traditional banner grabbing.

### 3. Workspace Persistence

Unlike traditional scanners that only output to stdout, Pentora maintains a **workspace** for scan history:

```
workspace/
├── scans/
│   ├── scan-2025-10-01-001/
│   │   ├── request.json
│   │   ├── status.json
│   │   └── results.jsonl
│   └── scan-2025-10-01-002/
├── cache/
│   └── fingerprints/
└── reports/
```

This enables:

- Historical comparison and trend analysis
- Automated compliance reporting
- Artifact storage for audit trails

### 4. Enterprise-Ready from Day One

While the open-source core provides powerful CLI-based scanning, we designed Pentora with enterprise scalability in mind:

- **Distributed Scanning**: Worker pools across multiple networks
- **Multi-Tenant Isolation**: Separate workspaces with RBAC
- **SIEM/SOAR Integration**: Native connectors for Splunk, QRadar, Sentinel
- **Compliance Packs**: CIS, PCI-DSS, NIST, HIPAA frameworks

## Getting Started

Installation is simple:

```bash
# Linux / macOS
curl -sSL https://pentora.io/install.sh | bash

# Verify
pentora version
```

Run your first scan:

```bash
# Basic network scan
pentora scan 192.168.1.0/24

# With vulnerability assessment
pentora scan 192.168.1.0/24 --vuln

# Discovery only
pentora scan 192.168.1.0/24 --only-discover
```

## What's Next

We're actively developing Pentora with several major features planned:

- **Q4 2025**: Web UI and REST API
- **Q1 2026**: Distributed scanning for Enterprise
- **Q2 2026**: Machine learning for anomaly detection
- **Q3 2026**: Integration marketplace

## Join the Community

Pentora is open source (Apache 2.0) and we welcome contributions:

- **GitHub**: [github.com/pentora/pentora](https://github.com/pentora-ai/pentora)
- **Discussions**: Share ideas and ask questions
- **Issues**: Report bugs or request features
- **Documentation**: [docs.pentora.io](https://docs.pentora.io)

## Try It Today

We encourage you to try Pentora and share your feedback. Whether you're a security engineer, DevOps practitioner, or compliance officer, we built Pentora to make your job easier.

Download now: [pentora.io/download](https://pentora.io/download)

---

_The Pentora Team_

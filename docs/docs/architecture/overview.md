# Architecture Overview

Pentora is built on a modular architecture centered around a DAG execution engine.

## High-Level Architecture

```
┌─────────────┐
│     CLI     │ ─────┐
└─────────────┘      │
                     ▼
┌─────────────┐   ┌──────────────┐   ┌─────────────┐
│  REST API   │──▶│ Orchestrator │──▶│   Modules   │
└─────────────┘   └──────────────┘   └─────────────┘
                     │         │
┌─────────────┐      │         │      ┌─────────────┐
│     UI      │ ─────┘         └─────▶│  Workspace  │
└─────────────┘                       └─────────────┘
```

## Core Components

### 1. CLI & API Layer
- Command-line interface for operators
- REST API for remote execution
- Authentication and authorization

### 2. Orchestrator (DAG Engine)
- Parses DAG definitions
- Manages module dependencies
- Coordinates parallel execution
- Handles failures and retries

### 3. Module System
- Discovery modules (ICMP, ARP, TCP)
- Scanner modules (SYN, Connect, Banner)
- Parser modules (HTTP, SSH, SMTP)
- Fingerprint modules
- Evaluation modules (CVE, compliance)
- Reporter modules (JSON, CSV, PDF)

### 4. Workspace
- Persistent scan storage
- Queue management (server mode)
- Cache (fingerprints, DNS)
- Audit logs (Enterprise)

### 5. Plugin System
- Embedded Go plugins
- External gRPC plugins
- WASM plugins (experimental)

See [Engine Internals](/docs/architecture/engine) for detailed design.

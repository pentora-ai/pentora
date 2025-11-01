# Pentora Documentation Index

Complete documentation for the Pentora security scanner.

## Getting Started (1 file)

- [Your First Scan](/getting-started/first-scan) - Step-by-step tutorial

## Core Concepts (6 files)

- [Overview](/concepts/overview) - Core concepts and philosophy
- [Scan Pipeline](/concepts/scan-pipeline) - 9-stage scan process
- [DAG Engine](/concepts/dag-engine) - Execution orchestration
- [Modules](/concepts/modules) - Module system and types
- [Fingerprinting](/concepts/fingerprinting) - Service detection

## CLI Reference (4 files)

- [CLI Overview](/cli/overview) - Command structure and usage
- [pentora scan](/cli/scan) - Scan command reference
- [pentora server](/cli/server) - Server control
- [pentora fingerprint](/cli/fingerprint) - Fingerprint management

## Configuration (3 files)

- [Configuration Overview](/configuration/overview) - Config structure
- [Scan Profiles](/configuration/scan-profiles) - Profile definitions
- [Logging](/configuration/logging) - Log configuration

## Architecture (5 files)

- [Architecture Overview](/architecture/overview) - High-level design
- [Engine Internals](/architecture/engine) - Planner, Orchestrator, Runtime
- [Module Architecture](/architecture/modules) - Module lifecycle
- [Plugin Architecture](/architecture/plugins) - Embedded vs External
- [Data Flow](/architecture/data-flow) - DataContext patterns

## Advanced Topics (4 files)

- [Custom Modules](/advanced/custom-modules) - Module development
- [External Plugins](/advanced/external-plugins) - gRPC/WASM plugins
- [Hooks & Events](/advanced/hooks-events) - Event system
- [Custom Fingerprints](/advanced/custom-fingerprints) - Fingerprint rules

## Enterprise Features (5 files)

- [Enterprise Overview](/enterprise/overview) - Features and pricing
- [Licensing](/enterprise/licensing) - JWT-based licensing
- [Distributed Scanning](/enterprise/distributed-scanning) - Worker pools
- [Multi-Tenant](/enterprise/multi-tenant) - Tenant isolation and RBAC
- [Integrations](/enterprise/integrations) - SIEM, ticketing, Slack

## Deployment (4 files)

- [Standalone CLI](/deployment/standalone) - Binary deployment
- [Server Mode](/deployment/server-mode) - Daemon deployment
- [Docker](/deployment/docker) - Container deployment
- [Air-Gapped](/deployment/air-gapped) - Offline deployment

## Guides (4 files)

- [Network Scanning](/guides/network-scanning) - Best practices
- [Vulnerability Assessment](/guides/vulnerability-assessment) - CVE analysis
- [Compliance Checks](/guides/compliance-checks) - CIS/PCI/NIST
- [Reporting](/guides/reporting) - Report customization

## Troubleshooting (3 files)

- [Common Issues](/troubleshooting/common-issues) - Solutions
- [Performance](/troubleshooting/performance) - Optimization
- [Debugging](/troubleshooting/debugging) - Debug mode

## API Reference

### REST API (4 files)

- [API Overview](/api/overview) - Authentication and basics
- [Authentication](/api/rest/authentication) - Token management
- [Scans API](/api/rest/scans) - Scan endpoints
- [Jobs API](/api/rest/jobs) - Job management (Enterprise)

### UI Portal (4 files)

- [UI Overview](/api/ui/overview) - Portal introduction
- [Dashboard](/api/ui/dashboard) - Dashboard features
- [Scan Management](/api/ui/scan-management) - Web-based scanning
- [Notifications](/api/ui/notifications) - Alert configuration

### Module API (3 files)

- [Module Interface](/api/modules/interface) - Module contract
- [DataContext](/api/modules/context) - Shared state API
- [Lifecycle](/api/modules/lifecycle) - Initialization and cleanup

## Key Features from NOTES.md

### Scan Pipeline (9 Stages)

1. Target Ingestion - Parse/validate targets
2. Asset Discovery - ICMP/ARP/TCP probes
3. Port Scanning - TCP/UDP with rate limiting
4. Service Fingerprinting - Layered protocol detection
5. Asset Profiling - Device/OS/app classification
6. Vulnerability Evaluation - CVE matching
7. Compliance & Risk Scoring - CIS/PCI/NIST
8. Reporting & Notification - JSON/PDF/integrations
9. Archival & Analytics - Storage persistence

### Pricing Tiers

- **Starter (OSS)**: Free - Full pipeline, CLI, local storage
- **Team**: $399/month - Scheduling, UI, webhooks, Slack
- **Business**: $1,499/month - Distributed, SIEM, SSO
- **Enterprise**: $80k-$120k/year - Multi-tenant, compliance, air-gapped

### Enterprise Features

- JWT-based licensing with offline grace period
- Distributed scanning with worker pools
- Multi-tenant storage with RBAC
- SSO integration (OIDC, SAML)
- SIEM integrations (Splunk, QRadar, Elastic)
- Ticketing (Jira, ServiceNow)
- Compliance packs (CIS, PCI DSS, NIST 800-53, HIPAA, ISO 27001)

## UI Features from UI-NOTES.md

### Dashboard

- Scan summary widgets
- Trend charts (ports, vulnerabilities, scans)
- System health (uptime, license, workers)
- Quick actions (start scan, schedule, invite)

### Scan Management

- Scan creation wizard (targets → profile → advanced → confirm)
- Real-time status timeline
- Target-level breakdown with manual notes
- Export to JSON/CSV/PDF, direct to Jira/ServiceNow

### Integrations

- Slack, Email, Webhooks
- SIEM: Splunk, QRadar, Elasticsearch
- Ticketing: Jira, ServiceNow
- Incident automation rules (Enterprise)

## Total Documentation

- **53 comprehensive documentation files**
- **20,000+ lines of documentation**
- **Complete coverage of CLI, API, UI, and Enterprise features**
- **Based on NOTES.md and UI-NOTES.md specifications**

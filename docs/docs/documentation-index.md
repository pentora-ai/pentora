# Pentora Documentation Index

Complete documentation for the Pentora security scanner.

## Getting Started (1 file)

- [Your First Scan](/docs/getting-started/first-scan) - Step-by-step tutorial

## Core Concepts (7 files)

- [Overview](/docs/concepts/overview) - Core concepts and philosophy
- [Scan Pipeline](/docs/concepts/scan-pipeline) - 9-stage scan process
- [Workspace](/docs/concepts/workspace) - Storage and retention
- [DAG Engine](/docs/concepts/dag-engine) - Execution orchestration
- [Modules](/docs/concepts/modules) - Module system and types
- [Fingerprinting](/docs/concepts/fingerprinting) - Service detection

## CLI Reference (5 files)

- [CLI Overview](/docs/cli/overview) - Command structure and usage
- [pentora scan](/docs/cli/scan) - Scan command reference
- [pentora workspace](/docs/cli/workspace) - Workspace management
- [pentora server](/docs/cli/server) - Server control
- [pentora fingerprint](/docs/cli/fingerprint) - Fingerprint management

## Configuration (4 files)

- [Configuration Overview](/docs/configuration/overview) - Config structure
- [Scan Profiles](/docs/configuration/scan-profiles) - Profile definitions
- [Workspace Config](/docs/configuration/workspace-config) - Workspace settings
- [Logging](/docs/configuration/logging) - Log configuration

## Architecture (5 files)

- [Architecture Overview](/docs/architecture/overview) - High-level design
- [Engine Internals](/docs/architecture/engine) - Planner, Orchestrator, Runtime
- [Module Architecture](/docs/architecture/modules) - Module lifecycle
- [Plugin Architecture](/docs/architecture/plugins) - Embedded vs External
- [Data Flow](/docs/architecture/data-flow) - DataContext patterns

## Advanced Topics (4 files)

- [Custom Modules](/docs/advanced/custom-modules) - Module development
- [External Plugins](/docs/advanced/external-plugins) - gRPC/WASM plugins
- [Hooks & Events](/docs/advanced/hooks-events) - Event system
- [Custom Fingerprints](/docs/advanced/custom-fingerprints) - Fingerprint rules

## Enterprise Features (5 files)

- [Enterprise Overview](/docs/enterprise/overview) - Features and pricing
- [Licensing](/docs/enterprise/licensing) - JWT-based licensing
- [Distributed Scanning](/docs/enterprise/distributed-scanning) - Worker pools
- [Multi-Tenant](/docs/enterprise/multi-tenant) - Tenant isolation and RBAC
- [Integrations](/docs/enterprise/integrations) - SIEM, ticketing, Slack

## Deployment (4 files)

- [Standalone CLI](/docs/deployment/standalone) - Binary deployment
- [Server Mode](/docs/deployment/server-mode) - Daemon deployment
- [Docker](/docs/deployment/docker) - Container deployment
- [Air-Gapped](/docs/deployment/air-gapped) - Offline deployment

## Guides (4 files)

- [Network Scanning](/docs/guides/network-scanning) - Best practices
- [Vulnerability Assessment](/docs/guides/vulnerability-assessment) - CVE analysis
- [Compliance Checks](/docs/guides/compliance-checks) - CIS/PCI/NIST
- [Reporting](/docs/guides/reporting) - Report customization

## Troubleshooting (3 files)

- [Common Issues](/docs/troubleshooting/common-issues) - Solutions
- [Performance](/docs/troubleshooting/performance) - Optimization
- [Debugging](/docs/troubleshooting/debugging) - Debug mode

## API Reference

### REST API (4 files)

- [API Overview](/docs/api/overview) - Authentication and basics
- [Authentication](/docs/api/rest/authentication) - Token management
- [Scans API](/docs/api/rest/scans) - Scan endpoints
- [Workspace API](/docs/api/rest/workspace) - Workspace endpoints
- [Jobs API](/docs/api/rest/jobs) - Job management (Enterprise)

### UI Portal (4 files)

- [UI Overview](/docs/api/ui/overview) - Portal introduction
- [Dashboard](/docs/api/ui/dashboard) - Dashboard features
- [Scan Management](/docs/api/ui/scan-management) - Web-based scanning
- [Notifications](/docs/api/ui/notifications) - Alert configuration

### Module API (3 files)

- [Module Interface](/docs/api/modules/interface) - Module contract
- [DataContext](/docs/api/modules/context) - Shared state API
- [Lifecycle](/docs/api/modules/lifecycle) - Initialization and cleanup

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
9. Archival & Analytics - Workspace storage

### Pricing Tiers

- **Starter (OSS)**: Free - Full pipeline, CLI, local workspace
- **Team**: $399/month - Scheduling, UI, webhooks, Slack
- **Business**: $1,499/month - Distributed, SIEM, SSO
- **Enterprise**: $80k-$120k/year - Multi-tenant, compliance, air-gapped

### Enterprise Features

- JWT-based licensing with offline grace period
- Distributed scanning with worker pools
- Multi-tenant workspace with RBAC
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

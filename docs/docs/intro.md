---
slug: /
sidebar_position: 1
---

# Introduction to Vulntor

<div style={{textAlign: 'center', margin: '2rem 0'}}>
  <img src="/img/vulntor-banner.svg" alt="Vulntor Banner" style={{maxWidth: '600px', width: '100%'}} />
</div>

<div className="card" style={{
  background: 'linear-gradient(135deg, rgba(0, 217, 255, 0.1) 0%, rgba(0, 217, 255, 0.05) 100%)',
  border: '2px solid rgba(0, 217, 255, 0.3)',
  borderRadius: '12px',
  padding: '1.5rem',
  marginBottom: '2rem'
}}>
  <h3 style={{color: '#00d9ff', marginTop: 0}}>What is Vulntor?</h3>
  <span>
    <strong>Vulntor</strong> is a modular, high-performance security scanner that rapidly discovers network services, captures banners, and maps findings into vulnerability intelligence. Built with a powerful DAG-based execution engine, Vulntor enables security teams to perform comprehensive network assessments with precision and efficiency.
  </span>
</div>

## What Makes Vulntor Special?

<div className="row equal-height-row" style={{marginTop: '1.5rem'}}>
  <div className="col col--4">
    <div className="card" style={{
      height: '100%',
      background: 'linear-gradient(135deg, rgba(0, 217, 255, 0.1) 0%, rgba(0, 217, 255, 0.05) 100%)',
      border: '2px solid rgba(0, 217, 255, 0.3)',
      borderRadius: '12px',
      transition: 'all 0.3s ease',
      boxShadow: '0 4px 6px rgba(0, 217, 255, 0.1)'
    }}>
      <div className="card__body" style={{textAlign: 'center', padding: '2rem 1.5rem'}}>
        <div style={{fontSize: '3rem', marginBottom: '1rem'}}>⚡</div>
        <h3 style={{color: '#00d9ff', marginBottom: '1rem'}}>Lightning Fast</h3>
        <span style={{color: 'var(--ifm-color-emphasis-700)', margin: 0}}>
          Efficiently discover live hosts using ICMP/ARP/TCP probes with intelligent rate limiting
        </span>
      </div>
    </div>
  </div>
  <div className="col col--4">
    <div className="card" style={{
      height: '100%',
      background: 'linear-gradient(135deg, rgba(139, 92, 246, 0.1) 0%, rgba(139, 92, 246, 0.05) 100%)',
      border: '2px solid rgba(139, 92, 246, 0.3)',
      borderRadius: '12px',
      transition: 'all 0.3s ease',
      boxShadow: '0 4px 6px rgba(139, 92, 246, 0.1)'
    }}>
      <div className="card__body" style={{textAlign: 'center', padding: '2rem 1.5rem'}}>
        <div style={{fontSize: '3rem', marginBottom: '1rem'}}>🎯</div>
        <h3 style={{color: '#8b5cf6', marginBottom: '1rem'}}>Accurate</h3>
        <span style={{color: 'var(--ifm-color-emphasis-700)', margin: 0}}>
          Protocol-specific probes with layered fingerprinting and confidence scoring
        </span>
      </div>
    </div>
  </div>
  <div className="col col--4">
    <div className="card" style={{
      height: '100%',
      background: 'linear-gradient(135deg, rgba(16, 185, 129, 0.1) 0%, rgba(16, 185, 129, 0.05) 100%)',
      border: '2px solid rgba(16, 185, 129, 0.3)',
      borderRadius: '12px',
      transition: 'all 0.3s ease',
      boxShadow: '0 4px 6px rgba(16, 185, 129, 0.1)'
    }}>
      <div className="card__body" style={{textAlign: 'center', padding: '2rem 1.5rem'}}>
        <div style={{fontSize: '3rem', marginBottom: '1rem'}}>🔧</div>
        <h3 style={{color: '#10b981', marginBottom: '1rem'}}>Modular</h3>
        <span style={{color: 'var(--ifm-color-emphasis-700)', margin: 0}}>
          Extensible plugin system for custom scanning logic and integrations
        </span>
      </div>
    </div>
  </div>
</div>

### Key Capabilities

<div className="timeline" style={{position: 'relative', maxWidth: '900px', margin: '3rem auto'}}>
  <div style={{position: 'absolute', left: '50%', transform: 'translateX(-50%)', top: 0, bottom: 0, width: '3px', background: 'linear-gradient(180deg, #00d9ff 0%, #6366f1 100%)'}}></div>

  <div style={{position: 'relative', marginBottom: '3rem', paddingRight: '50%', paddingLeft: 0}}>
    <div style={{textAlign: 'right', paddingRight: '2.5rem'}}>
      <h4 style={{color: '#00d9ff', marginBottom: '0.5rem'}}>🔍 Fast Network Discovery</h4>
      <p style={{color: 'var(--ifm-color-emphasis-700)', marginBottom: 0}}>Efficiently discover live hosts using ICMP/ARP/TCP probes</p>
    </div>
    <div style={{position: 'absolute', right: 'calc(50% - 10px)', top: '5px', width: '20px', height: '20px', borderRadius: '50%', background: '#00d9ff', border: '3px solid var(--ifm-background-color)', boxShadow: '0 0 10px rgba(0,217,255,0.5)', zIndex: 1}}></div>
  </div>

  <div style={{position: 'relative', marginBottom: '3rem', paddingLeft: '50%', paddingRight: 0}}>
    <div style={{textAlign: 'left', paddingLeft: '2.5rem'}}>
      <h4 style={{color: '#4a9eff', marginBottom: '0.5rem'}}>🎯 Advanced Port Scanning</h4>
      <p style={{color: 'var(--ifm-color-emphasis-700)', marginBottom: 0}}>TCP/UDP scanning with intelligent rate limiting and retry logic</p>
    </div>
    <div style={{position: 'absolute', left: 'calc(50% - 10px)', top: '5px', width: '20px', height: '20px', borderRadius: '50%', background: '#4a9eff', border: '3px solid var(--ifm-background-color)', boxShadow: '0 0 10px rgba(74,158,255,0.5)', zIndex: 1}}></div>
  </div>

  <div style={{position: 'relative', marginBottom: '3rem', paddingRight: '50%', paddingLeft: 0}}>
    <div style={{textAlign: 'right', paddingRight: '2.5rem'}}>
      <h4 style={{color: '#8b5cf6', marginBottom: '0.5rem'}}>🔬 Layered Fingerprinting</h4>
      <p style={{color: 'var(--ifm-color-emphasis-700)', marginBottom: 0}}>Protocol-specific probes that accurately identify services and versions</p>
    </div>
    <div style={{position: 'absolute', right: 'calc(50% - 10px)', top: '5px', width: '20px', height: '20px', borderRadius: '50%', background: '#8b5cf6', border: '3px solid var(--ifm-background-color)', boxShadow: '0 0 10px rgba(139,92,246,0.5)', zIndex: 1}}></div>
  </div>

  <div style={{position: 'relative', marginBottom: '3rem', paddingLeft: '50%', paddingRight: 0}}>
    <div style={{textAlign: 'left', paddingLeft: '2.5rem'}}>
      <h4 style={{color: '#10b981', marginBottom: '0.5rem'}}>🛡️ Vulnerability Intelligence</h4>
      <p style={{color: 'var(--ifm-color-emphasis-700)', marginBottom: 0}}>Match detected services against CVE databases and misconfigurations</p>
    </div>
    <div style={{position: 'absolute', left: 'calc(50% - 10px)', top: '5px', width: '20px', height: '20px', borderRadius: '50%', background: '#10b981', border: '3px solid var(--ifm-background-color)', boxShadow: '0 0 10px rgba(16,185,129,0.5)', zIndex: 1}}></div>
  </div>

  <div style={{position: 'relative', marginBottom: '3rem', paddingRight: '50%', paddingLeft: 0}}>
    <div style={{textAlign: 'right', paddingRight: '2.5rem'}}>
      <h4 style={{color: '#f59e0b', marginBottom: '0.5rem'}}>✅ Compliance Assessment</h4>
      <p style={{color: 'var(--ifm-color-emphasis-700)', marginBottom: 0}}>Built-in support for CIS, PCI-DSS, and NIST compliance frameworks</p>
    </div>
    <div style={{position: 'absolute', right: 'calc(50% - 10px)', top: '5px', width: '20px', height: '20px', borderRadius: '50%', background: '#f59e0b', border: '3px solid var(--ifm-background-color)', boxShadow: '0 0 10px rgba(245,158,11,0.5)', zIndex: 1}}></div>
  </div>

  <div style={{position: 'relative', marginBottom: '3rem', paddingLeft: '50%', paddingRight: 0}}>
    <div style={{textAlign: 'left', paddingLeft: '2.5rem'}}>
      <h4 style={{color: '#ec4899', marginBottom: '0.5rem'}}>🧩 Modular Architecture</h4>
      <p style={{color: 'var(--ifm-color-emphasis-700)', marginBottom: 0}}>Extensible plugin system for custom scanning logic</p>
    </div>
    <div style={{position: 'absolute', left: 'calc(50% - 10px)', top: '5px', width: '20px', height: '20px', borderRadius: '50%', background: '#ec4899', border: '3px solid var(--ifm-background-color)', boxShadow: '0 0 10px rgba(236,72,153,0.5)', zIndex: 1}}></div>
  </div>

  <div style={{position: 'relative', marginBottom: '0', paddingRight: '50%', paddingLeft: 0}}>
    <div style={{textAlign: 'right', paddingRight: '2.5rem'}}>
      <h4 style={{color: '#6366f1', marginBottom: '0.5rem'}}>💾 Storage Management</h4>
      <p style={{color: 'var(--ifm-color-emphasis-700)', marginBottom: 0}}>Persistent storage for scan history, results, and analytics</p>
    </div>
    <div style={{position: 'absolute', right: 'calc(50% - 10px)', top: '5px', width: '20px', height: '20px', borderRadius: '50%', background: '#6366f1', border: '3px solid var(--ifm-background-color)', boxShadow: '0 0 10px rgba(99,102,241,0.5)', zIndex: 1}}></div>
  </div>
</div>

## Who Should Use Vulntor?

<div className="row" style={{marginTop: '1.5rem', marginBottom: '2rem'}}>
  <div className="col col--6" style={{marginBottom: '1rem'}}>
    <div className="card" style={{height: '100%'}}>
      <div className="card__header">
        <h3>🔐 Security Professionals</h3>
      </div>
      <div className="card__body">
        <p>Technical operators who need powerful CLI tools for network assessments, penetration testing, and security audits.</p>
      </div>
    </div>
  </div>
  <div className="col col--6" style={{marginBottom: '1rem'}}>
    <div className="card" style={{height: '100%'}}>
      <div className="card__header">
        <h3>⚙️ DevSecOps Teams</h3>
      </div>
      <div className="card__body">
        <p>Teams integrating security scanning into CI/CD pipelines with automated vulnerability detection.</p>
      </div>
    </div>
  </div>
  <div className="col col--6" style={{marginBottom: '1rem'}}>
    <div className="card" style={{height: '100%'}}>
      <div className="card__header">
        <h3>📋 Compliance Officers</h3>
      </div>
      <div className="card__body">
        <p>Organizations requiring regular compliance scans against industry standards (CIS, PCI-DSS, NIST).</p>
      </div>
    </div>
  </div>
  <div className="col col--6" style={{marginBottom: '1rem'}}>
    <div className="card" style={{height: '100%'}}>
      <div className="card__header">
        <h3>🏢 Enterprise Security Teams</h3>
      </div>
      <div className="card__body">
        <p>Large organizations needing distributed scanning, multi-tenant storage, and SIEM/SOAR integrations.</p>
      </div>
    </div>
  </div>
</div>

## Core Philosophy

**Design Principles** - Vulntor is built on five core principles:

1. 🧩 **Modularity**: Every scan phase is a composable module in a directed acyclic graph (DAG)
2. ⚡ **Performance**: Concurrent execution with intelligent rate limiting
3. 🎯 **Accuracy**: Layered fingerprinting with confidence scoring
4. 🔄 **Flexibility**: Both stateless (Nmap-style) and storage-backed operations
5. 📊 **Transparency**: Structured logging and comprehensive audit trails

## Key Features

<div className="row equal-height-row" style={{marginTop: '2rem'}}>
  <div className="col col--6">
    <div className="card" style={{
      height: '100%',
      background: 'linear-gradient(135deg, rgba(16, 185, 129, 0.1) 0%, rgba(16, 185, 129, 0.05) 100%)',
      border: '2px solid rgba(16, 185, 129, 0.3)',
      borderRadius: '12px'
    }}>
      <div className="card__header" style={{background: 'rgba(16, 185, 129, 0.1)', borderBottom: '1px solid rgba(16, 185, 129, 0.2)', padding: '1rem 1.5rem'}}>
        <h3 style={{color: '#10b981', margin: 0, fontSize: '1.25rem'}}>✅ Open Source Core</h3>
      </div>
      <div className="card__body" style={{padding: '1rem 1.5rem'}}>
        <ul style={{listStyle: 'none', padding: 0, margin: 0}}>
          <li style={{padding: '0.5rem 0', borderBottom: '1px solid rgba(16, 185, 129, 0.1)', display: 'flex', alignItems: 'flex-start', gap: '0.75rem'}}>
            <span style={{color: '#10b981', fontSize: '1rem', marginTop: '0.1rem', flexShrink: 0}}>✓</span>
            <span style={{fontSize: '0.95rem'}}>CLI-based scanning with full pipeline control</span>
          </li>
          <li style={{padding: '0.5rem 0', borderBottom: '1px solid rgba(16, 185, 129, 0.1)', display: 'flex', alignItems: 'flex-start', gap: '0.75rem'}}>
            <span style={{color: '#10b981', fontSize: '1rem', marginTop: '0.1rem', flexShrink: 0}}>✓</span>
            <span style={{fontSize: '0.95rem'}}>Asset discovery and port scanning</span>
          </li>
          <li style={{padding: '0.5rem 0', borderBottom: '1px solid rgba(16, 185, 129, 0.1)', display: 'flex', alignItems: 'flex-start', gap: '0.75rem'}}>
            <span style={{color: '#10b981', fontSize: '1rem', marginTop: '0.1rem', flexShrink: 0}}>✓</span>
            <span style={{fontSize: '0.95rem'}}>Service fingerprinting with extensible probe system</span>
          </li>
          <li style={{padding: '0.5rem 0', borderBottom: '1px solid rgba(16, 185, 129, 0.1)', display: 'flex', alignItems: 'flex-start', gap: '0.75rem'}}>
            <span style={{color: '#10b981', fontSize: '1rem', marginTop: '0.1rem', flexShrink: 0}}>✓</span>
            <span style={{fontSize: '0.95rem'}}>Vulnerability evaluation against CVE databases</span>
          </li>
          <li style={{padding: '0.5rem 0', borderBottom: '1px solid rgba(16, 185, 129, 0.1)', display: 'flex', alignItems: 'flex-start', gap: '0.75rem'}}>
            <span style={{color: '#10b981', fontSize: '1rem', marginTop: '0.1rem', flexShrink: 0}}>✓</span>
            <span style={{fontSize: '0.95rem'}}>Storage for scan history and result storage</span>
          </li>
          <li style={{padding: '0.5rem 0', borderBottom: '1px solid rgba(16, 185, 129, 0.1)', display: 'flex', alignItems: 'flex-start', gap: '0.75rem'}}>
            <span style={{color: '#10b981', fontSize: '1rem', marginTop: '0.1rem', flexShrink: 0}}>✓</span>
            <span style={{fontSize: '0.95rem'}}>JSON/CSV/PDF export formats</span>
          </li>
          <li style={{padding: '0.5rem 0', display: 'flex', alignItems: 'flex-start', gap: '0.75rem'}}>
            <span style={{color: '#10b981', fontSize: '1rem', marginTop: '0.1rem', flexShrink: 0}}>✓</span>
            <span style={{fontSize: '0.95rem'}}>Hook system for custom integrations</span>
          </li>
        </ul>
      </div>
    </div>
  </div>

  <div className="col col--6">
    <div className="card" style={{
      height: '100%',
      background: 'linear-gradient(135deg, rgba(139, 92, 246, 0.1) 0%, rgba(139, 92, 246, 0.05) 100%)',
      border: '2px solid rgba(139, 92, 246, 0.3)',
      borderRadius: '12px'
    }}>
      <div className="card__header" style={{background: 'rgba(139, 92, 246, 0.1)', borderBottom: '1px solid rgba(139, 92, 246, 0.2)', padding: '1rem 1.5rem'}}>
        <h3 style={{color: '#8b5cf6', margin: 0, fontSize: '1.25rem'}}>🔒 Enterprise Edition</h3>
      </div>
      <div className="card__body" style={{padding: '1rem 1.5rem'}}>
        <ul style={{listStyle: 'none', padding: 0, margin: 0}}>
          <li style={{padding: '0.5rem 0', borderBottom: '1px solid rgba(139, 92, 246, 0.1)', display: 'flex', alignItems: 'flex-start', gap: '0.75rem'}}>
            <span style={{color: '#8b5cf6', fontSize: '1rem', marginTop: '0.1rem', flexShrink: 0}}>★</span>
            <span style={{fontSize: '0.95rem'}}>Distributed scanning across worker pools</span>
          </li>
          <li style={{padding: '0.5rem 0', borderBottom: '1px solid rgba(139, 92, 246, 0.1)', display: 'flex', alignItems: 'flex-start', gap: '0.75rem'}}>
            <span style={{color: '#8b5cf6', fontSize: '1rem', marginTop: '0.1rem', flexShrink: 0}}>★</span>
            <span style={{fontSize: '0.95rem'}}>Multi-tenant storage isolation</span>
          </li>
          <li style={{padding: '0.5rem 0', borderBottom: '1px solid rgba(139, 92, 246, 0.1)', display: 'flex', alignItems: 'flex-start', gap: '0.75rem'}}>
            <span style={{color: '#8b5cf6', fontSize: '1rem', marginTop: '0.1rem', flexShrink: 0}}>★</span>
            <span style={{fontSize: '0.95rem'}}>Role-based access control (RBAC) and SSO</span>
          </li>
          <li style={{padding: '0.5rem 0', borderBottom: '1px solid rgba(139, 92, 246, 0.1)', display: 'flex', alignItems: 'flex-start', gap: '0.75rem'}}>
            <span style={{color: '#8b5cf6', fontSize: '1rem', marginTop: '0.1rem', flexShrink: 0}}>★</span>
            <span style={{fontSize: '0.95rem'}}>Advanced compliance packs (CIS/PCI/NIST)</span>
          </li>
          <li style={{padding: '0.5rem 0', borderBottom: '1px solid rgba(139, 92, 246, 0.1)', display: 'flex', alignItems: 'flex-start', gap: '0.75rem'}}>
            <span style={{color: '#8b5cf6', fontSize: '1rem', marginTop: '0.1rem', flexShrink: 0}}>★</span>
            <span style={{fontSize: '0.95rem'}}>SIEM/SOAR integrations (Splunk, QRadar, Sentinel)</span>
          </li>
          <li style={{padding: '0.5rem 0', borderBottom: '1px solid rgba(139, 92, 246, 0.1)', display: 'flex', alignItems: 'flex-start', gap: '0.75rem'}}>
            <span style={{color: '#8b5cf6', fontSize: '1rem', marginTop: '0.1rem', flexShrink: 0}}>★</span>
            <span style={{fontSize: '0.95rem'}}>Ticketing system integration (Jira, ServiceNow)</span>
          </li>
          <li style={{padding: '0.5rem 0', borderBottom: '1px solid rgba(139, 92, 246, 0.1)', display: 'flex', alignItems: 'flex-start', gap: '0.75rem'}}>
            <span style={{color: '#8b5cf6', fontSize: '1rem', marginTop: '0.1rem', flexShrink: 0}}>★</span>
            <span style={{fontSize: '0.95rem'}}>Web portal with dashboards and scheduling</span>
          </li>
          <li style={{padding: '0.5rem 0', borderBottom: '1px solid rgba(139, 92, 246, 0.1)', display: 'flex', alignItems: 'flex-start', gap: '0.75rem'}}>
            <span style={{color: '#8b5cf6', fontSize: '1rem', marginTop: '0.1rem', flexShrink: 0}}>★</span>
            <span style={{fontSize: '0.95rem'}}>Air-gapped deployment support</span>
          </li>
          <li style={{padding: '0.5rem 0', display: 'flex', alignItems: 'flex-start', gap: '0.75rem'}}>
            <span style={{color: '#8b5cf6', fontSize: '1rem', marginTop: '0.1rem', flexShrink: 0}}>★</span>
            <span style={{fontSize: '0.95rem'}}>License-managed plugin marketplace</span>
          </li>
        </ul>
      </div>
    </div>
  </div>
</div>

## Quick Start

```bash title="Install Vulntor"
curl -sSL https://vulntor.io/install.sh | bash
```

```bash title="Run a basic scan"
vulntor scan 192.168.1.0/24
```

```bash title="Scan with vulnerability assessment"
vulntor scan 192.168.1.100 --vuln
```

```bash title="Discovery-only mode"
vulntor scan 192.168.1.0/24 --only-discover
```

```bash title="View storage scans"
vulntor storage list
```

## Architecture Overview

Vulntor uses a **DAG-based execution engine** where each scan phase is represented as a node:

<div style={{display: 'flex', justifyContent: 'center', margin: '2rem 0'}}>
  <img src="/img/dag-sketch.svg" alt="DAG Pipeline Sketch" style={{maxWidth: '800px', width: '100%', borderRadius: '8px'}} />
</div>

### Module Types

<div className="row equal-height-row">
  <div className="col col--4">
    <div className="card">
      <div className="card__header">
        <h4>🔧 Embedded</h4>
      </div>
      <div className="card__body">
        <p>Built-in Go code for maximum performance</p>
      </div>
    </div>
  </div>
  <div className="col col--4">
    <div className="card">
      <div className="card__header">
        <h4>🔌 External</h4>
      </div>
      <div className="card__body">
        <p>Isolated plugins via gRPC or WASM</p>
      </div>
    </div>
  </div>
  <div className="col col--4">
    <div className="card">
      <div className="card__header">
        <h4>✏️ Custom</h4>
      </div>
      <div className="card__body">
        <p>User-defined modules for specific needs</p>
      </div>
    </div>
  </div>
</div>

## Use Cases

<div className="row" style={{marginTop: '1.5rem', display: 'flex', flexWrap: 'wrap', alignItems: 'stretch' }}>
  <div className="col col--6" style={{display: 'flex', marginBottom: '1rem'}}>
    <div className="card">
      <div className="card__header">
        <h3>🔍 Network Asset Discovery</h3>
      </div>
      <div className="card__body">
        <p>Identify all active devices, open ports, and running services across your network infrastructure.</p>
      </div>
    </div>
  </div>
  <div className="col col--6" style={{display: 'flex', marginBottom: '1rem'}}>
    <div className="card">
      <div className="card__header">
        <h3>🛡️ Vulnerability Assessment</h3>
      </div>
      <div className="card__body">
        <p>Detect vulnerable service versions, misconfigurations, and CVE matches before attackers do.</p>
      </div>
    </div>
  </div>
  <div className="col col--6" style={{display: 'flex', marginBottom: '1rem'}}>
    <div className="card">
      <div className="card__header">
        <h3>✅ Compliance Auditing</h3>
      </div>
      <div className="card__body">
        <p>Generate compliance reports for PCI-DSS, CIS benchmarks, NIST frameworks, and custom policies.</p>
      </div>
    </div>
  </div>
  <div className="col col--6" style={{display: 'flex', marginBottom: '1rem'}}>
    <div className="card">
      <div className="card__header">
        <h3>📊 Continuous Monitoring</h3>
      </div>
      <div className="card__body">
        <p>Schedule recurring scans and integrate with SIEM/ticketing systems for automated incident response.</p>
      </div>
    </div>
  </div>
  <div className="col col--6" style={{display: 'flex', marginBottom: '1rem'}}>
    <div className="card">
      <div className="card__header">
        <h3>👻 Shadow IT Detection</h3>
      </div>
      <div className="card__body">
        <p>Discover unauthorized services, outdated software, and security policy violations.</p>
      </div>
    </div>
  </div>
</div>

## Getting Started

**Ready to dive in?** Head over to the [Installation Guide](./getting-started/installation.md) to install Vulntor, or jump to the [Quick Start Guide](./getting-started/quick-start.md) to run your first scan.

<div className="row" style={{marginTop: '2rem'}}>
  <div className="col col--4">
    <div className="card" style={{textAlign: 'center', height: '100%'}}>
      <div className="card__body">
        <h3>📥 Install</h3>
        <p>Get Vulntor up and running</p>
        <a href="getting-started/installation" className="button button--primary">Install Now</a>
      </div>
    </div>
  </div>
  <div className="col col--4">
    <div className="card" style={{textAlign: 'center', height: '100%'}}>
      <div className="card__body">
        <h3>🚀 Quick Start</h3>
        <p>Run your first scan</p>
        <a href="getting-started/quick-start" className="button button--primary">Get Started</a>
      </div>
    </div>
  </div>
  <div className="col col--4">
    <div className="card" style={{textAlign: 'center', height: '100%'}}>
      <div className="card__body">
        <h3>📖 Learn More</h3>
        <p>Explore core concepts</p>
        <a href="concepts/overview" className="button button--primary">Learn More</a>
      </div>
    </div>
  </div>
</div>

## Community & Support

<div className="row equal-height-row" style={{marginTop: '1.5rem'}}>
  <div className="col col--3">
    <a href="https://docs.vulntor.io" className="card" style={{textDecoration: 'none', height: '100%'}}>
      <div className="card__body" style={{textAlign: 'center'}}>
        <div style={{fontSize: '2rem'}}>📖</div>
        <h4>Documentation</h4>
      </div>
    </a>
  </div>
  <div className="col col--3">
    <a href="https://github.com/vulntor-ai/vulntor/discussions" className="card" style={{textDecoration: 'none', height: '100%'}}>
      <div className="card__body" style={{textAlign: 'center'}}>
        <div style={{fontSize: '2rem'}}>💬</div>
        <h4>Discussions</h4>
      </div>
    </a>
  </div>
  <div className="col col--3">
    <a href="https://github.com/vulntor-ai/vulntor/issues" className="card" style={{textDecoration: 'none', height: '100%'}}>
      <div className="card__body" style={{textAlign: 'center'}}>
        <div style={{fontSize: '2rem'}}>🐛</div>
        <h4>Issue Tracker</h4>
      </div>
    </a>
  </div>
  <div className="col col--3">
    <a href="https://github.com/vulntor-ai/vulntor/security/policy" className="card" style={{textDecoration: 'none', height: '100%'}}>
      <div className="card__body" style={{textAlign: 'center'}}>
        <div style={{fontSize: '2rem'}}>🔒</div>
        <h4>Security Policy</h4>
      </div>
    </a>
  </div>
</div>

---

> **Note:** Vulntor is actively developed. Features marked with 🔒 are available in the Enterprise Edition. Check the [Pricing Page](https://vulntor.io/pricing) for licensing options.

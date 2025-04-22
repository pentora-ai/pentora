![Pentora](https://pentora.ai/logo.png)

# Pentora

[![Go Report Card](https://goreportcard.com/badge/github.com/pentoraai/pentora)](https://goreportcard.com/report/github.com/pentoraai/pentora)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build](https://img.shields.io/github/actions/workflow/status/pentoraai/pentora/test.yml?branch=main)](https://github.com/pentoraai/pentora/actions)

**Pentora** is a fast, modular, and extensible security scanner designed for modern networks. It detects open ports, collects service banners, and performs optional CVE matching for basic vulnerability assessments.

## 🚀 Features

- 🔍 Fast port scanning on target IPs
- 📦 Modular service detection with parsers (SSH, HTTP, FTP, ...)
- 🔐 Plugin-based CVE matching (optional)
- 💻 Lightweight CLI interface
- 🌐 API and Web UI support (WIP)
- ⚙️ Cross-platform binaries (macOS, Linux, Windows)

## 🧪 CLI Usage

```bash
# Basic scan (default ports: 22,80,443)
pentora scan 192.168.1.1

# Custom ports
pentora scan 192.168.1.1 --ports 21,22,80,8080

# Port discovery (range 1–1000)
pentora scan 192.168.1.1 --discover

# Scan with CVE plugin matching
tentora scan 192.168.1.1 --ports 22 --vuln
```

## 🧩 Plugin System

Pentora uses modular plugins to evaluate specific vulnerabilities based on banner detection:

```go
plugin.Register(&plugin.Plugin{
  ID: "ssh_cve_2016_0777",
  Name: "OpenSSH 7.1p2 Vulnerability",
  RequirePorts: []int{22},
  RequireKeys: []string{"ssh/banner"},
  MatchFunc: func(ctx map[string]string) *plugin.MatchResult {
    if strings.Contains(ctx["ssh/banner"], "OpenSSH_7.1p2") {
      return &plugin.MatchResult{
        CVE: []string{"CVE-2016-0777"},
        Summary: "Roaming enabled vulnerability in OpenSSH",
        Port: 22,
        Info: ctx["ssh/banner"],
      }
    }
    return nil
  },
})
```

## 🔧 Installation

```bash
git clone https://github.com/pentoraai/pentora.git
cd pentora
go build -o pentora ./cmd/pentora
```

> Or build cross-platform installers via `make pkg` or `make dmg`

## 📂 Project Structure

- `cmd/pentora` – CLI entrypoint
- `cli/` – Cobra-based command definitions
- `scanner/` – Port scanner and banner probe engine
- `parser/` – Protocol-aware service detection
- `plugin/` – Rule-based CVE matching framework
- `api/` – REST API (WIP)
- `ui/` – Web UI (WIP)

## 📜 License

MIT License. See the `LICENSE` file for more details.

---

Join the community or contribute to the project 💬

> https://pentora.ai

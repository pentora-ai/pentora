# CLI Configuration

Learn how to configure Pentora CLI using configuration files, environment variables, and command-line flags.

## Configuration Precedence

Configuration loaded in order (later overrides earlier):

1. Builtin defaults
2. System config: `/etc/pentora/config.yaml`
3. User config: `~/.config/pentora/config.yaml`
4. Workspace config: `<workspace>/config/pentora.yaml`
5. Custom config: `--config /path/to/config.yaml`
6. Environment variables: `PENTORA_*`
7. CLI flags: `--flag value`

## Global Flags

Available on all commands:

```bash
--config string          Config file path (default: ~/.config/pentora/config.yaml)
--workspace-dir string   Workspace directory (default: OS-specific)
--no-workspace          Disable workspace persistence
--log-level string      Logging level: debug, info, warn, error (default: info)
--log-format string     Log format: json, text (default: text)
--verbosity int         Increase logging verbosity (0-3)
--quiet                 Suppress non-error output
--no-color              Disable colored output
--help, -h              Show help
--version, -v           Show version
```

## Environment Variables

Override config and flags:

```bash
PENTORA_CONFIG           # Config file path
PENTORA_WORKSPACE        # Workspace directory
PENTORA_LOG_LEVEL        # Logging level
PENTORA_SERVER           # Server URL for remote mode
PENTORA_API_TOKEN        # API authentication token
```

## Config File Format

YAML format:

```yaml
# ~/.config/pentora/config.yaml
workspace:
  dir: ~/.local/share/pentora
  enabled: true

scanner:
  default_profile: standard
  rate: 1000
  timeout: 3s

fingerprint:
  cache_dir: ${workspace}/cache/fingerprints
  catalog:
    remote_url: https://catalog.pentora.io/fingerprints.yaml

logging:
  level: info
  format: text

server:
  bind: 0.0.0.0:8080
  workers: 4
```

See [Configuration Overview](/configuration/overview) for complete schema.

## Shell Completion

Generate shell completion scripts:

### Bash

```bash
pentora completion bash > /etc/bash_completion.d/pentora
source /etc/bash_completion.d/pentora
```

### Zsh

```bash
pentora completion zsh > ~/.zsh/completion/_pentora
# Add to ~/.zshrc:
fpath=(~/.zsh/completion $fpath)
autoload -U compinit && compinit
```

### Fish

```bash
pentora completion fish > ~/.config/fish/completions/pentora.fish
```

### PowerShell

```powershell
pentora completion powershell | Out-String | Invoke-Expression
```

## Debugging

### Enable Debug Logging

```bash
pentora scan --targets 192.168.1.100 --log-level debug
```

### Log to File

```bash
pentora scan --targets 192.168.1.100 2> scan-debug.log
```

### Trace Execution

Show detailed execution flow:

```bash
pentora scan --targets 192.168.1.100 --log-level trace --log-format json | jq
```

### Dry Run

Validate configuration without executing:

```bash
pentora scan --targets 192.168.1.100 --dry-run
```

Shows what would be executed without actually running the scan.

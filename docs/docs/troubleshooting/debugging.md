# Debug Mode and Log Analysis

Advanced debugging techniques for troubleshooting Pentora.

## Enable Debug Logging

```bash
pentora scan --targets 192.168.1.100 --log-level debug
```

## Verbose Modes

```bash
# Level 1: Verbose
pentora scan --targets 192.168.1.100 -v

# Level 2: Very verbose (trace)
pentora scan --targets 192.168.1.100 -vv

# Level 3: Maximum verbosity
pentora scan --targets 192.168.1.100 -vvv
```

## Log to File

```bash
pentora scan --targets 192.168.1.100 --log-level debug 2> debug.log
```

## JSON Logging

Structured logs for parsing:
```bash
pentora scan --targets 192.168.1.100 --log-format json | jq
```

## Analyzing Logs

### Search for Errors
```bash
grep ERROR debug.log
```

### Filter by Component
```bash
jq 'select(.component == "orchestrator")' debug.json
```

### Timeline of Events
```bash
jq -r '.timestamp + " " + .message' debug.json | sort
```

## Dry Run

Validate without executing:
```bash
pentora scan --targets 192.168.1.100 --dry-run
```

## DAG Validation

```bash
pentora dag validate custom-scan.yaml
```

## Trace Execution

Follow execution flow:
```bash
pentora scan --targets 192.168.1.100 --trace
```

## Network Debugging

### Capture Packets
```bash
sudo tcpdump -i eth0 -w capture.pcap &
pentora scan --targets 192.168.1.100
sudo killall tcpdump
```

### Analyze with Wireshark
```bash
wireshark capture.pcap
```

## Module Debugging

Enable module-specific logging:
```yaml
logging:
  modules:
    discovery: debug
    scanner: trace
    fingerprint: debug
```

## Stack Traces

Enable panic stack traces:
```bash
export GOTRACEBACK=all
pentora scan --targets 192.168.1.100
```

## Profiling (Development)

CPU profiling:
```bash
pentora scan --targets 192.168.1.100 --cpuprofile cpu.prof
go tool pprof cpu.prof
```

Memory profiling:
```bash
pentora scan --targets 192.168.1.100 --memprofile mem.prof
go tool pprof mem.prof
```

See [Common Issues](/troubleshooting/common-issues) for specific problems.

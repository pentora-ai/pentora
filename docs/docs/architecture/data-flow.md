# Data Flow

## DataContext

Shared state container passed through DAG:

```go
type DataContext interface {
    Get(key string) (interface{}, error)
    Set(key string, value interface{}) error
    Has(key string) bool
    Keys() []string
}
```

## Standard Keys

| Key | Producer | Consumer | Type |
|-----|----------|----------|------|
| `targets` | Target Ingestion | Discovery | `[]Target` |
| `discovered_hosts` | Discovery | Port Scanner | `[]Host` |
| `open_ports` | Port Scanner | Banner Grab | `[]Port` |
| `banners` | Banner Grab | Fingerprint | `[]Banner` |
| `service_fingerprints` | Fingerprint | Asset Profiler | `[]Fingerprint` |
| `asset_profiles` | Asset Profiler | Vuln Evaluator | `[]AssetProfile` |
| `vulnerabilities` | Vuln Evaluator | Reporter | `[]Vulnerability` |

## Example Flow

```
Discovery → Set("discovered_hosts", hosts)
   ↓
Port Scanner → Get("discovered_hosts") → Scan → Set("open_ports", ports)
   ↓
Fingerprint → Get("open_ports") → Probe → Set("fingerprints", results)
```

See [DAG Engine](/docs/concepts/dag-engine) for execution orchestration.

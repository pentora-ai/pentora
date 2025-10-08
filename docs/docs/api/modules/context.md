# DataContext API

Interface for reading and writing shared scan data.

## DataContext Interface

```go
type DataContext interface {
    // Generic accessors
    Get(key string) (interface{}, error)
    Set(key string, value interface{}) error
    Has(key string) bool
    Keys() []string
    
    // Typed accessors
    GetTargets(key string) ([]Target, error)
    GetHosts(key string) ([]Host, error)
    GetPorts(key string) ([]Port, error)
    GetBanners(key string) ([]Banner, error)
    GetFingerprints(key string) ([]Fingerprint, error)
}
```

## Standard Keys

| Key | Type | Description |
|-----|------|-------------|
| `targets` | `[]Target` | Parsed targets |
| `discovered_hosts` | `[]Host` | Live hosts |
| `open_ports` | `[]Port` | Open ports |
| `banners` | `[]Banner` | Service banners |
| `service_fingerprints` | `[]Fingerprint` | Service IDs |
| `asset_profiles` | `[]AssetProfile` | Asset info |
| `vulnerabilities` | `[]Vulnerability` | CVEs |

## Usage Example

```go
func (m *Module) Execute(ctx context.Context, data DataContext) error {
    // Read
    hosts, err := data.GetHosts("discovered_hosts")
    if err != nil {
        return err
    }
    
    // Process
    var results []Result
    for _, host := range hosts {
        result := m.process(host)
        results = append(results, result)
    }
    
    // Write
    return data.Set("custom_results", results)
}
```

See [Module Interface](/api/modules/interface) for complete API.

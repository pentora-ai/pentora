# Custom Module Development

Create custom scan modules to extend Pentora's capabilities.

## Module Interface

```go
package mymodule

import (
    "context"
    "github.com/pentora/pentora/pkg/module"
)

type CustomModule struct {
    config Config
}

func (m *CustomModule) Name() string {
    return "custom_scanner"
}

func (m *CustomModule) Requires() []string {
    return []string{"discovered_hosts"}
}

func (m *CustomModule) Provides() []string {
    return []string{"custom_results"}
}

func (m *CustomModule) Execute(ctx context.Context, data module.DataContext) error {
    hosts, _ := data.GetHosts("discovered_hosts")
    
    var results []Result
    for _, host := range hosts {
        result := m.customScan(host)
        results = append(results, result)
    }
    
    return data.Set("custom_results", results)
}

func init() {
    module.Register("custom_scanner", &CustomModule{})
}
```

## Registration

Modules auto-register via `init()` or explicit registration:

```go
import _ "github.com/company/pentora-custom-modules"
```

## Usage in DAG

```yaml
nodes:
  - instance_id: custom
    module_type: custom_scanner
    depends_on: [discovery]
    config:
      timeout: 10s
```

See [Module API Reference](/docs/api/modules/interface) for complete interface.

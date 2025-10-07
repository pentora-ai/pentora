# Module API Interface

Reference for implementing custom modules.

## Module Interface

```go
type Module interface {
    // Metadata
    Name() string
    Description() string
    Version() string

    // Configuration
    ConfigSchema() Schema
    Configure(config Config) error

    // Dependencies
    Requires() []string
    Provides() []string

    // Execution
    Execute(ctx context.Context, data DataContext) error

    // Lifecycle
    Initialize() error
    Cleanup() error
}
```

## Methods

### Name()
Returns unique module identifier.

**Example**:
```go
func (m *CustomModule) Name() string {
    return "custom_scanner"
}
```

### Requires()
Declares required DataContext keys.

**Example**:
```go
func (m *CustomModule) Requires() []string {
    return []string{"discovered_hosts", "open_ports"}
}
```

### Provides()
Declares produced DataContext keys.

**Example**:
```go
func (m *CustomModule) Provides() []string {
    return []string{"scan_results"}
}
```

### Execute()
Main module logic.

**Example**:
```go
func (m *CustomModule) Execute(ctx context.Context, data DataContext) error {
    hosts, _ := data.GetHosts("discovered_hosts")
    
    results := m.scan(hosts)
    
    return data.Set("scan_results", results)
}
```

See [Custom Modules Guide](/docs/advanced/custom-modules) for examples.

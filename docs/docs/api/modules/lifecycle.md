# Module Lifecycle Hooks

Initialize resources and clean up gracefully.

## Lifecycle Phases

```
Registration → Initialize → Configure → Execute → Cleanup
```

## Initialize()

One-time setup called before any scans.

**Use cases**:
- Open persistent connections
- Load data files
- Allocate resources

**Example**:
```go
func (m *Module) Initialize() error {
    // Open database connection
    db, err := sql.Open("postgres", m.dsn)
    if err != nil {
        return err
    }
    m.db = db
    
    // Load signatures
    m.signatures, err = loadSignatures("signatures.yaml")
    return err
}
```

## Configure()

Apply scan-specific configuration.

**Example**:
```go
func (m *Module) Configure(config Config) error {
    m.timeout = config.GetDuration("timeout")
    m.concurrency = config.GetInt("concurrency")
    
    if m.concurrency < 1 {
        return errors.New("concurrency must be >= 1")
    }
    
    return nil
}
```

## Execute()

Perform module work (called per scan).

**Example**:
```go
func (m *Module) Execute(ctx context.Context, data DataContext) error {
    // Main module logic
    return nil
}
```

## Cleanup()

Release resources when module unloads.

**Example**:
```go
func (m *Module) Cleanup() error {
    if m.db != nil {
        return m.db.Close()
    }
    return nil
}
```

See [Custom Modules](/docs/advanced/custom-modules) for development guide.

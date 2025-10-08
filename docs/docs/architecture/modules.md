# Module Architecture

## Module Interface

All modules implement a common interface:

```go
type Module interface {
    Name() string
    Description() string
    Version() string
    ConfigSchema() Schema
    Configure(config Config) error
    Requires() []string
    Provides() []string
    Execute(ctx context.Context, data DataContext) error
    Initialize() error
    Cleanup() error
}
```

## Module Lifecycle

1. **Registration**: `init()` or dynamic loading
2. **Initialization**: One-time setup (connections, data)
3. **Configuration**: Apply runtime config
4. **Execution**: Perform work (can be called multiple times)
5. **Cleanup**: Release resources

## Module Types

- **Embedded**: Compiled Go code, in-process
- **External**: Separate process via gRPC
- **WASM**: WebAssembly sandboxed execution

See [Module System](/concepts/modules) for development guide.

# Plugin Architecture

Vulntor supports two plugin models: embedded and external.

## Embedded Plugins (Go)

Compiled into binary or loaded as `.so` shared objects.

**Advantages**: Fast, no IPC overhead
**Disadvantages**: Requires recompilation, same language

## External Plugins (gRPC)

Separate processes communicating via gRPC.

**Advantages**: 
- Any language (Python, Rust, etc.)
- Isolation (crashes don't affect Vulntor)
- Hot reload

**Disadvantages**: IPC overhead (~10-50ms per call)

## WASM Plugins (Experimental)

WebAssembly modules with sandboxed execution.

**Advantages**: Security, portability
**Disadvantages**: Limited ecosystem, performance overhead

## Enterprise Plugin Marketplace

Browse, install, and manage plugins via UI with licensing enforcement.

See [External Plugins Guide](/advanced/external-plugins) for development.

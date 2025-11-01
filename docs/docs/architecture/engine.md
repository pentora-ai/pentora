# Engine Internals

The Pentora engine consists of three primary components: Planner, Orchestrator, and Runtime.

## Components

### Planner
- Parses DAG definitions (YAML/JSON)
- Validates structure (cycles, dependencies)
- Resolves module instances
- Performs topological sort
- Groups nodes into execution layers

### Orchestrator
- Initializes DataContext
- Executes nodes in dependency order
- Manages parallelism
- Handles errors and retries
- Coordinates cleanup

### Runtime
- Module lifecycle management
- Resource pooling (connections, workers)
- Event emission
- Progress tracking

## Data Flow

```
Request → Planner → DAG → Orchestrator → Modules → DataContext → Results
```

### DataContext
Shared key-value store passed through pipeline:
- Type-safe accessors
- Thread-safe operations
- Size limits and eviction
- Persistence to storage

## Execution Model

1. **Planning Phase**: Build execution plan
2. **Initialization**: Setup modules and context
3. **Execution**: Run nodes layer by layer
4. **Cleanup**: Release resources

See [DAG Engine Concept](/concepts/dag-engine) for usage patterns.

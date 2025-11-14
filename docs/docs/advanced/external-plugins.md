# External Plugin Development

Develop plugins in any language using gRPC.

## gRPC Interface

```proto
service ModuleService {
    rpc Execute(ExecuteRequest) returns (ExecuteResponse);
    rpc GetMetadata(Empty) returns (Metadata);
}
```

## Python Example

```python
import grpc
from vulntor_pb2 import ExecuteRequest, ExecuteResponse

class CustomModule:
    def Execute(self, request, context):
        targets = request.context.get('targets')
        results = self.scan(targets)
        return ExecuteResponse(context={'results': results})

# Start gRPC server on localhost:50051
```

## Registration

```yaml
plugins:
  - name: custom_python_scanner
    type: grpc
    endpoint: localhost:50051
    timeout: 30s
```

## Usage

```bash
vulntor scan --targets 192.168.1.100 --plugin custom_python_scanner
```

See [Plugin Architecture](/architecture/plugins) for design details.

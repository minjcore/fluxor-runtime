# Queue RPC gRPC Server

A Fluxor binary that provides gRPC-based RPC service for cache lookup operations, using Protocol Buffers for efficient communication.

## Overview

This service exposes the same cache lookup functionality as the RabbitMQ-based `queue-rpc` but uses gRPC instead. It provides:

1. High-performance gRPC server for cache operations
2. Protocol Buffers for efficient serialization
3. Simple cache lookup interface via `GetCache` RPC method
4. Integration with Fluxor's cache system

## Building

```bash
go build -o queue-rpc-grpc ./cmd/queue-rpc-grpc
```

Or from the project root:

```bash
go build -o bin/queue-rpc-grpc ./cmd/queue-rpc-grpc
```

## Running

### Start the Server

```bash
./queue-rpc-grpc
```

Or with custom address:

```bash
GRPC_ADDRESS=localhost:50051 ./queue-rpc-grpc
```

The server will:
1. Start a gRPC server on the specified address (default: `localhost:50051`)
2. Pre-populate cache with test data
3. Listen for gRPC requests

### Environment Variables

- `GRPC_ADDRESS` - gRPC server address (default: `localhost:50051`)

## gRPC Service Definition

The service is defined in `proto/fluxor/queue/rpc.proto`:

```protobuf
service QueueRPC {
  rpc GetCache(GetCacheRequest) returns (GetCacheResponse);
}

message GetCacheRequest {
  string cache_key = 1;
  map<string, string> data = 2;
}

message GetCacheResponse {
  bool success = 1;
  bytes data = 2;
  string error = 3;
}
```

## Using the Client

### Go Client

Use the provided client example:

```bash
# Build the client
go build -o queue-rpc-grpc-client ./cmd/queue-rpc-grpc-client

# Run the client
./queue-rpc-grpc-client
```

Or use the client programmatically:

```go
import (
    "context"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    pb "github.com/fluxorio/fluxor/proto/fluxor/queue"
)

// Connect to server
conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

// Create client
client := pb.NewQueueRPCClient(conn)

// Make request
req := &pb.GetCacheRequest{
    CacheKey: "user:123",
}

response, err := client.GetCache(context.Background(), req)
if err != nil {
    log.Fatal(err)
}

if response.Success {
    fmt.Printf("Data: %s\n", string(response.Data))
} else {
    fmt.Printf("Error: %s\n", response.Error)
}
```

### Other Languages

Since this is a standard gRPC service, you can generate clients for any language supported by gRPC:

1. Install the gRPC plugin for your language
2. Generate client code from `proto/fluxor/queue/rpc.proto`
3. Use the generated client to connect to the server

## Example Output

Server output:
```
Starting Queue RPC gRPC runtime...
Deploying QueueRPCGRPCVerticle...
QueueRPCGRPCVerticle deployed successfully
Starting application...
QueueRPCGRPCVerticle Started
Pre-populated cache: user:123
Pre-populated cache: config:app
Pre-populated cache: data:test
gRPC Server started on localhost:50051
QueueRPCGRPCVerticle ready - listening to gRPC requests
Queue RPC gRPC runtime started successfully - listening on gRPC port
```

Client output:
```
Connecting to gRPC server at localhost:50051...
Making test RPC calls...

Making RPC call for cache key: user:123
✅ RPC Success for user:123: {"id": 123, "name": "John Doe", "email": "john@example.com"}

Making RPC call for cache key: config:app
✅ RPC Success for config:app: {"app_name": "fluxor", "version": "1.0.0"}

Making RPC call for cache key: data:test
✅ RPC Success for data:test: {"key": "test", "value": "test_data"}

Making RPC call for cache key: not:found
❌ RPC Error for not:found: cache error for key not:found: key not found

Test RPC calls completed
```

## Differences from RabbitMQ RPC

| Feature | RabbitMQ RPC | gRPC RPC |
|---------|--------------|----------|
| Protocol | AMQP (RabbitMQ) | gRPC (HTTP/2) |
| Serialization | JSON | Protocol Buffers |
| Transport | Message queues | Direct TCP connection |
| Performance | Good | Excellent (binary, HTTP/2) |
| Language Support | Language-specific AMQP clients | Any gRPC-supported language |
| Service Discovery | Queue names | Network addresses |
| Streaming | No | Yes (can be added) |

## Code Structure

- `QueueRPCGRPCVerticle`: A verticle that implements gRPC server
  - Uses `BaseVerticle` for lifecycle management
  - Creates `GRPCServer` to handle gRPC requests
  - Pre-populates cache with test data
  - Handles graceful shutdown

## Configuration

The server uses in-memory cache by default. The cache is pre-populated with test data:
- `user:123` - User data (JSON)
- `config:app` - Application config (JSON)
- `data:test` - Test data (JSON)

To use a different cache implementation, modify `cmd/queue-rpc-grpc/main.go` to use a different cache backend (e.g., Redis).

## Generating Proto Code

If you modify the proto files, regenerate the Go code:

```bash
make proto
```

This requires:
- `protoc` (Protocol Buffers compiler)
- `protoc-gen-go` (Go plugin)
- `protoc-gen-go-grpc` (gRPC Go plugin)

Install dependencies:
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## Testing

1. Start the server:
   ```bash
   ./queue-rpc-grpc
   ```

2. In another terminal, run the client:
   ```bash
   ./queue-rpc-grpc-client
   ```

3. Verify that cache lookups work correctly

## Troubleshooting

### Port Already in Use

If you get an error about the port being in use, change the address:
```bash
GRPC_ADDRESS=localhost:50052 ./queue-rpc-grpc
```

### Connection Refused

Make sure the server is running before starting the client.

### Proto Code Not Generated

Run `make proto` to generate the required Go code from proto definitions.


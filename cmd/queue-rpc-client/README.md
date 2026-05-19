# Queue RPC Client - Test Client for Queue RPC

A simple test client that demonstrates how to use RabbitMQ RPC client to call the RPC server.

## Overview

This example shows:
1. How to create a Queue RPC Client Verticle
2. How to send RPC requests to `queue_ha_core`
3. How to receive responses from `queue_reply`
4. How to handle RPC errors

## Building

```bash
go build -o queue-rpc-client.exe ./cmd/queue-rpc-client
```

Or from the project root:

```bash
go build -o bin/queue-rpc-client ./cmd/queue-rpc-client
```

## Running

### Prerequisites

1. Start RabbitMQ (if not already running)
2. Start the RPC Server first:
```bash
./queue-rpc  # or ./cmd/queue-rpc/queue-rpc
```

### Run the client

```bash
./queue-rpc-client.exe
```

## How it works

1. **Client starts**: Creates Queue Component and RPC Client
2. **Makes test calls**: Sends RPC requests for different cache keys:
   - `user:123` - Should succeed
   - `config:app` - Should succeed
   - `data:test` - Should succeed
   - `not:found` - Should fail (not in cache)
3. **Receives responses**: Processes success/error responses
4. **Logs results**: Shows success or error for each call

## Example Output

```
Starting Queue RPC Client...
Deploying QueueRPCClientVerticle...
QueueRPCClientVerticle deployed successfully
Starting application...
QueueRPCClientVerticle Started
RPC Client started, reply queue: queue_reply
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

## Testing Flow

1. **Terminal 1** - Start RPC Server:
```bash
./queue-rpc
```

2. **Terminal 2** - Start RPC Client:
```bash
./queue-rpc-client
```

The client will automatically make test calls and show results.

## Code Structure

- `QueueRPCClientVerticle`: A verticle that implements RPC client
  - Uses `BaseVerticle` for lifecycle management
  - Creates `QueueComponent` for RabbitMQ connection
  - Creates `RPCClient` to send RPC requests
  - Makes test calls to different cache keys
  - Handles graceful shutdown


# Queue RPC - Simple Queue RPC Example

A minimal Fluxor binary that demonstrates how to use RabbitMQ RPC pattern with queue `queue_ha_core` and cache.

## Overview

This example shows:
1. How to create a Queue RPC Server Verticle
2. How to consume RPC requests from `queue_ha_core`
3. How to retrieve data from cache using cache key
4. How to reply to RPC requests

## Building

```bash
go build -o queue-rpc.exe ./cmd/queue-rpc
```

Or from the project root:

```bash
go build -o bin/queue-rpc ./cmd/queue-rpc
```

## Running

### Prerequisites

1. Start RabbitMQ:
```bash
docker run -d --name rabbitmq \
  -p 5672:5672 \
  -p 15672:15672 \
  rabbitmq:3-management
```

2. Create queues (optional, will be auto-created):
- `queue_ha_core` - Request queue
- `queue_reply` - Reply queue

### Run the example

```bash
./queue-rpc.exe
```

Or with environment variables:

```bash
RABBITMQ_HOST=localhost \
RABBITMQ_USER=guest \
RABBITMQ_PASS=guest \
./queue-rpc.exe
```

Or using `.env.local` file (recommended):

Create a `.env.local` file in the **project root directory** (where `go.mod` is located):

```bash
# .env.local (in project root)
RABBITMQ_HOST=localhost
RABBITMQ_USER=guest
RABBITMQ_PASS=guest
```

Then run:

```bash
./queue-rpc.exe
```

The application will automatically find the project root (by locating `go.mod`) and load variables from `.env.local` if it exists. Environment variables set in the shell take precedence over `.env.local` values.

## How it works

1. **Server starts**: Creates Queue Component and RPC Server
2. **Pre-populates cache**: Adds test data to cache
3. **Consumes requests**: Listens to `queue_ha_core` queue
4. **Processes requests**: 
   - Receives RPC request with `CacheKey`
   - Retrieves data from cache
   - Sends response to reply queue
5. **Handles errors**: Returns error response if cache key not found

## Example Output

```
Starting Queue RPC runtime...
Deploying QueueRPCServerVerticle...
QueueRPCServerVerticle deployed successfully
Starting application...
QueueRPCServerVerticle Started
Pre-populated cache: user:123
Pre-populated cache: config:app
Pre-populated cache: data:test
RPC Server started, consuming from queue: queue_ha_core
QueueRPCServerVerticle ready - listening to RPC requests
Queue RPC runtime started successfully - listening to RPC requests on queue_ha_core
```

## Testing with RPC Client

You can test the server using the RPC client from `examples/queue-rpc`:

```go
// In another process
cache := cache.NewMemoryCache()
conn, _ := queue.NewConnection(config)

client, err := queue.NewRPCClient(conn, cache, "queue_reply", 30*time.Second)

req := queue.RPCRequest{
    CacheKey: "user:123",
}

response, err := client.Call(ctx, "queue_ha_core", req)
if response.Success {
    fmt.Printf("Data: %s\n", string(response.Data))
}
```

## Code Structure

- `QueueRPCServerVerticle`: A verticle that implements RPC server
  - Uses `BaseVerticle` for lifecycle management
  - Creates `QueueComponent` for RabbitMQ connection
  - Creates `RPCServer` to handle RPC requests
  - Pre-populates cache with test data
  - Handles graceful shutdown

## Configuration

Configuration can be provided in multiple ways (in order of precedence):

1. **Environment variables** (highest priority)
2. **`.env.local` file** (loaded automatically if present)
3. **Default values** (lowest priority)

### Using `.env.local` file (Recommended)

Create a `.env.local` file in the **project root directory** (where `go.mod` is located):

**Option 1: Using Connection URL (Recommended for real servers)**

```bash
# .env.local (in project root)
# Full connection URL - easiest way to connect to real servers
RABBITMQ_URL=amqp://username:password@rabbitmq.example.com:5672/vhost
```

**Option 2: Using Individual Settings**

```bash
# .env.local (in project root)
RABBITMQ_HOST=rabbitmq.example.com
RABBITMQ_PORT=5672
RABBITMQ_USER=myuser
RABBITMQ_PASS=mypassword
RABBITMQ_VHOST=/myvhost

# Optional: Connection timeout (default: 30s)
RABBITMQ_TIMEOUT=60s

# Optional: Enable TLS
RABBITMQ_TLS=true
RABBITMQ_TLS_INSECURE=false  # Set to true to skip certificate verification (not recommended)
```

The application automatically finds the project root by looking for the `go.mod` file, then loads `.env.local` from that location.

The `.env.local` file supports:
- Comments (lines starting with `#`)
- Empty lines (ignored)
- Quoted values (single or double quotes are automatically removed)
- Variables already set in the environment will not be overridden

### Using environment variables

Set environment variables directly:

```bash
export RABBITMQ_HOST=localhost
export RABBITMQ_USER=guest
export RABBITMQ_PASS=guest
```

### Using config.json

Edit `config.json` to change RabbitMQ settings:

```json
{
  "rabbitmq": {
    "host": "localhost",
    "port": 5672,
    "username": "guest",
    "password": "guest",
    "vhost": "/"
  }
}
```

### Configuration Variables

**Connection URL (Recommended):**
- `RABBITMQ_URL` - Full RabbitMQ connection URL (e.g., `amqp://user:pass@host:port/vhost`)
  - If provided, other individual settings are ignored
  - Format: `amqp://username:password@host:port/vhost`

**Individual Settings (used if RABBITMQ_URL is not set):**
- `RABBITMQ_HOST` (default: localhost) - RabbitMQ server hostname
- `RABBITMQ_PORT` (default: 5672) - RabbitMQ server port
- `RABBITMQ_USER` (default: guest) - RabbitMQ username
- `RABBITMQ_PASS` (default: guest) - RabbitMQ password
- `RABBITMQ_VHOST` (default: /) - Virtual host

**Advanced Options:**
- `RABBITMQ_TIMEOUT` (default: 30s) - Connection timeout (e.g., `60s`, `5m`)
- `RABBITMQ_TLS` (default: false) - Enable TLS/SSL connection (`true` or `1` to enable)
- `RABBITMQ_TLS_INSECURE` (default: false) - Skip certificate verification (`true` to enable, not recommended for production)

### Connecting to Real RabbitMQ Servers

**Example 1: Cloud RabbitMQ (CloudAMQP, AWS MQ, etc.)**

```bash
# .env.local
RABBITMQ_URL=amqp://user:password@rabbitmq-12345.cloudamqp.com:5672/my-vhost
```

**Example 2: Self-hosted with TLS**

```bash
# .env.local
RABBITMQ_HOST=rabbitmq.company.com
RABBITMQ_PORT=5671
RABBITMQ_USER=production_user
RABBITMQ_PASS=secure_password
RABBITMQ_VHOST=/production
RABBITMQ_TLS=true
RABBITMQ_TIMEOUT=60s
```

**Example 3: Local Development**

```bash
# .env.local
RABBITMQ_HOST=localhost
RABBITMQ_PORT=5672
RABBITMQ_USER=guest
RABBITMQ_PASS=guest
```


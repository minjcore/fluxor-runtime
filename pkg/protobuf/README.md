# Protobuf Package

This package provides Protocol Buffers (protobuf) serialization support for Fluxor, offering an alternative to JSON for high-performance message encoding/decoding.

## Features

- **Fail-fast validation**: Input validation before encoding/decoding
- **Type-safe**: Uses `proto.Message` interface for compile-time safety
- **Performance**: Binary format is more efficient than JSON for network communication
- **Codec interface**: Can be used as an alternative codec in EventBus

## Installation

### Prerequisites

Install the Protocol Buffers compiler:

```bash
# macOS
brew install protobuf

# Linux (Ubuntu/Debian)
apt-get install protobuf-compiler

# Or download from: https://grpc.io/docs/protoc-installation/
```

### Generate Go Code

Generate Go code from `.proto` files:

```bash
make proto
```

Or manually:

```bash
protoc --go_out=. --go_opt=paths=source_relative proto/fluxor/common/*.proto
```

## Usage

### Basic Encoding/Decoding

```go
import (
    "github.com/fluxorio/fluxor/pkg/protobuf"
    "github.com/fluxorio/fluxor/proto/fluxor/common"
)

// Encode a protobuf message
user := &common.User{
    Id:    "123",
    Name:  "John Doe",
    Email: "john@example.com",
}

data, err := protobuf.ProtobufEncode(user)
if err != nil {
    log.Fatal(err)
}

// Decode a protobuf message
var decodedUser common.User
err = protobuf.ProtobufDecode(data, &decodedUser)
if err != nil {
    log.Fatal(err)
}
```

### Using with EventBus

The EventBus currently uses JSON by default. To use protobuf, you can manually encode/decode:

```go
// Sending a protobuf message
user := &common.User{Id: "123", Name: "John"}
data, _ := protobuf.ProtobufEncode(user)
eventBus.Send("user.created", data)

// Receiving a protobuf message
eventBus.Consumer("user.created", func(msg core.Message) {
    var user common.User
    data := msg.Body().([]byte)
    protobuf.ProtobufDecode(data, &user)
    // Process user...
})
```

### Using with TCP Server

```go
tcpServer.SetHandler(func(ctx *tcp.ConnContext) error {
    // Read protobuf message
    data := make([]byte, 4096)
    n, err := ctx.Conn.Read(data)
    if err != nil {
        return err
    }
    
    var request common.Request
    if err := protobuf.ProtobufDecode(data[:n], &request); err != nil {
        return err
    }
    
    // Process request and send protobuf response
    response := &common.Response{
        RequestId: request.RequestId,
        Success:   true,
        Payload:   []byte("response data"),
    }
    
    responseData, _ := protobuf.ProtobufEncode(response)
    _, err = ctx.Conn.Write(responseData)
    return err
})
```

## Proto Files

Proto definitions are located in `proto/fluxor/common/`:

- `message.proto`: Generic message types for EventBus
- `user.proto`: User entity and operations
- `payment.proto`: Payment request/response types

## Codec Interface

The package provides a `Codec` interface that can be used for pluggable serialization:

```go
codec := &protobuf.Codec{}
data, err := codec.Encode(message)
err = codec.Decode(data, &decoded)
```

## Performance Considerations

Protobuf offers several advantages over JSON:

- **Smaller payload size**: Binary format is typically 20-30% smaller
- **Faster encoding/decoding**: No text parsing required
- **Type safety**: Schema validation at compile time
- **Backward compatibility**: Can evolve schemas while maintaining compatibility

However, JSON remains more human-readable and easier to debug. Choose protobuf when:
- High throughput is required
- Network bandwidth is a concern
- Inter-service communication needs optimization
- You need schema evolution support

## Error Handling

All functions follow Fluxor's fail-fast principles:

- `ProtobufEncode`: Returns error if value is nil or doesn't implement `proto.Message`
- `ProtobufDecode`: Returns error if data is empty, target is nil, or target doesn't implement `proto.Message`

Errors are wrapped with context for easier debugging.


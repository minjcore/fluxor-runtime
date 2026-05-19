# Protobuf Implementation

This document describes the protobuf implementation in Fluxor.

## Overview

Protobuf support has been added to Fluxor as an alternative to JSON for high-performance binary message serialization. The implementation follows Fluxor's fail-fast principles and provides a clean API similar to the existing JSON encoding/decoding utilities.

## Components

### 1. Core Package (`pkg/protobuf`)

- **ProtobufEncode**: Encodes proto.Message to bytes
- **ProtobufDecode**: Decodes bytes to proto.Message
- **Codec**: Interface for pluggable serialization

### 2. Proto Definitions (`proto/fluxor/common/`)

- `message.proto`: Generic message types for EventBus
- `user.proto`: User entity and operations
- `payment.proto`: Payment request/response types

### 3. Code Generation

Use `make proto` to generate Go code from `.proto` files. This requires:
- `protoc` compiler installed
- `google.golang.org/protobuf` Go package

### 4. Example (`examples/protobuf-tcp/`)

Demonstrates protobuf usage with TCP server, including:
- Length-prefixed message protocol
- Encoding/decoding protobuf messages
- Error handling

## Usage Patterns

### Basic Encoding/Decoding

```go
import "github.com/fluxorio/fluxor/pkg/protobuf"

// Encode
data, err := protobuf.ProtobufEncode(message)

// Decode
err = protobuf.ProtobufDecode(data, &decodedMessage)
```

### With EventBus

```go
// Send protobuf message
data, _ := protobuf.ProtobufEncode(user)
eventBus.Send("user.created", data)

// Receive protobuf message
eventBus.Consumer("user.created", func(msg core.Message) {
    var user common.User
    data := msg.Body().([]byte)
    protobuf.ProtobufDecode(data, &user)
})
```

### With TCP Server

See `examples/protobuf-tcp/main.go` for a complete example.

## Design Decisions

1. **Fail-fast validation**: All functions validate inputs before processing
2. **Type safety**: Requires proto.Message interface for compile-time safety
3. **Error handling**: Returns EventBusError for consistency with other packages
4. **Codec interface**: Provides pluggable serialization for future extensibility

## Future Enhancements

1. **EventBus integration**: Add optional protobuf codec to EventBus
2. **gRPC support**: Add gRPC server/client support
3. **Streaming**: Support for streaming protobuf messages
4. **Schema registry**: Optional schema registry for versioning

## Testing

Tests focus on fail-fast validation since full integration tests require generated proto code. Run `make proto` first to generate code, then add integration tests using generated types.

## Dependencies

- `google.golang.org/protobuf`: Core protobuf library
- `protoc`: Protocol Buffers compiler (external tool)


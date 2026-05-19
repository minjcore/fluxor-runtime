package protobuf

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"
)

var (
	// ErrInvalidInput is returned when input validation fails
	ErrInvalidInput = errors.New("invalid input")
	// ErrNilValue is returned when trying to encode/decode nil value
	ErrNilValue = errors.New("cannot encode/decode nil value")
	// ErrEmptyData is returned when trying to decode empty data
	ErrEmptyData = errors.New("cannot decode empty data")
	// ErrNotProtoMessage is returned when value doesn't implement proto.Message
	ErrNotProtoMessage = errors.New("value must implement proto.Message")
)

// ProtobufEncode encodes a protobuf message to bytes (fail-fast).
// The value must implement proto.Message interface.
func ProtobufEncode(v interface{}) ([]byte, error) {
	// Fail-fast: validate input
	if v == nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, ErrNilValue)
	}

	// Fail-fast: must be a proto.Message
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("%w: %v, got %T", ErrInvalidInput, ErrNotProtoMessage, v)
	}

	// Encode using protobuf
	data, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("protobuf encode failed: %w", err)
	}

	return data, nil
}

// ProtobufDecode decodes protobuf bytes to a message (fail-fast).
// The target must be a pointer to a struct that implements proto.Message.
func ProtobufDecode(data []byte, v interface{}) error {
	// Fail-fast: validate inputs
	if len(data) == 0 {
		return fmt.Errorf("%w: %v", ErrInvalidInput, ErrEmptyData)
	}
	if v == nil {
		return fmt.Errorf("%w: %v", ErrInvalidInput, ErrNilValue)
	}

	// Fail-fast: must be a pointer to proto.Message
	msg, ok := v.(proto.Message)
	if !ok {
		return fmt.Errorf("%w: %v, got %T", ErrInvalidInput, ErrNotProtoMessage, v)
	}

	// Decode using protobuf
	if err := proto.Unmarshal(data, msg); err != nil {
		return fmt.Errorf("protobuf decode failed: %w", err)
	}

	return nil
}

// Codec provides a codec interface for protobuf serialization.
// This allows protobuf to be used as an alternative to JSON in EventBus.
type Codec struct{}

// Encode encodes a value to bytes.
func (c *Codec) Encode(v interface{}) ([]byte, error) {
	return ProtobufEncode(v)
}

// Decode decodes bytes to a value.
func (c *Codec) Decode(data []byte, v interface{}) error {
	return ProtobufDecode(data, v)
}

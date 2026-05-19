package core

import (
	"encoding/json"
	"fmt"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// JSON is a convenience alias for JSON objects (Dev UX).
type JSON map[string]any

// JSONEncode encodes a value to JSON bytes (fail-fast).
// Panics on nil input; returns error only for encoding failures (e.g. unsupported type).
// Uses standard encoding/json for JSON encoding.
// Note: Previously used Sonic for better performance, but switched to stdlib
// for Go 1.24 compatibility. Will switch back when Sonic supports Go 1.24.
func JSONEncode(v interface{}) ([]byte, error) {
	failfast.NotNil(v, "value to encode")

	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("json encode failed: %w", err)
	}
	return data, nil
}

// JSONDecode decodes JSON bytes into v (fail-fast).
// Panics on empty data or nil target; returns error only for decode failures (e.g. malformed JSON).
// Uses standard encoding/json for JSON decoding.
// Note: Previously used Sonic for better performance, but switched to stdlib
// for Go 1.24 compatibility. Will switch back when Sonic supports Go 1.24.
func JSONDecode(data []byte, v interface{}) error {
	failfast.If(len(data) > 0, "data cannot be empty")
	failfast.NotNil(v, "decode target")

	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("json decode failed: %w", err)
	}
	return nil
}

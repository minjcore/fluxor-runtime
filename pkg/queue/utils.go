package queue

import (
	"encoding/json"
	"fmt"
)

// JSONEncode encodes v to JSON bytes
func JSONEncode(v interface{}) ([]byte, error) {
	if data, ok := v.([]byte); ok {
		return data, nil
	}
	return json.Marshal(v)
}

// JSONDecode decodes JSON bytes into v
func JSONDecode(data []byte, v interface{}) error {
	if v == nil {
		return &Error{Code: "INVALID_INPUT", Message: "decode target cannot be nil"}
	}
	return json.Unmarshal(data, v)
}

// ValidateContext validates context (fail-fast)
func ValidateContext(ctx interface{}) {
	if ctx == nil {
		panic("context cannot be nil")
	}
}

// ValidateQueueName validates queue name (fail-fast)
func ValidateQueueName(name string) error {
	if name == "" {
		return &Error{Code: "INVALID_INPUT", Message: "queue name cannot be empty"}
	}
	if len(name) > 255 {
		return &Error{Code: "INVALID_INPUT", Message: fmt.Sprintf("queue name too long: %d > 255", len(name))}
	}
	return nil
}

// ValidateExchangeName validates exchange name (fail-fast)
func ValidateExchangeName(name string) error {
	if name == "" {
		return &Error{Code: "INVALID_INPUT", Message: "exchange name cannot be empty"}
	}
	if len(name) > 255 {
		return &Error{Code: "INVALID_INPUT", Message: fmt.Sprintf("exchange name too long: %d > 255", len(name))}
	}
	return nil
}

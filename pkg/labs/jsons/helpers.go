package jsons

import (
	"encoding/json"
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/tidwall/gjson"
)

// sonicAvailable indicates if sonic library is available
// Note: sonic may have issues with Go 1.24+, so we use build tags if needed
var sonicAvailable = false

// encodeWithStdlib encodes data using standard library encoding/json
func encodeWithStdlib(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

// decodeWithStdlib decodes data using standard library encoding/json
func decodeWithStdlib(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// encodeWithJsoniter encodes data using jsoniter library
func encodeWithJsoniter(data interface{}) ([]byte, error) {
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	return json.Marshal(data)
}

// decodeWithJsoniter decodes data using jsoniter library
func decodeWithJsoniter(data []byte, v interface{}) error {
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	return json.Unmarshal(data, v)
}

// encodeWithSonic encodes data using sonic library (if available)
// Note: This is a placeholder. In production, use build tags for Go version compatibility
func encodeWithSonic(data interface{}) ([]byte, error) {
	if !sonicAvailable {
		return nil, fmt.Errorf("sonic not available (check Go version compatibility)")
	}
	// Placeholder - would use sonic.Marshal if available
	// For now, fallback to stdlib
	return json.Marshal(data)
}

// decodeWithSonic decodes data using sonic library (if available)
// Note: This is a placeholder. In production, use build tags for Go version compatibility
func decodeWithSonic(data []byte, v interface{}) error {
	if !sonicAvailable {
		return fmt.Errorf("sonic not available (check Go version compatibility)")
	}
	// Placeholder - would use sonic.Unmarshal if available
	// For now, fallback to stdlib
	return json.Unmarshal(data, v)
}

// encodeWithEasyjson encodes data using easyjson (requires code generation)
// Note: This requires code generation step - see easyjson documentation
func encodeWithEasyjson(data interface{}) ([]byte, error) {
	// Placeholder - requires code generation
	// For types that have generated MarshalJSON methods, use them here
	// For now, fallback to stdlib
	return json.Marshal(data)
}

// decodeWithEasyjson decodes data using easyjson (requires code generation)
// Note: This requires code generation step - see easyjson documentation
func decodeWithEasyjson(data []byte, v interface{}) error {
	// Placeholder - requires code generation
	// For types that have generated UnmarshalJSON methods, use them here
	// For now, fallback to stdlib
	return json.Unmarshal(data, v)
}

// parseWithGjson parses JSON string and extracts value by path (read-only)
// gjson is optimized for path-based queries without full unmarshaling
func parseWithGjson(jsonStr string, path string) gjson.Result {
	return gjson.Get(jsonStr, path)
}

// parseMultipleWithGjson parses multiple paths from JSON string
func parseMultipleWithGjson(jsonStr string, paths ...string) map[string]gjson.Result {
	results := make(map[string]gjson.Result)
	for _, path := range paths {
		results[path] = gjson.Get(jsonStr, path)
	}
	return results
}

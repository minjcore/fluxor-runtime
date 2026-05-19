package eventloop

import (
	"hash/fnv"
)

// KeyExtractor extracts a routing key from event headers
// Note: FloxID from context is checked BEFORE this extractor is called
// This extractor only handles headers (X-Route-Key > X-Flox-ID > X-User-ID > ...)
type KeyExtractor func(headers map[string]string, address string, body interface{}) string

// DefaultKeyExtractor extracts key from headers with fallback chain
// Note: FloxID from context has higher priority and is checked in Dispatcher.Dispatch()
// This extractor handles headers only: X-Route-Key > X-Flox-ID > X-User-ID > X-Session-ID > X-Request-ID
func DefaultKeyExtractor(headers map[string]string, address string, body interface{}) string {
	if headers == nil {
		return ""
	}

	// Priority order: X-Route-Key > X-Flox-ID > X-User-ID > X-Session-ID > X-Request-ID
	// FloxID is a universal routing identifier (aggregate ID, stream ID, entity ID, etc.)
	keyOrder := []string{
		"X-Route-Key",  // Explicit routing key (highest priority)
		"X-Flox-ID",    // Universal routing ID (aggregate ID, stream ID, entity ID, etc.)
		"X-User-ID",    // User-based routing
		"X-Session-ID", // Session-based routing
		"X-Request-ID", // Request-based routing (lowest priority)
	}

	for _, key := range keyOrder {
		if val, ok := headers[key]; ok && val != "" {
			return val
		}
	}

	return ""
}

// HashKey hashes a key string to an integer
// Uses FNV-1a hash for fast, good distribution
func HashKey(key string) uint32 {
	if key == "" {
		return 0
	}
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

// RouteKey routes a key to a loop index using consistent hashing
func RouteKey(key string, numLoops int) int {
	if numLoops <= 0 {
		return 0
	}
	if key == "" {
		return 0 // Fallback to first loop if no key
	}
	hash := HashKey(key)
	// Safe conversion: Perform modulo on uint32, then convert to int
	// The result of hash % uint32(numLoops) is guaranteed to be in [0, numLoops-1]
	// which will always fit in int since numLoops is already an int
	modResult := hash % uint32(numLoops)
	// Convert to int safely - modResult is always < numLoops, so it fits in int
	// On 32-bit systems, clamp to int32 max to prevent overflow
	const maxInt32 = 1<<31 - 1
	if modResult > uint32(maxInt32) {
		// Clamp to maxInt32 to prevent overflow on 32-bit systems
		// This is safe because modResult < numLoops, and if numLoops > maxInt32,
		// we'd have other issues anyway
		return maxInt32 % numLoops
	}
	return int(modResult)
}

// ExtractRouteKey extracts routing key from event using extractor
func ExtractRouteKey(event *Event, extractor KeyExtractor) string {
	if extractor == nil {
		extractor = DefaultKeyExtractor
	}
	return extractor(event.Headers, event.Address, event.Body)
}

// ValidateKey validates that a key is not empty (for fail-fast)
func ValidateKey(key string) error {
	if key == "" {
		return &EventLoopError{
			Code:    "INVALID_KEY",
			Message: "routing key cannot be empty (use fallback extractor if key is optional)",
		}
	}
	return nil
}

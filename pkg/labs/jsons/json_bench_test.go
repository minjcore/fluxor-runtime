package jsons

import (
	"encoding/json"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/tidwall/gjson"
)

// ============================================================================
// ENCODING (MARSHAL) BENCHMARKS
// ============================================================================

// BenchmarkEncode_Small benchmarks encoding small objects (< 1KB)
func BenchmarkEncode_Small_Stdlib(b *testing.B) {
	user := generateSmallUser()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(user)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncode_Small_Jsoniter(b *testing.B) {
	user := generateSmallUser()
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(user)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEncode_Medium benchmarks encoding medium objects (1KB-100KB)
func BenchmarkEncode_Medium_Stdlib(b *testing.B) {
	users := generateMediumUsers(100)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(users)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncode_Medium_Jsoniter(b *testing.B) {
	users := generateMediumUsers(100)
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(users)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEncode_Large benchmarks encoding large objects (100KB-10MB)
func BenchmarkEncode_Large_Stdlib(b *testing.B) {
	nested := generateLargeNestedData(1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(nested)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncode_Large_Jsoniter(b *testing.B) {
	nested := generateLargeNestedData(1000)
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(nested)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEncode_Structured benchmarks encoding complex nested structures
func BenchmarkEncode_Structured_Stdlib(b *testing.B) {
	config := generateConfigData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEncode_Structured_Jsoniter(b *testing.B) {
	config := generateConfigData()
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEncode_Parallel benchmarks concurrent encoding
func BenchmarkEncode_Parallel_Stdlib(b *testing.B) {
	user := generateSmallUser()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := json.Marshal(user)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkEncode_Parallel_Jsoniter(b *testing.B) {
	user := generateSmallUser()
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := json.Marshal(user)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// ============================================================================
// DECODING (UNMARSHAL) BENCHMARKS
// ============================================================================

// BenchmarkDecode_Small benchmarks decoding small JSON strings
func BenchmarkDecode_Small_Stdlib(b *testing.B) {
	user := generateSmallUser()
	data, _ := json.Marshal(user)
	var result User
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := json.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecode_Small_Jsoniter(b *testing.B) {
	user := generateSmallUser()
	data, _ := json.Marshal(user)
	var result User
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := json.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecode_Medium benchmarks decoding medium JSON strings
func BenchmarkDecode_Medium_Stdlib(b *testing.B) {
	users := generateMediumUsers(100)
	data, _ := json.Marshal(users)
	var result []User
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := json.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecode_Medium_Jsoniter(b *testing.B) {
	users := generateMediumUsers(100)
	data, _ := json.Marshal(users)
	var result []User
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := json.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecode_Large benchmarks decoding large JSON strings
func BenchmarkDecode_Large_Stdlib(b *testing.B) {
	nested := generateLargeNestedData(1000)
	data, _ := json.Marshal(nested)
	var result NestedData
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := json.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecode_Large_Jsoniter(b *testing.B) {
	nested := generateLargeNestedData(1000)
	data, _ := json.Marshal(nested)
	var result NestedData
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := json.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecode_Structured benchmarks decoding complex nested structures
func BenchmarkDecode_Structured_Stdlib(b *testing.B) {
	config := generateConfigData()
	data, _ := json.Marshal(config)
	var result ConfigData
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := json.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecode_Structured_Jsoniter(b *testing.B) {
	config := generateConfigData()
	data, _ := json.Marshal(config)
	var result ConfigData
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := json.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDecode_Parallel benchmarks concurrent decoding
func BenchmarkDecode_Parallel_Stdlib(b *testing.B) {
	user := generateSmallUser()
	data, _ := json.Marshal(user)
	var result User
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := json.Unmarshal(data, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkDecode_Parallel_Jsoniter(b *testing.B) {
	user := generateSmallUser()
	data, _ := json.Marshal(user)
	var result User
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := json.Unmarshal(data, &result)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// ============================================================================
// THROUGHPUT TESTS
// ============================================================================

// BenchmarkThroughput_Encode benchmarks encoding throughput
func BenchmarkThroughput_Encode_Stdlib(b *testing.B) {
	user := generateSmallUser()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(user)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkThroughput_Encode_Jsoniter(b *testing.B) {
	user := generateSmallUser()
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(user)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkThroughput_Decode benchmarks decoding throughput
func BenchmarkThroughput_Decode_Stdlib(b *testing.B) {
	user := generateSmallUser()
	data, _ := json.Marshal(user)
	var result User
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := json.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkThroughput_Decode_Jsoniter(b *testing.B) {
	user := generateSmallUser()
	data, _ := json.Marshal(user)
	var result User
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := json.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkThroughput_RoundTrip benchmarks full encode/decode cycle
func BenchmarkThroughput_RoundTrip_Stdlib(b *testing.B) {
	user := generateSmallUser()
	var result User
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(user)
		if err != nil {
			b.Fatal(err)
		}
		err = json.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkThroughput_RoundTrip_Jsoniter(b *testing.B) {
	user := generateSmallUser()
	var result User
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(user)
		if err != nil {
			b.Fatal(err)
		}
		err = json.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ============================================================================
// MEMORY ALLOCATION BENCHMARKS
// ============================================================================

// BenchmarkAlloc_Encode benchmarks allocation per encode operation
// Run with: go test -bench=BenchmarkAlloc_Encode -benchmem
func BenchmarkAlloc_Encode_Stdlib(b *testing.B) {
	user := generateSmallUser()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(user)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAlloc_Encode_Jsoniter(b *testing.B) {
	user := generateSmallUser()
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(user)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAlloc_Decode benchmarks allocation per decode operation
func BenchmarkAlloc_Decode_Stdlib(b *testing.B) {
	user := generateSmallUser()
	data, _ := json.Marshal(user)
	var result User
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err := json.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAlloc_Decode_Jsoniter(b *testing.B) {
	user := generateSmallUser()
	data, _ := json.Marshal(user)
	var result User
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		err := json.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ============================================================================
// USE CASE SPECIFIC BENCHMARKS
// ============================================================================

// BenchmarkEventBus_JSON benchmarks EventBus-like usage (small messages)
func BenchmarkEventBus_JSON_Stdlib(b *testing.B) {
	msg := generateEventBusMessage()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkEventBus_JSON_Jsoniter(b *testing.B) {
	msg := generateEventBusMessage()
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAPI_JSON benchmarks REST API-like usage (medium messages)
func BenchmarkAPI_JSON_Stdlib(b *testing.B) {
	msg := generateAPIMessage()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAPI_JSON_Jsoniter(b *testing.B) {
	msg := generateAPIMessage()
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkConfig_JSON benchmarks config loading (structured data)
func BenchmarkConfig_JSON_Stdlib(b *testing.B) {
	config := generateConfigData()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConfig_JSON_Jsoniter(b *testing.B) {
	config := generateConfigData()
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(config)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHighThroughput_SmallPayloads benchmarks many small messages
func BenchmarkHighThroughput_SmallPayloads_Stdlib(b *testing.B) {
	user := generateSmallUser()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := json.Marshal(user)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkHighThroughput_SmallPayloads_Jsoniter(b *testing.B) {
	user := generateSmallUser()
	json := jsoniter.ConfigCompatibleWithStandardLibrary
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := json.Marshal(user)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// ============================================================================
// GJSON BENCHMARKS (Read-only parsing)
// ============================================================================

// BenchmarkGjson_PartialParse benchmarks path-based queries
func BenchmarkGjson_PartialParse(b *testing.B) {
	nested := generateLargeNestedData(1000)
	data, _ := json.Marshal(nested)
	jsonStr := string(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Parse only specific path
		_ = gjson.Get(jsonStr, "users.0.name")
		_ = gjson.Get(jsonStr, "meta.page")
		_ = gjson.Get(jsonStr, "count")
	}
}

// BenchmarkGjson_LargeFiles benchmarks large file parsing
func BenchmarkGjson_LargeFiles(b *testing.B) {
	nested := generateLargeNestedData(5000)
	data, _ := json.Marshal(nested)
	jsonStr := string(data)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Parse multiple paths without full unmarshaling
		result := gjson.Get(jsonStr, "users.#")
		_ = result.Int()
		result = gjson.Get(jsonStr, "users.100.email")
		_ = result.String()
	}
}

// TestGjson_PathQueries tests path query functionality
func TestGjson_PathQueries(t *testing.T) {
	user := generateSmallUser()
	data, _ := json.Marshal(user)
	jsonStr := string(data)

	name := gjson.Get(jsonStr, "name")
	if name.String() != user.Name {
		t.Errorf("expected %s, got %s", user.Name, name.String())
	}

	email := gjson.Get(jsonStr, "email")
	if email.String() != user.Email {
		t.Errorf("expected %s, got %s", user.Email, email.String())
	}
}

// ============================================================================
// STANDARD LIBRARY TESTS (Compatibility, Error Handling, Edge Cases)
// ============================================================================

// TestStdlib_Compatibility tests RFC 7159 compliance
func TestStdlib_Compatibility(t *testing.T) {
	// Test valid JSON
	validJSON := `{"name":"test","value":42}`
	var result map[string]interface{}
	err := json.Unmarshal([]byte(validJSON), &result)
	if err != nil {
		t.Fatalf("valid JSON should parse: %v", err)
	}

	// Test numbers
	numberJSON := `{"int":42,"float":3.14}`
	err = json.Unmarshal([]byte(numberJSON), &result)
	if err != nil {
		t.Fatalf("numbers should parse: %v", err)
	}

	// Test arrays
	arrayJSON := `{"items":[1,2,3]}`
	err = json.Unmarshal([]byte(arrayJSON), &result)
	if err != nil {
		t.Fatalf("arrays should parse: %v", err)
	}
}

// TestStdlib_ErrorHandling tests robust error handling
func TestStdlib_ErrorHandling(t *testing.T) {
	// Test invalid JSON
	invalidJSON := `{"name":"test",invalid}`
	var result map[string]interface{}
	err := json.Unmarshal([]byte(invalidJSON), &result)
	if err == nil {
		t.Fatal("invalid JSON should return error")
	}

	// Test incomplete JSON
	incompleteJSON := `{"name":"test"`
	err = json.Unmarshal([]byte(incompleteJSON), &result)
	if err == nil {
		t.Fatal("incomplete JSON should return error")
	}
}

// ============================================================================
// JSONITER TESTS (Drop-in Replacement, Advanced Features)
// ============================================================================

// TestJsoniter_DropInReplacement tests API compatibility with stdlib
func TestJsoniter_DropInReplacement(t *testing.T) {
	user := generateSmallUser()
	json := jsoniter.ConfigCompatibleWithStandardLibrary

	// Marshal should work the same way
	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("jsoniter marshal failed: %v", err)
	}

	// Unmarshal should work the same way
	var result User
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("jsoniter unmarshal failed: %v", err)
	}

	if result.ID != user.ID {
		t.Errorf("expected ID %s, got %s", user.ID, result.ID)
	}
}

// BenchmarkJsoniter_VsStdlib benchmarks jsoniter vs stdlib comparison
func BenchmarkJsoniter_VsStdlib(b *testing.B) {
	user := generateSmallUser()

	b.Run("Stdlib", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(user)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Jsoniter", func(b *testing.B) {
		json := jsoniter.ConfigCompatibleWithStandardLibrary
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(user)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

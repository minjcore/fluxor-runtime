package core

import (
	"encoding/json"
	"testing"
)

// BenchmarkJSONEncode benchmarks the JSON encoding wrapper
func BenchmarkJSONEncode(b *testing.B) {
	data := map[string]interface{}{
		"name":  "test",
		"value": 42,
		"nested": map[string]string{
			"key": "value",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := JSONEncode(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkJSONEncode_Standard benchmarks standard json.Marshal for comparison
func BenchmarkJSONEncode_Standard(b *testing.B) {
	data := map[string]interface{}{
		"name":  "test",
		"value": 42,
		"nested": map[string]string{
			"key": "value",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkJSONEncode_SonicDirect benchmarks Sonic Marshal directly for comparison
// Disabled due to Sonic incompatibility with Go 1.24
/*
func BenchmarkJSONEncode_SonicDirect(b *testing.B) {
	data := map[string]interface{}{
		"name":  "test",
		"value": 42,
		"nested": map[string]string{
			"key": "value",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := sonic.Marshal(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}
*/

// BenchmarkJSONDecode benchmarks the JSON decoding wrapper
func BenchmarkJSONDecode(b *testing.B) {
	data := []byte(`{"name":"test","value":42,"nested":{"key":"value"}}`)
	var result map[string]interface{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := JSONDecode(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkJSONDecode_Standard benchmarks standard json.Unmarshal for comparison
func BenchmarkJSONDecode_Standard(b *testing.B) {
	data := []byte(`{"name":"test","value":42,"nested":{"key":"value"}}`)
	var result map[string]interface{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := json.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkJSONDecode_SonicDirect benchmarks Sonic Unmarshal directly for comparison
// Disabled due to Sonic incompatibility with Go 1.24
/*
func BenchmarkJSONDecode_SonicDirect(b *testing.B) {
	data := []byte(`{"name":"test","value":42,"nested":{"key":"value"}}`)
	var result map[string]interface{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := sonic.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}
*/

// BenchmarkJSONEncode_Parallel benchmarks concurrent encoding
func BenchmarkJSONEncode_Parallel(b *testing.B) {
	data := map[string]interface{}{
		"name":  "test",
		"value": 42,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := JSONEncode(data)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

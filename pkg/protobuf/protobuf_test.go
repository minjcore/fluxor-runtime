package protobuf

import (
	"testing"
)

// Note: Full integration tests require generated proto code.
// To generate code: make proto
// Then update tests to use generated types from proto/fluxor/common/

func TestProtobufEncode_FailFast(t *testing.T) {
	tests := []struct {
		name    string
		v       interface{}
		wantErr bool
	}{
		{
			name:    "nil value",
			v:       nil,
			wantErr: true,
		},
		{
			name:    "non-proto message",
			v:       map[string]string{"key": "value"},
			wantErr: true,
		},
		{
			name:    "string value",
			v:       "not a proto message",
			wantErr: true,
		},
		{
			name:    "int value",
			v:       42,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ProtobufEncode(tt.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProtobufEncode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
			}
		})
	}
}

func TestProtobufDecode_FailFast(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		v       interface{}
		wantErr bool
	}{
		{
			name:    "empty data",
			data:    []byte{},
			v:       &struct{}{},
			wantErr: true,
		},
		{
			name:    "nil target",
			data:    []byte{0x0a, 0x03, 0x31, 0x32, 0x33},
			v:       nil,
			wantErr: true,
		},
		{
			name:    "non-proto message target",
			data:    []byte{0x0a, 0x03, 0x31, 0x32, 0x33},
			v:       map[string]string{},
			wantErr: true,
		},
		{
			name:    "string target",
			data:    []byte{0x0a, 0x03, 0x31, 0x32, 0x33},
			v:       new(string),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ProtobufDecode(tt.data, tt.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("ProtobufDecode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
			}
		})
	}
}

func TestProtobufEncode_FailFast_NilValue(t *testing.T) {
	_, err := ProtobufEncode(nil)
	if err == nil {
		t.Error("ProtobufEncode() should fail-fast with nil value")
	}
	if err.Error() == "" {
		t.Error("ProtobufEncode() error should have a message")
	}
}

func TestProtobufDecode_FailFast_EmptyData(t *testing.T) {
	var result struct{}
	err := ProtobufDecode([]byte{}, &result)
	if err == nil {
		t.Error("ProtobufDecode() should fail-fast with empty data")
	}
}

func TestProtobufDecode_FailFast_NilTarget(t *testing.T) {
	data := []byte{0x0a, 0x03, 0x31, 0x32, 0x33} // Valid protobuf data
	err := ProtobufDecode(data, nil)
	if err == nil {
		t.Error("ProtobufDecode() should fail-fast with nil target")
	}
}

// TestCodec tests the Codec interface
func TestCodec(t *testing.T) {
	codec := &Codec{}

	// Test encoding non-proto message (should fail)
	_, err := codec.Encode(map[string]string{"key": "value"})
	if err == nil {
		t.Error("Codec.Encode() should fail with non-proto message")
	}

	// Test decoding with invalid target (should fail)
	err = codec.Decode([]byte{0x0a, 0x03, 0x31, 0x32, 0x33}, map[string]string{})
	if err == nil {
		t.Error("Codec.Decode() should fail with non-proto target")
	}
}

// Note: Full integration tests with actual proto messages require generated code.
// Example test with generated code:
//
// func TestProtobufEncodeDecode_Integration(t *testing.T) {
//     original := &common.User{
//         Id:   "123",
//         Name: "test user",
//     }
//
//     encoded, err := ProtobufEncode(original)
//     if err != nil {
//         t.Fatalf("ProtobufEncode() error = %v", err)
//     }
//
//     var decoded common.User
//     err = ProtobufDecode(encoded, &decoded)
//     if err != nil {
//         t.Fatalf("ProtobufDecode() error = %v", err)
//     }
//
//     if decoded.Id != original.Id {
//         t.Errorf("decoded.Id = %v, want %v", decoded.Id, original.Id)
//     }
// }

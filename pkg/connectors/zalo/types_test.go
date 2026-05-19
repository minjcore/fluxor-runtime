package zalo

import (
	"encoding/json"
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		want  string
	}{
		{"empty", "", ""},
		{"10 digits leading 0", "0898527311", "84898527311"},
		{"with plus", "+84898527311", "84898527311"},
		{"with spaces", "089 852 7311", "84898527311"},
		{"9 digits", "898527311", "84898527311"},
		{"already 84", "84898527311", "84898527311"},
		{"84 with more digits", "84898527311000", "84898527311"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizePhone(tt.phone)
			if got != tt.want {
				t.Errorf("NormalizePhone(%q) = %q, want %q", tt.phone, got, tt.want)
			}
		})
	}
}

func TestFlexInt_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want int
	}{
		{"number", "3600", 3600},
		{"string", `"3600"`, 3600},
		{"zero", "0", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fi flexInt
			err := json.Unmarshal([]byte(tt.raw), &fi)
			if err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if int(fi) != tt.want {
				t.Errorf("flexInt = %d, want %d", fi, tt.want)
			}
		})
	}
}

func TestTokenResponse_ExpiresIn_StringOrNumber(t *testing.T) {
	// Zalo API may return expires_in as string or number
	t.Run("expires_in as number", func(t *testing.T) {
		raw := `{"access_token":"at","refresh_token":"rt","expires_in":3600}`
		var tr TokenResponse
		if err := json.Unmarshal([]byte(raw), &tr); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if int(tr.ExpiresIn) != 3600 {
			t.Errorf("ExpiresIn = %d, want 3600", tr.ExpiresIn)
		}
	})
	t.Run("expires_in as string", func(t *testing.T) {
		raw := `{"access_token":"at","refresh_token":"rt","expires_in":"3600"}`
		var tr TokenResponse
		if err := json.Unmarshal([]byte(raw), &tr); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		if int(tr.ExpiresIn) != 3600 {
			t.Errorf("ExpiresIn = %d, want 3600", tr.ExpiresIn)
		}
	})
}

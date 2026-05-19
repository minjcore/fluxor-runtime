package authn

import (
	"testing"
	"time"
)

func TestPrincipal_IsExpired(t *testing.T) {
	tests := []struct {
		name     string
		principal *Principal
		want     bool
	}{
		{
			name: "not expired - no expiration",
			principal: &Principal{
				ExpiresAt: nil,
			},
			want: false,
		},
		{
			name: "not expired - future expiration",
			principal: &Principal{
				ExpiresAt: func() *time.Time {
					t := time.Now().Add(time.Hour)
					return &t
				}(),
			},
			want: false,
		},
		{
			name: "expired - past expiration",
			principal: &Principal{
				ExpiresAt: func() *time.Time {
					t := time.Now().Add(-time.Hour)
					return &t
				}(),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.principal.IsExpired(); got != tt.want {
				t.Errorf("Principal.IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrincipal_GetAttribute(t *testing.T) {
	principal := &Principal{
		Attributes: map[string]interface{}{
			"name":  "John",
			"email": "john@example.com",
			"age":   30,
		},
	}

	tests := []struct {
		name  string
		key   string
		want  interface{}
		wantOk bool
	}{
		{
			name:   "existing attribute",
			key:    "name",
			want:   "John",
			wantOk: true,
		},
		{
			name:   "non-existing attribute",
			key:    "missing",
			want:   nil,
			wantOk: false,
		},
		{
			name:   "nil attributes",
			key:    "name",
			want:   nil,
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := principal
			if tt.name == "nil attributes" {
				p = &Principal{Attributes: nil}
			}
			got, gotOk := p.GetAttribute(tt.key)
			if got != tt.want {
				t.Errorf("Principal.GetAttribute() got = %v, want %v", got, tt.want)
			}
			if gotOk != tt.wantOk {
				t.Errorf("Principal.GetAttribute() gotOk = %v, want %v", gotOk, tt.wantOk)
			}
		})
	}
}

func TestPrincipal_SetAttribute(t *testing.T) {
	principal := &Principal{Attributes: nil}

	principal.SetAttribute("name", "John")
	if principal.Attributes == nil {
		t.Error("SetAttribute should initialize Attributes map")
	}
	if principal.Attributes["name"] != "John" {
		t.Errorf("SetAttribute() = %v, want %v", principal.Attributes["name"], "John")
	}

	principal.SetAttribute("age", 30)
	if principal.Attributes["age"] != 30 {
		t.Errorf("SetAttribute() = %v, want %v", principal.Attributes["age"], 30)
	}
}


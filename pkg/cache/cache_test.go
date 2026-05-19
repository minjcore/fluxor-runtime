package cache

import (
	"context"
	"testing"
	"time"
)

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "valid key",
			key:     "user:123",
			wantErr: false,
		},
		{
			name:    "valid long key",
			key:     "user:123:profile:settings",
			wantErr: false,
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: true,
		},
		{
			name:    "key too long",
			key:     string(make([]byte, 251)), // 251 characters
			wantErr: true,
		},
		{
			name:    "key at max length",
			key:     string(make([]byte, 250)), // 250 characters (max)
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKey(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				// Verify error message contains "fail-fast"
				if err.Error()[:10] != "fail-fast:" {
					t.Errorf("ValidateKey() error message should start with 'fail-fast:', got %q", err.Error())
				}
			}
		})
	}
}

func TestValidateTTL(t *testing.T) {
	tests := []struct {
		name    string
		ttl     time.Duration
		wantErr bool
	}{
		{
			name:    "positive TTL",
			ttl:     5 * time.Minute,
			wantErr: false,
		},
		{
			name:    "zero TTL",
			ttl:     0,
			wantErr: false,
		},
		{
			name:    "negative TTL",
			ttl:     -1 * time.Second,
			wantErr: true,
		},
		{
			name:    "large TTL",
			ttl:     24 * time.Hour,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTTL(tt.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTTL(%v) error = %v, wantErr %v", tt.ttl, err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				// Verify error message contains "fail-fast"
				if err.Error()[:10] != "fail-fast:" {
					t.Errorf("ValidateTTL() error message should start with 'fail-fast:', got %q", err.Error())
				}
			}
		})
	}
}

func TestValidateContext(t *testing.T) {
	t.Run("valid context", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ValidateContext() should not panic with valid context, got: %v", r)
			}
		}()
		ctx := context.Background()
		ValidateContext(ctx)
	})

	t.Run("nil context", func(t *testing.T) {
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("ValidateContext() should panic with nil context")
			}
			err, ok := r.(error)
			if !ok {
				t.Fatalf("Expected error type, got: %T", r)
			}
			if err.Error()[:10] != "fail-fast:" {
				t.Errorf("Expected error message to start with 'fail-fast:', got %q", err.Error())
			}
		}()
		ValidateContext(nil)
	})

	t.Run("context with timeout", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ValidateContext() should not panic with context with timeout, got: %v", r)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		ValidateContext(ctx)
	})

	t.Run("context with cancel", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ValidateContext() should not panic with context with cancel, got: %v", r)
			}
		}()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ValidateContext(ctx)
	})
}

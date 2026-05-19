package core

import (
	"testing"
	"time"
)

func TestValidateAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{"valid address", "test.address", false},
		{"empty address", "", true},
		{"long address", string(make([]byte, 256)), true},
		{"normal address", "api.users", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAddress(tt.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		wantErr bool
	}{
		{"valid timeout", 5 * time.Second, false},
		{"zero timeout", 0, true},
		{"negative timeout", -1 * time.Second, true},
		{"too large timeout", 10 * time.Minute, true},
		{"max valid timeout", 5 * time.Minute, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimeout(tt.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTimeout() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateBody(t *testing.T) {
	tests := []struct {
		name    string
		body    interface{}
		wantErr bool
	}{
		{"valid body", "test", false},
		{"nil body", nil, true},
		{"map body", map[string]string{"key": "value"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBody(tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBody() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFailFast(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("FailFast() should panic")
		}
	}()

	FailFast(&EventBusError{Code: "TEST", Message: "test error"})
}

func TestFailFast_NoPanicWhenNoError(t *testing.T) {
	// FailFast should not panic when error is nil
	defer func() {
		if r := recover(); r != nil {
			t.Error("FailFast() should not panic when error is nil")
		}
	}()

	FailFast(nil)
}

func TestFailFastIf(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("FailFastIf() should panic when condition is true")
		}
	}()

	FailFastIf(true, "test condition")
}

func TestFailFastIf_NoPanicWhenConditionFalse(t *testing.T) {
	// FailFastIf should not panic when condition is false
	defer func() {
		if r := recover(); r != nil {
			t.Error("FailFastIf() should not panic when condition is false")
		}
	}()

	FailFastIf(false, "test condition")
}

// simpleTestVerticle is a simple implementation for testing
type simpleTestVerticle struct{}

func (v *simpleTestVerticle) Start(ctx FluxorContext) error { return nil }
func (v *simpleTestVerticle) Stop(ctx FluxorContext) error  { return nil }

func TestValidateVerticle(t *testing.T) {
	tests := []struct {
		name     string
		verticle Verticle
		wantErr  bool
	}{
		{"nil verticle", nil, true},
		{"valid verticle", &simpleTestVerticle{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateVerticle(tt.verticle)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateVerticle() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAddress_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{"exactly 255 chars", string(make([]byte, 255)), false},
		{"256 chars", string(make([]byte, 256)), true},
		{"255 chars with content", "a" + string(make([]byte, 254)), false},
		{"single char", "a", false},
		{"whitespace only", "   ", false}, // Currently allowed, test documents behavior
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAddress(tt.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAddress() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTimeout_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		wantErr bool
	}{
		{"one nanosecond", 1, false},
		{"one second", time.Second, false},
		{"exactly 5 minutes", 5 * time.Minute, false},
		{"just over 5 minutes", 5*time.Minute + time.Nanosecond, true},
		{"6 minutes", 6 * time.Minute, true},
		{"one hour", time.Hour, true},
		{"negative duration", -time.Second, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimeout(tt.timeout)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTimeout() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateBody_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		body    interface{}
		wantErr bool
	}{
		{"empty string", "", false},
		{"zero int", 0, false},
		{"zero float", 0.0, false},
		{"false bool", false, false},
		{"empty map", map[string]string{}, false},
		{"empty slice", []string{}, false},
		{"nil", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBody(tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBody() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

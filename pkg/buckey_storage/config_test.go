package buckey_storage

import (
	"encoding/json"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		backend string
		wantErr bool
	}{
		{"memory ok", BackendMemory, false},
		{"replicated ok", BackendReplicated, false},
		{"empty", "", true},
		{"unsupported", "unknown", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Backend: tt.backend}
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfig_LoadFromJSON(t *testing.T) {
	data := []byte(`{"backend":"memory","prefix":"app/"}`)
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if cfg.Backend != BackendMemory {
		t.Errorf("Backend: got %q", cfg.Backend)
	}
	if cfg.Prefix != "app/" {
		t.Errorf("Prefix: got %q", cfg.Prefix)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate: %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Backend != BackendMemory {
		t.Errorf("Default Backend: got %q", cfg.Backend)
	}
	if cfg.Prefix != "" {
		t.Errorf("Default Prefix: got %q", cfg.Prefix)
	}
}

package sbom

import (
	"encoding/json"
	"testing"
	"time"
)

func TestGenerator_Generate(t *testing.T) {
	generator := NewGenerator("test-project", FormatSPDX)

	components := []Component{
		{
			Name:    "example-lib",
			Version: "1.0.0",
			Type:    "library",
			PURL:    "pkg:golang/example-lib@1.0.0",
			Licenses: []string{"MIT"},
		},
		{
			Name:    "another-lib",
			Version: "2.1.0",
			Type:    "library",
			PURL:    "pkg:golang/another-lib@2.1.0",
			Licenses: []string{"Apache-2.0"},
		},
	}

	sbom, err := generator.Generate(components)
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	if sbom.Name != "test-project" {
		t.Errorf("Generate() name = %s, want test-project", sbom.Name)
	}

	if len(sbom.Components) != 2 {
		t.Errorf("Generate() components count = %d, want 2", len(sbom.Components))
	}
}

func TestGenerator_GenerateJSON(t *testing.T) {
	generator := NewGenerator("test-project", FormatSPDX)

	components := []Component{
		{
			Name:    "example-lib",
			Version: "1.0.0",
			Type:    "library",
			PURL:    "pkg:golang/example-lib@1.0.0",
		},
	}

	jsonData, err := generator.GenerateJSON(components)
	if err != nil {
		t.Fatalf("GenerateJSON() failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("GenerateJSON() produced invalid JSON: %v", err)
	}

	if result["name"] != "test-project" {
		t.Errorf("GenerateJSON() name = %v, want test-project", result["name"])
	}
}

func TestGenerator_CycloneDX(t *testing.T) {
	generator := NewGenerator("test-project", FormatCycloneDX)

	components := []Component{
		{
			Name:    "example-lib",
			Version: "1.0.0",
			Type:    "library",
			PURL:    "pkg:golang/example-lib@1.0.0",
		},
	}

	jsonData, err := generator.GenerateJSON(components)
	if err != nil {
		t.Fatalf("GenerateJSON() failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		t.Fatalf("GenerateJSON() produced invalid JSON: %v", err)
	}

	if result["bomFormat"] != "CycloneDX" {
		t.Errorf("GenerateJSON() bomFormat = %v, want CycloneDX", result["bomFormat"])
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		sbom    *SBOM
		wantErr bool
	}{
		{
			name: "Valid SBOM",
			sbom: &SBOM{
				Name:    "test",
				Format:  FormatSPDX,
				Created: time.Now(),
				Components: []Component{
					{Name: "comp1", Version: "1.0.0"},
				},
			},
			wantErr: false,
		},
		{
			name: "Missing name",
			sbom: &SBOM{
				Format: FormatSPDX,
				Components: []Component{
					{Name: "comp1", Version: "1.0.0"},
				},
			},
			wantErr: true,
		},
		{
			name: "Missing format",
			sbom: &SBOM{
				Name: "test",
				Components: []Component{
					{Name: "comp1", Version: "1.0.0"},
				},
			},
			wantErr: true,
		},
		{
			name: "No components",
			sbom: &SBOM{
				Name:     "test",
				Format:   FormatSPDX,
				Components: []Component{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.sbom)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

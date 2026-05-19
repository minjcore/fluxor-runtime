package supplychain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/security/sbom"
)

func TestNewProvenanceTracker(t *testing.T) {
	tracker := NewProvenanceTracker()

	if tracker == nil {
		t.Fatal("NewProvenanceTracker returned nil")
	}

	if tracker.provenances == nil {
		t.Fatal("Provenance tracker's provenances map is nil")
	}
}

func TestProvenanceTrackerAddAndGet(t *testing.T) {
	tracker := NewProvenanceTracker()

	component := sbom.Component{
		Name:    "test-component",
		Version: "1.0.0",
		Type:    "library",
	}

	provenance := &Provenance{
		Component:  component,
		Source:     "https://example.com/repo",
		Repository: "https://github.com/example/repo",
		Commit:     "abc123",
		Metadata:   make(map[string]interface{}),
	}

	tracker.AddProvenance(component, provenance)

	retrieved, exists := tracker.GetProvenance(component)
	if !exists {
		t.Fatal("Provenance not found")
	}

	if retrieved.Source != provenance.Source {
		t.Errorf("Expected source %s, got %s", provenance.Source, retrieved.Source)
	}

	if retrieved.Repository != provenance.Repository {
		t.Errorf("Expected repository %s, got %s", provenance.Repository, retrieved.Repository)
	}
}

func TestProvenanceTrackerVerifyProvenance(t *testing.T) {
	tracker := NewProvenanceTracker()

	component := sbom.Component{
		Name:    "test-component",
		Version: "1.0.0",
		Type:    "library",
	}

	provenance := &Provenance{
		Component:  component,
		Source:     "https://example.com/repo",
		Repository: "https://github.com/example/repo",
		Commit:     "abc123",
		Metadata:   make(map[string]interface{}),
	}

	tracker.AddProvenance(component, provenance)

	result, err := tracker.VerifyProvenance(component)
	if err != nil {
		t.Fatalf("VerifyProvenance failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Expected valid provenance, got invalid: %v", result.Issues)
	}
}

func TestProvenanceTrackerVerifyProvenanceMissing(t *testing.T) {
	tracker := NewProvenanceTracker()

	component := sbom.Component{
		Name:    "test-component",
		Version: "1.0.0",
		Type:    "library",
	}

	result, err := tracker.VerifyProvenance(component)
	if err != nil {
		t.Fatalf("VerifyProvenance failed: %v", err)
	}

	if result.Valid {
		t.Error("Expected invalid provenance for missing provenance")
	}

	if len(result.Issues) == 0 {
		t.Error("Expected issues to be reported")
	}
}

func TestProvenanceTrackerVerifyProvenanceInvalid(t *testing.T) {
	tracker := NewProvenanceTracker()

	component := sbom.Component{
		Name:    "test-component",
		Version: "1.0.0",
		Type:    "library",
	}

	// Provenance with missing source
	provenance := &Provenance{
		Component: component,
		// Source is missing
		Metadata: make(map[string]interface{}),
	}

	tracker.AddProvenance(component, provenance)

	result, err := tracker.VerifyProvenance(component)
	if err != nil {
		t.Fatalf("VerifyProvenance failed: %v", err)
	}

	if result.Valid {
		t.Error("Expected invalid provenance due to missing source")
	}
}

func TestExtractProvenanceFromSBOM(t *testing.T) {
	sbomDoc := &sbom.SBOM{
		Format:  sbom.FormatSPDX,
		Version: "1.0",
		Name:    "test-app",
		Created: time.Now(),
		Creator: sbom.Creator{
			Name: "Test",
		},
		Components: []sbom.Component{
			{
				Name:    "component-1",
				Version: "1.0.0",
				PURL:    "pkg:golang/component-1@1.0.0",
				ExternalRefs: []sbom.ExternalRef{
					{
						Type: "vcs",
						URL:  "https://github.com/example/repo",
					},
				},
				Properties: map[string]string{
					"repository": "https://github.com/example/repo",
					"commit":     "abc123",
				},
			},
		},
	}

	provenances := ExtractProvenanceFromSBOM(sbomDoc)

	if len(provenances) != 1 {
		t.Fatalf("Expected 1 provenance, got %d", len(provenances))
	}

	key := "component-1:1.0.0"
	prov, exists := provenances[key]
	if !exists {
		t.Fatal("Provenance not found for component-1:1.0.0")
	}

	if prov.Source != "pkg:golang/component-1@1.0.0" {
		t.Errorf("Expected source %s, got %s", "pkg:golang/component-1@1.0.0", prov.Source)
	}

	if prov.Repository != "https://github.com/example/repo" {
		t.Errorf("Expected repository %s, got %s", "https://github.com/example/repo", prov.Repository)
	}

	if prov.Commit != "abc123" {
		t.Errorf("Expected commit abc123, got %s", prov.Commit)
	}
}

func TestSerializeProvenance(t *testing.T) {
	provenance := &Provenance{
		Component: sbom.Component{
			Name:    "test-component",
			Version: "1.0.0",
		},
		Source:     "https://example.com/repo",
		Repository: "https://github.com/example/repo",
		Commit:     "abc123",
		Metadata:   make(map[string]interface{}),
	}

	data, err := SerializeProvenance(provenance)
	if err != nil {
		t.Fatalf("SerializeProvenance failed: %v", err)
	}

	if len(data) == 0 {
		t.Error("Serialized data is empty")
	}

	// Verify it's valid JSON
	var decoded Provenance
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal serialized provenance: %v", err)
	}

	if decoded.Source != provenance.Source {
		t.Errorf("Expected source %s, got %s", provenance.Source, decoded.Source)
	}
}

func TestDeserializeProvenance(t *testing.T) {
	provenance := &Provenance{
		Component: sbom.Component{
			Name:    "test-component",
			Version: "1.0.0",
		},
		Source:     "https://example.com/repo",
		Repository: "https://github.com/example/repo",
		Commit:     "abc123",
		Metadata:   make(map[string]interface{}),
	}

	data, err := json.Marshal(provenance)
	if err != nil {
		t.Fatalf("Failed to marshal provenance: %v", err)
	}

	decoded, err := DeserializeProvenance(data)
	if err != nil {
		t.Fatalf("DeserializeProvenance failed: %v", err)
	}

	if decoded.Source != provenance.Source {
		t.Errorf("Expected source %s, got %s", provenance.Source, decoded.Source)
	}

	if decoded.Repository != provenance.Repository {
		t.Errorf("Expected repository %s, got %s", provenance.Repository, decoded.Repository)
	}
}

func TestDeserializeProvenanceInvalidJSON(t *testing.T) {
	invalidJSON := []byte("{ invalid json }")

	_, err := DeserializeProvenance(invalidJSON)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestProvenanceWithBuildInfo(t *testing.T) {
	provenance := &Provenance{
		Component: sbom.Component{
			Name:    "test-component",
			Version: "1.0.0",
		},
		Source:  "https://example.com/repo",
		BuildInfo: &BuildInfo{
			Builder:   "go",
			BuildID:   "build-123",
			BuildTime: time.Now(),
			BuildConfig: map[string]interface{}{
				"go_version": "1.21",
			},
		},
		Metadata: make(map[string]interface{}),
	}

	tracker := NewProvenanceTracker()
	tracker.AddProvenance(provenance.Component, provenance)

	result, err := tracker.VerifyProvenance(provenance.Component)
	if err != nil {
		t.Fatalf("VerifyProvenance failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Expected valid provenance, got invalid: %v", result.Issues)
	}
}

func TestProvenanceWithAttestations(t *testing.T) {
	provenance := &Provenance{
		Component: sbom.Component{
			Name:    "test-component",
			Version: "1.0.0",
		},
		Source: "https://example.com/repo",
		Attestations: []Attestation{
			{
				Type:    "slsa",
				Content: "attestation content",
				SignedBy: "builder",
				SignedAt: time.Now(),
			},
		},
		Metadata: make(map[string]interface{}),
	}

	tracker := NewProvenanceTracker()
	tracker.AddProvenance(provenance.Component, provenance)

	result, err := tracker.VerifyProvenance(provenance.Component)
	if err != nil {
		t.Fatalf("VerifyProvenance failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Expected valid provenance, got invalid: %v", result.Issues)
	}
}

func TestProvenanceWithInvalidAttestations(t *testing.T) {
	provenance := &Provenance{
		Component: sbom.Component{
			Name:    "test-component",
			Version: "1.0.0",
		},
		Source: "https://example.com/repo",
		Attestations: []Attestation{
			{
				// Type is missing
				Content: "attestation content",
			},
		},
		Metadata: make(map[string]interface{}),
	}

	tracker := NewProvenanceTracker()
	tracker.AddProvenance(provenance.Component, provenance)

	result, err := tracker.VerifyProvenance(provenance.Component)
	if err != nil {
		t.Fatalf("VerifyProvenance failed: %v", err)
	}

	if result.Valid {
		t.Error("Expected invalid provenance due to missing attestation type")
	}
}

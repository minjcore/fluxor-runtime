package supplychain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/pkg/security/sbom"
)

// Provenance represents the provenance information for a component
type Provenance struct {
	Component     sbom.Component          `json:"component"`
	Source        string                  `json:"source"`
	Repository    string                  `json:"repository,omitempty"`
	Commit        string                  `json:"commit,omitempty"`
	BuildInfo     *BuildInfo              `json:"build_info,omitempty"`
	Attestations  []Attestation           `json:"attestations,omitempty"`
	Verified      bool                    `json:"verified"`
	VerifiedAt    time.Time               `json:"verified_at,omitempty"`
	Metadata      map[string]interface{}  `json:"metadata,omitempty"`
}

// BuildInfo represents build information
type BuildInfo struct {
	Builder     string                 `json:"builder"`
	BuildID     string                 `json:"build_id,omitempty"`
	BuildTime   time.Time              `json:"build_time,omitempty"`
	BuildConfig map[string]interface{} `json:"build_config,omitempty"`
}

// Attestation represents a provenance attestation
type Attestation struct {
	Type        string                 `json:"type"`
	Content     string                 `json:"content"`
	Signature   string                 `json:"signature,omitempty"`
	SignedBy    string                 `json:"signed_by,omitempty"`
	SignedAt    time.Time              `json:"signed_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ProvenanceTracker tracks component provenance
type ProvenanceTracker struct {
	provenances map[string]*Provenance
}

// NewProvenanceTracker creates a new provenance tracker
func NewProvenanceTracker() *ProvenanceTracker {
	return &ProvenanceTracker{
		provenances: make(map[string]*Provenance),
	}
}

// AddProvenance adds provenance information for a component
func (pt *ProvenanceTracker) AddProvenance(component sbom.Component, provenance *Provenance) {
	key := pt.componentKey(component)
	pt.provenances[key] = provenance
}

// GetProvenance retrieves provenance information for a component
func (pt *ProvenanceTracker) GetProvenance(component sbom.Component) (*Provenance, bool) {
	key := pt.componentKey(component)
	provenance, exists := pt.provenances[key]
	return provenance, exists
}

// VerifyProvenance verifies the provenance of a component
func (pt *ProvenanceTracker) VerifyProvenance(component sbom.Component) (*ProvenanceVerificationResult, error) {
	provenance, exists := pt.GetProvenance(component)
	if !exists {
		return &ProvenanceVerificationResult{
			Component: component,
			Valid:     false,
			Issues:    []string{"No provenance information found"},
		}, nil
	}

	result := &ProvenanceVerificationResult{
		Component:  component,
		Provenance: provenance,
		Valid:      true,
		Issues:     make([]string, 0),
		VerifiedAt: time.Now(),
	}

	// Verify source
	if provenance.Source == "" {
		result.Valid = false
		result.Issues = append(result.Issues, "Provenance source is missing")
	}

	// Verify repository if provided
	if provenance.Repository != "" {
		// Could add URL validation here
	}

	// Verify attestations
	for i, att := range provenance.Attestations {
		if att.Type == "" {
			result.Valid = false
			result.Issues = append(result.Issues, fmt.Sprintf("Attestation %d missing type", i))
		}
		if att.Content == "" {
			result.Valid = false
			result.Issues = append(result.Issues, fmt.Sprintf("Attestation %d missing content", i))
		}
	}

	provenance.Verified = result.Valid
	provenance.VerifiedAt = result.VerifiedAt

	return result, nil
}

// componentKey generates a key for a component
func (pt *ProvenanceTracker) componentKey(component sbom.Component) string {
	return fmt.Sprintf("%s:%s", component.Name, component.Version)
}

// ProvenanceVerificationResult represents the result of provenance verification
type ProvenanceVerificationResult struct {
	Component  sbom.Component          `json:"component"`
	Provenance *Provenance             `json:"provenance,omitempty"`
	Valid      bool                    `json:"valid"`
	Issues     []string                `json:"issues,omitempty"`
	VerifiedAt time.Time               `json:"verified_at"`
	Metadata   map[string]interface{}  `json:"metadata,omitempty"`
}

// ExtractProvenanceFromSBOM extracts provenance information from an SBOM
func ExtractProvenanceFromSBOM(sbomDoc *sbom.SBOM) map[string]*Provenance {
	provenances := make(map[string]*Provenance)

	for _, comp := range sbomDoc.Components {
		provenance := &Provenance{
			Component: comp,
			Source:    comp.PURL,
			Metadata:  make(map[string]interface{}),
		}

		// Extract from external refs
		for _, ref := range comp.ExternalRefs {
			switch ref.Type {
			case "vcs":
				provenance.Repository = ref.URL
			case "build":
				if provenance.BuildInfo == nil {
					provenance.BuildInfo = &BuildInfo{}
				}
				// Could parse build info from ref
			}
		}

		// Extract from properties
		if comp.Properties != nil {
			if repo, ok := comp.Properties["repository"]; ok {
				provenance.Repository = repo
			}
			if commit, ok := comp.Properties["commit"]; ok {
				provenance.Commit = commit
			}
		}

		key := fmt.Sprintf("%s:%s", comp.Name, comp.Version)
		provenances[key] = provenance
	}

	return provenances
}

// SerializeProvenance serializes provenance to JSON
func SerializeProvenance(provenance *Provenance) ([]byte, error) {
	return json.MarshalIndent(provenance, "", "  ")
}

// DeserializeProvenance deserializes provenance from JSON
func DeserializeProvenance(data []byte) (*Provenance, error) {
	var provenance Provenance
	if err := json.Unmarshal(data, &provenance); err != nil {
		return nil, fmt.Errorf("failed to unmarshal provenance: %w", err)
	}
	return &provenance, nil
}

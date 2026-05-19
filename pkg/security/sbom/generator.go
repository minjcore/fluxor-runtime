package sbom

import (
	"encoding/json"
	"fmt"
	"runtime"
	"time"
)

// SBOMFormat represents the SBOM format
type SBOMFormat string

const (
	// FormatSPDX represents SPDX (Software Package Data Exchange) format
	FormatSPDX SBOMFormat = "spdx"
	// FormatCycloneDX represents CycloneDX format
	FormatCycloneDX SBOMFormat = "cyclonedx"
)

// Component represents a software component in the SBOM
type Component struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Type         string            `json:"type"` // library, application, framework, etc.
	PURL         string            `json:"purl,omitempty"` // Package URL
	CPE          string            `json:"cpe,omitempty"`  // Common Platform Enumeration
	Licenses     []string          `json:"licenses,omitempty"`
	Hashes       map[string]string `json:"hashes,omitempty"` // algorithm -> hash value
	ExternalRefs []ExternalRef    `json:"externalRefs,omitempty"`
	Properties   map[string]string  `json:"properties,omitempty"`
}

// ExternalRef represents an external reference to a component
type ExternalRef struct {
	Type    string `json:"type"`
	URL     string `json:"url,omitempty"`
	Comment string `json:"comment,omitempty"`
}

// SBOM represents a Software Bill of Materials
type SBOM struct {
	Format      SBOMFormat          `json:"format"`
	Version     string              `json:"sbomVersion"`
	Name        string              `json:"name"`
	Created     time.Time           `json:"created"`
	Creator     Creator             `json:"creator"`
	Components  []Component         `json:"components"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Relationships []Relationship    `json:"relationships,omitempty"`
}

// Creator represents the entity that created the SBOM
type Creator struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Tool    string `json:"tool,omitempty"`
}

// Relationship represents a relationship between components
type Relationship struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Type      string `json:"type"` // DEPENDS_ON, CONTAINS, etc.
	Confidence float64 `json:"confidence,omitempty"`
}

// Generator generates Software Bill of Materials
type Generator struct {
	format   SBOMFormat
	name     string
	creator  Creator
	metadata map[string]interface{}
}

// NewGenerator creates a new SBOM generator
func NewGenerator(name string, format SBOMFormat) *Generator {
	return &Generator{
		format:  format,
		name:    name,
		creator: Creator{
			Name:    "Fluxor",
			Version: getVersion(),
			Tool:    "fluxor-sbom-generator",
		},
		metadata: make(map[string]interface{}),
	}
}

// getVersion returns the current Go version
func getVersion() string {
	return runtime.Version()
}

// AddComponent adds a component to the SBOM
func (g *Generator) AddComponent(component Component) {
	// Implementation would be in the Generate method
}

// SetCreator sets the creator information
func (g *Generator) SetCreator(creator Creator) {
	g.creator = creator
}

// SetMetadata sets metadata for the SBOM
func (g *Generator) SetMetadata(key string, value interface{}) {
	if g.metadata == nil {
		g.metadata = make(map[string]interface{})
	}
	g.metadata[key] = value
}

// Generate creates an SBOM document
func (g *Generator) Generate(components []Component) (*SBOM, error) {
	sbom := &SBOM{
		Format:     g.format,
		Version:    "1.0",
		Name:       g.name,
		Created:    time.Now(),
		Creator:    g.creator,
		Components: components,
		Metadata:   g.metadata,
	}

	return sbom, nil
}

// GenerateJSON generates an SBOM in JSON format
func (g *Generator) GenerateJSON(components []Component) ([]byte, error) {
	sbom, err := g.Generate(components)
	if err != nil {
		return nil, err
	}

	switch g.format {
	case FormatSPDX:
		return g.generateSPDXJSON(sbom)
	case FormatCycloneDX:
		return g.generateCycloneDXJSON(sbom)
	default:
		return json.MarshalIndent(sbom, "", "  ")
	}
}

// generateSPDXJSON generates SPDX-compliant JSON
func (g *Generator) generateSPDXJSON(sbom *SBOM) ([]byte, error) {
	spdx := map[string]interface{}{
		"spdxVersion": "SPDX-2.3",
		"dataLicense": "CC0-1.0",
		"SPDXID":      "SPDXRef-DOCUMENT",
		"name":        sbom.Name,
		"documentNamespace": fmt.Sprintf("https://fluxor.io/spdx/%s", sbom.Name),
		"creationInfo": map[string]interface{}{
			"created": sbom.Created.Format(time.RFC3339),
			"creators": []string{
				fmt.Sprintf("Tool: %s-%s", sbom.Creator.Tool, sbom.Creator.Version),
				fmt.Sprintf("Organization: %s", sbom.Creator.Name),
			},
		},
		"packages": g.convertComponentsToSPDX(sbom.Components),
	}

	return json.MarshalIndent(spdx, "", "  ")
}

// generateCycloneDXJSON generates CycloneDX-compliant JSON
func (g *Generator) generateCycloneDXJSON(sbom *SBOM) ([]byte, error) {
	cyclonedx := map[string]interface{}{
		"bomFormat":    "CycloneDX",
		"specVersion":  "1.5",
		"version":      1,
		"metadata": map[string]interface{}{
			"timestamp": sbom.Created.Format(time.RFC3339),
			"tools": []map[string]interface{}{
				{
					"name":    sbom.Creator.Tool,
					"version": sbom.Creator.Version,
				},
			},
		},
		"components": g.convertComponentsToCycloneDX(sbom.Components),
	}

	return json.MarshalIndent(cyclonedx, "", "  ")
}

// convertComponentsToSPDX converts components to SPDX format
func (g *Generator) convertComponentsToSPDX(components []Component) []map[string]interface{} {
	packages := make([]map[string]interface{}, 0, len(components))
	for i, comp := range components {
		pkg := map[string]interface{}{
			"SPDXID":      fmt.Sprintf("SPDXRef-Package-%d", i+1),
			"name":        comp.Name,
			"versionInfo": comp.Version,
			"downloadLocation": "NOASSERTION",
		}

		if comp.PURL != "" {
			pkg["externalRefs"] = []map[string]interface{}{
				{
					"referenceCategory": "PACKAGE-MANAGER",
					"referenceType":     "purl",
					"referenceLocator":  comp.PURL,
				},
			}
		}

		if len(comp.Licenses) > 0 {
			pkg["licenseDeclared"] = comp.Licenses[0]
		}

		packages = append(packages, pkg)
	}
	return packages
}

// convertComponentsToCycloneDX converts components to CycloneDX format
func (g *Generator) convertComponentsToCycloneDX(components []Component) []map[string]interface{} {
	cdxComponents := make([]map[string]interface{}, 0, len(components))
	for _, comp := range components {
		cdxComp := map[string]interface{}{
			"type":    comp.Type,
			"name":    comp.Name,
			"version": comp.Version,
		}

		if comp.PURL != "" {
			cdxComp["purl"] = comp.PURL
		}

		if comp.CPE != "" {
			cdxComp["cpe"] = comp.CPE
		}

		if len(comp.Hashes) > 0 {
			hashes := make([]map[string]string, 0)
			for alg, hash := range comp.Hashes {
				hashes = append(hashes, map[string]string{
					"alg":  alg,
					"content": hash,
				})
			}
			cdxComp["hashes"] = hashes
		}

		if len(comp.Licenses) > 0 {
			licenses := make([]map[string]interface{}, 0)
			for _, license := range comp.Licenses {
				licenses = append(licenses, map[string]interface{}{
					"license": map[string]interface{}{
						"id": license,
					},
				})
			}
			cdxComp["licenses"] = licenses
		}

		cdxComponents = append(cdxComponents, cdxComp)
	}
	return cdxComponents
}

// Validate validates an SBOM document
func Validate(sbom *SBOM) error {
	if sbom.Name == "" {
		return fmt.Errorf("SBOM name is required")
	}
	if len(sbom.Components) == 0 {
		return fmt.Errorf("SBOM must contain at least one component")
	}
	if sbom.Format == "" {
		return fmt.Errorf("SBOM format is required")
	}
	return nil
}

// Package sbom provides Software Bill of Materials (SBOM) generation and management.
//
// An SBOM is a comprehensive inventory of software components, dependencies, and their relationships.
// This package supports generating SBOMs in multiple formats including SPDX and CycloneDX.
//
// Features:
//   - Generate SBOMs in SPDX format (version 2.3+)
//   - Generate SBOMs in CycloneDX format (version 1.5+)
//   - Component tracking with metadata (licenses, hashes, PURLs, CPEs)
//   - Relationship tracking between components
//   - SBOM validation
//
// Example usage:
//
//	generator := sbom.NewGenerator("my-application", sbom.FormatSPDX)
//
//	components := []sbom.Component{
//	    {
//	        Name:    "example-library",
//	        Version: "1.0.0",
//	        Type:    "library",
//	        PURL:    "pkg:golang/example-library@1.0.0",
//	        Licenses: []string{"MIT"},
//	    },
//	}
//
//	jsonData, err := generator.GenerateJSON(components)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Save or distribute the SBOM
//	os.WriteFile("sbom.json", jsonData, 0644)
//
// SBOMs are essential for:
//   - Supply chain security
//   - Vulnerability management
//   - License compliance
//   - Dependency tracking
//   - Regulatory compliance (e.g., Executive Order 14028)
//
// Path: pkg/security/sbom
package sbom

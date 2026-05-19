// Package supplychain provides supply chain security verification and management.
//
// Supply chain security is critical for ensuring the integrity, authenticity, and safety
// of software components and dependencies. This package provides comprehensive tools for
// verifying Software Bill of Materials (SBOMs), detecting vulnerabilities, tracking
// component provenance, and detecting tampering.
//
// Features:
//   - SBOM verification (integrity and signature verification)
//   - Dependency integrity checking
//   - Vulnerability scanning and assessment
//   - Component provenance tracking and verification
//   - Tampering detection
//   - Component-level verification
//
// Example usage:
//
//	// Create a verifier with default configuration
//	config := supplychain.DefaultVerifierConfig()
//	config.EnableVulnerabilityCheck = true
//	config.EnableSBOMSignatureCheck = true
//	config.PublicKeyPEM = publicKeyBytes
//
//	verifier := supplychain.NewVerifier(config)
//
//	// Verify an SBOM
//	result, err := verifier.VerifySBOM(sbomDoc, sbomJSON, signature)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if !result.Valid {
//	    fmt.Printf("SBOM verification failed:\n")
//	    for _, err := range result.Errors {
//	        fmt.Printf("  - %s\n", err)
//	    }
//	}
//
//	// Scan for vulnerabilities
//	scanner := supplychain.NewBasicVulnerabilityScanner()
//	vulns, err := scanner.Scan(sbomDoc)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	for _, vuln := range vulns {
//	    fmt.Printf("Found vulnerability: %s (Severity: %s)\n", vuln.ID, vuln.Severity)
//	}
//
//	// Track component provenance
//	tracker := supplychain.NewProvenanceTracker()
//	provenance := &supplychain.Provenance{
//	    Component:  component,
//	    Source:     "https://example.com/repo",
//	    Repository: "https://github.com/example/repo",
//	    Commit:     "abc123",
//	}
//	tracker.AddProvenance(component, provenance)
//
//	// Verify provenance
//	provResult, err := tracker.VerifyProvenance(component)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	if provResult.Valid {
//	    fmt.Println("Provenance verified successfully")
//	}
//
// This package integrates with:
//   - pkg/security/sbom: For SBOM generation and validation
//   - pkg/security/signing: For signature verification
//
// Supply chain security is essential for:
//   - Compliance with security regulations (e.g., Executive Order 14028)
//   - Detecting compromised dependencies
//   - Ensuring software integrity
//   - Tracking component origins
//   - Vulnerability management
//
// Path: pkg/security/supplychain
package supplychain

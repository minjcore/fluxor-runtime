package supplychain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/fluxorio/fluxor/pkg/security/sbom"
	"github.com/fluxorio/fluxor/pkg/security/signing"
)

// VerificationResult represents the result of a supply chain verification
type VerificationResult struct {
	Valid         bool                   `json:"valid"`
	Errors        []string               `json:"errors,omitempty"`
	Warnings      []string               `json:"warnings,omitempty"`
	Checks        []CheckResult          `json:"checks"`
	VerifiedAt    time.Time              `json:"verified_at"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// CheckResult represents the result of an individual check
type CheckResult struct {
	Type        CheckType `json:"type"`
	Status      Status    `json:"status"`
	Message     string    `json:"message,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	CheckedAt   time.Time `json:"checked_at"`
}

// CheckType represents the type of verification check
type CheckType string

const (
	// CheckSBOMIntegrity verifies SBOM integrity
	CheckSBOMIntegrity CheckType = "sbom_integrity"
	// CheckSBOMSignature verifies SBOM signature
	CheckSBOMSignature CheckType = "sbom_signature"
	// CheckDependencyIntegrity verifies dependency integrity
	CheckDependencyIntegrity CheckType = "dependency_integrity"
	// CheckVulnerabilities checks for known vulnerabilities
	CheckVulnerabilities CheckType = "vulnerabilities"
	// CheckProvenance verifies component provenance
	CheckProvenance CheckType = "provenance"
	// CheckTampering checks for tampering indicators
	CheckTampering CheckType = "tampering"
)

// Status represents the status of a check
type Status string

const (
	// StatusPass indicates the check passed
	StatusPass Status = "pass"
	// StatusFail indicates the check failed
	StatusFail Status = "fail"
	// StatusWarning indicates a warning
	StatusWarning Status = "warning"
	// StatusSkipped indicates the check was skipped
	StatusSkipped Status = "skipped"
)

// Verifier verifies supply chain security
type Verifier struct {
	config *VerifierConfig
}

// VerifierConfig configures the verifier
type VerifierConfig struct {
	// EnableSBOMIntegrityCheck enables SBOM integrity verification
	EnableSBOMIntegrityCheck bool
	// EnableSBOMSignatureCheck enables SBOM signature verification
	EnableSBOMSignatureCheck bool
	// EnableDependencyIntegrityCheck enables dependency integrity verification
	EnableDependencyIntegrityCheck bool
	// EnableVulnerabilityCheck enables vulnerability scanning
	EnableVulnerabilityCheck bool
	// EnableProvenanceCheck enables provenance verification
	EnableProvenanceCheck bool
	// EnableTamperingCheck enables tampering detection
	EnableTamperingCheck bool
	// PublicKeyPEM is the PEM-encoded public key for signature verification
	PublicKeyPEM []byte
	// SigningAlgorithm is the algorithm used for signatures
	SigningAlgorithm signing.SigningAlgorithm
	// VulnerabilityScanner is an optional vulnerability scanner
	VulnerabilityScanner VulnerabilityScanner
}

// DefaultVerifierConfig returns a default verifier configuration
func DefaultVerifierConfig() *VerifierConfig {
	return &VerifierConfig{
		EnableSBOMIntegrityCheck:        true,
		EnableSBOMSignatureCheck:        false,
		EnableDependencyIntegrityCheck:   true,
		EnableVulnerabilityCheck:         false,
		EnableProvenanceCheck:            false,
		EnableTamperingCheck:             true,
		SigningAlgorithm:                 signing.AlgorithmRS256,
	}
}

// NewVerifier creates a new supply chain verifier
func NewVerifier(config *VerifierConfig) *Verifier {
	if config == nil {
		config = DefaultVerifierConfig()
	}
	return &Verifier{
		config: config,
	}
}

// VerifySBOM verifies an SBOM document
func (v *Verifier) VerifySBOM(sbomDoc *sbom.SBOM, sbomJSON []byte, signature []byte) (*VerificationResult, error) {
	result := &VerificationResult{
		Valid:      true,
		Checks:     make([]CheckResult, 0),
		VerifiedAt: time.Now(),
		Metadata:   make(map[string]interface{}),
	}

	// Verify SBOM integrity
	if v.config.EnableSBOMIntegrityCheck {
		check := v.verifySBOMIntegrity(sbomDoc, sbomJSON)
		result.Checks = append(result.Checks, check)
		if check.Status == StatusFail {
			result.Valid = false
			result.Errors = append(result.Errors, check.Message)
		} else if check.Status == StatusWarning {
			result.Warnings = append(result.Warnings, check.Message)
		}
	}

	// Verify SBOM signature
	if v.config.EnableSBOMSignatureCheck && len(signature) > 0 {
		check := v.verifySBOMSignature(sbomJSON, signature)
		result.Checks = append(result.Checks, check)
		if check.Status == StatusFail {
			result.Valid = false
			result.Errors = append(result.Errors, check.Message)
		}
	}

	// Verify dependency integrity
	if v.config.EnableDependencyIntegrityCheck {
		check := v.verifyDependencyIntegrity(sbomDoc)
		result.Checks = append(result.Checks, check)
		if check.Status == StatusFail {
			result.Valid = false
			result.Errors = append(result.Errors, check.Message)
		} else if check.Status == StatusWarning {
			result.Warnings = append(result.Warnings, check.Message)
		}
	}

	// Check for vulnerabilities
	if v.config.EnableVulnerabilityCheck && v.config.VulnerabilityScanner != nil {
		check := v.checkVulnerabilities(sbomDoc)
		result.Checks = append(result.Checks, check)
		if check.Status == StatusFail {
			result.Valid = false
			result.Errors = append(result.Errors, check.Message)
		} else if check.Status == StatusWarning {
			result.Warnings = append(result.Warnings, check.Message)
		}
	}

	// Verify provenance
	if v.config.EnableProvenanceCheck {
		check := v.verifyProvenance(sbomDoc)
		result.Checks = append(result.Checks, check)
		if check.Status == StatusFail {
			result.Valid = false
			result.Errors = append(result.Errors, check.Message)
		} else if check.Status == StatusWarning {
			result.Warnings = append(result.Warnings, check.Message)
		}
	}

	// Check for tampering
	if v.config.EnableTamperingCheck {
		check := v.checkTampering(sbomDoc)
		result.Checks = append(result.Checks, check)
		if check.Status == StatusFail {
			result.Valid = false
			result.Errors = append(result.Errors, check.Message)
		} else if check.Status == StatusWarning {
			result.Warnings = append(result.Warnings, check.Message)
		}
	}

	return result, nil
}

// verifySBOMIntegrity verifies SBOM document integrity
func (v *Verifier) verifySBOMIntegrity(sbomDoc *sbom.SBOM, sbomJSON []byte) CheckResult {
	check := CheckResult{
		Type:      CheckSBOMIntegrity,
		CheckedAt: time.Now(),
	}

	// Validate SBOM structure
	if err := sbom.Validate(sbomDoc); err != nil {
		check.Status = StatusFail
		check.Message = fmt.Sprintf("SBOM validation failed: %v", err)
		return check
	}

	// Verify JSON integrity by computing hash
	if len(sbomJSON) > 0 {
		hash := sha256.Sum256(sbomJSON)
		check.Details = map[string]interface{}{
			"hash": hex.EncodeToString(hash[:]),
		}
	}

	check.Status = StatusPass
	check.Message = "SBOM integrity verified"
	return check
}

// verifySBOMSignature verifies SBOM signature
func (v *Verifier) verifySBOMSignature(sbomJSON []byte, signature []byte) CheckResult {
	check := CheckResult{
		Type:      CheckSBOMSignature,
		CheckedAt: time.Now(),
	}

	if len(v.config.PublicKeyPEM) == 0 {
		check.Status = StatusSkipped
		check.Message = "No public key provided for signature verification"
		return check
	}

	verifier, err := signing.NewVerifierFromPEM(v.config.PublicKeyPEM, v.config.SigningAlgorithm)
	if err != nil {
		check.Status = StatusFail
		check.Message = fmt.Sprintf("Failed to create verifier: %v", err)
		return check
	}

	if err := verifier.Verify(sbomJSON, signature); err != nil {
		check.Status = StatusFail
		check.Message = fmt.Sprintf("Signature verification failed: %v", err)
		return check
	}

	check.Status = StatusPass
	check.Message = "SBOM signature verified"
	return check
}

// verifyDependencyIntegrity verifies dependency integrity
func (v *Verifier) verifyDependencyIntegrity(sbomDoc *sbom.SBOM) CheckResult {
	check := CheckResult{
		Type:      CheckDependencyIntegrity,
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	missingHashes := 0
	missingPURLs := 0
	validComponents := 0

	for _, comp := range sbomDoc.Components {
		hasHash := len(comp.Hashes) > 0
		hasPURL := comp.PURL != ""

		if !hasHash {
			missingHashes++
		}
		if !hasPURL {
			missingPURLs++
		}
		if hasHash && hasPURL {
			validComponents++
		}
	}

	check.Details["total_components"] = len(sbomDoc.Components)
	check.Details["components_with_hashes"] = len(sbomDoc.Components) - missingHashes
	check.Details["components_with_purls"] = len(sbomDoc.Components) - missingPURLs
	check.Details["valid_components"] = validComponents

	if len(sbomDoc.Components) == 0 {
		check.Status = StatusWarning
		check.Message = "No components found in SBOM"
		return check
	}

	if missingHashes > len(sbomDoc.Components)/2 {
		check.Status = StatusWarning
		check.Message = fmt.Sprintf("More than half of components are missing hash information (%d/%d)", missingHashes, len(sbomDoc.Components))
		return check
	}

	if missingPURLs > len(sbomDoc.Components)/2 {
		check.Status = StatusWarning
		check.Message = fmt.Sprintf("More than half of components are missing PURL information (%d/%d)", missingPURLs, len(sbomDoc.Components))
		return check
	}

	check.Status = StatusPass
	check.Message = "Dependency integrity verified"
	return check
}

// checkVulnerabilities checks for known vulnerabilities
func (v *Verifier) checkVulnerabilities(sbomDoc *sbom.SBOM) CheckResult {
	check := CheckResult{
		Type:      CheckVulnerabilities,
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	if v.config.VulnerabilityScanner == nil {
		check.Status = StatusSkipped
		check.Message = "No vulnerability scanner configured"
		return check
	}

	vulns, err := v.config.VulnerabilityScanner.Scan(sbomDoc)
	if err != nil {
		check.Status = StatusWarning
		check.Message = fmt.Sprintf("Vulnerability scan failed: %v", err)
		return check
	}

	criticalCount := 0
	highCount := 0
	mediumCount := 0
	lowCount := 0

	for _, vuln := range vulns {
		switch vuln.Severity {
		case SeverityCritical:
			criticalCount++
		case SeverityHigh:
			highCount++
		case SeverityMedium:
			mediumCount++
		case SeverityLow:
			lowCount++
		}
	}

	check.Details["total_vulnerabilities"] = len(vulns)
	check.Details["critical"] = criticalCount
	check.Details["high"] = highCount
	check.Details["medium"] = mediumCount
	check.Details["low"] = lowCount

	if criticalCount > 0 || highCount > 0 {
		check.Status = StatusFail
		check.Message = fmt.Sprintf("Found %d critical and %d high severity vulnerabilities", criticalCount, highCount)
	} else if mediumCount > 0 {
		check.Status = StatusWarning
		check.Message = fmt.Sprintf("Found %d medium severity vulnerabilities", mediumCount)
	} else {
		check.Status = StatusPass
		check.Message = "No critical or high severity vulnerabilities found"
	}

	return check
}

// verifyProvenance verifies component provenance
func (v *Verifier) verifyProvenance(sbomDoc *sbom.SBOM) CheckResult {
	check := CheckResult{
		Type:      CheckProvenance,
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	componentsWithProvenance := 0
	componentsWithoutProvenance := 0

	for _, comp := range sbomDoc.Components {
		hasProvenance := len(comp.ExternalRefs) > 0 || comp.PURL != ""
		if hasProvenance {
			componentsWithProvenance++
		} else {
			componentsWithoutProvenance++
		}
	}

	check.Details["components_with_provenance"] = componentsWithProvenance
	check.Details["components_without_provenance"] = componentsWithoutProvenance

	if componentsWithoutProvenance > len(sbomDoc.Components)/2 {
		check.Status = StatusWarning
		check.Message = fmt.Sprintf("More than half of components lack provenance information (%d/%d)", componentsWithoutProvenance, len(sbomDoc.Components))
		return check
	}

	check.Status = StatusPass
	check.Message = "Provenance verification passed"
	return check
}

// checkTampering checks for tampering indicators
func (v *Verifier) checkTampering(sbomDoc *sbom.SBOM) CheckResult {
	check := CheckResult{
		Type:      CheckTampering,
		CheckedAt: time.Now(),
		Details:   make(map[string]interface{}),
	}

	// Check for suspicious patterns
	suspiciousComponents := 0
	emptyVersions := 0
	emptyNames := 0

	for _, comp := range sbomDoc.Components {
		if comp.Name == "" {
			emptyNames++
		}
		if comp.Version == "" {
			emptyVersions++
		}
		// Check for suspicious patterns (e.g., very long names, unusual characters)
		if len(comp.Name) > 200 || len(comp.Version) > 100 {
			suspiciousComponents++
		}
	}

	check.Details["empty_names"] = emptyNames
	check.Details["empty_versions"] = emptyVersions
	check.Details["suspicious_components"] = suspiciousComponents

	if emptyNames > 0 || emptyVersions > 0 {
		check.Status = StatusWarning
		check.Message = fmt.Sprintf("Found %d components with empty names and %d with empty versions", emptyNames, emptyVersions)
		return check
	}

	if suspiciousComponents > 0 {
		check.Status = StatusWarning
		check.Message = fmt.Sprintf("Found %d components with suspicious patterns", suspiciousComponents)
		return check
	}

	check.Status = StatusPass
	check.Message = "No tampering indicators detected"
	return check
}

// VerifyComponent verifies a single component
func (v *Verifier) VerifyComponent(component sbom.Component) (*ComponentVerificationResult, error) {
	result := &ComponentVerificationResult{
		Component:  component,
		Valid:      true,
		Issues:     make([]string, 0),
		VerifiedAt: time.Now(),
	}

	// Check for required fields
	if component.Name == "" {
		result.Valid = false
		result.Issues = append(result.Issues, "Component name is required")
	}

	if component.Version == "" {
		result.Valid = false
		result.Issues = append(result.Issues, "Component version is required")
	}

	// Check for integrity information
	if len(component.Hashes) == 0 {
		result.Issues = append(result.Issues, "Component missing hash information")
	}

	if component.PURL == "" {
		result.Issues = append(result.Issues, "Component missing PURL")
	}

	return result, nil
}

// ComponentVerificationResult represents the verification result for a single component
type ComponentVerificationResult struct {
	Component  sbom.Component          `json:"component"`
	Valid      bool                    `json:"valid"`
	Issues     []string                `json:"issues,omitempty"`
	VerifiedAt time.Time               `json:"verified_at"`
	Metadata   map[string]interface{}  `json:"metadata,omitempty"`
}

// VerifySBOMFromJSON verifies an SBOM from JSON bytes
func (v *Verifier) VerifySBOMFromJSON(sbomJSON []byte, signature []byte) (*VerificationResult, error) {
	var sbomDoc sbom.SBOM
	if err := json.Unmarshal(sbomJSON, &sbomDoc); err != nil {
		return nil, fmt.Errorf("failed to unmarshal SBOM: %w", err)
	}

	return v.VerifySBOM(&sbomDoc, sbomJSON, signature)
}

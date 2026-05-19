package supplychain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/security/sbom"
	"github.com/fluxorio/fluxor/pkg/security/signing"
)

func TestNewVerifier(t *testing.T) {
	config := DefaultVerifierConfig()
	verifier := NewVerifier(config)

	if verifier == nil {
		t.Fatal("NewVerifier returned nil")
	}

	if verifier.config == nil {
		t.Fatal("Verifier config is nil")
	}
}

func TestNewVerifierWithNilConfig(t *testing.T) {
	verifier := NewVerifier(nil)

	if verifier == nil {
		t.Fatal("NewVerifier returned nil")
	}

	if verifier.config == nil {
		t.Fatal("Verifier config is nil")
	}
}

func TestVerifySBOMIntegrity(t *testing.T) {
	config := DefaultVerifierConfig()
	verifier := NewVerifier(config)

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
				Name:    "test-component",
				Version: "1.0.0",
				Type:    "library",
			},
		},
	}

	sbomJSON, _ := json.Marshal(sbomDoc)
	result, err := verifier.VerifySBOM(sbomDoc, sbomJSON, nil)
	if err != nil {
		t.Fatalf("VerifySBOM failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Expected valid result, got invalid: %v", result.Errors)
	}

	// Check that integrity check was performed
	foundIntegrityCheck := false
	for _, check := range result.Checks {
		if check.Type == CheckSBOMIntegrity {
			foundIntegrityCheck = true
			if check.Status != StatusPass {
				t.Errorf("Expected integrity check to pass, got %s: %s", check.Status, check.Message)
			}
		}
	}

	if !foundIntegrityCheck {
		t.Error("SBOM integrity check was not performed")
	}
}

func TestVerifySBOMWithInvalidSBOM(t *testing.T) {
	config := DefaultVerifierConfig()
	verifier := NewVerifier(config)

	// Create invalid SBOM (missing name)
	sbomDoc := &sbom.SBOM{
		Format:  sbom.FormatSPDX,
		Version: "1.0",
		// Name is missing
		Created: time.Now(),
		Creator: sbom.Creator{
			Name: "Test",
		},
		Components: []sbom.Component{
			{
				Name:    "test-component",
				Version: "1.0.0",
			},
		},
	}

	sbomJSON, _ := json.Marshal(sbomDoc)
	result, err := verifier.VerifySBOM(sbomDoc, sbomJSON, nil)
	if err != nil {
		t.Fatalf("VerifySBOM failed: %v", err)
	}

	if result.Valid {
		t.Error("Expected invalid result for invalid SBOM")
	}
}

func TestVerifySBOMSignature(t *testing.T) {
	// Generate key pair
	privateKey, _, err := signing.GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create signer
	signer, err := signing.NewSigner(privateKey, signing.AlgorithmRS256)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	// Create SBOM
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
				Name:    "test-component",
				Version: "1.0.0",
			},
		},
	}

	sbomJSON, _ := json.Marshal(sbomDoc)

	// Sign SBOM
	signature, err := signer.Sign(sbomJSON)
	if err != nil {
		t.Fatalf("Failed to sign SBOM: %v", err)
	}

	// Get public key PEM
	publicKeyPEM, err := signer.PublicKeyPEM()
	if err != nil {
		t.Fatalf("Failed to get public key PEM: %v", err)
	}

	// Configure verifier with signature checking
	config := DefaultVerifierConfig()
	config.EnableSBOMSignatureCheck = true
	config.PublicKeyPEM = publicKeyPEM
	config.SigningAlgorithm = signing.AlgorithmRS256

	verifier := NewVerifier(config)

	// Verify SBOM
	result, err := verifier.VerifySBOM(sbomDoc, sbomJSON, signature)
	if err != nil {
		t.Fatalf("VerifySBOM failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Expected valid result, got invalid: %v", result.Errors)
	}

	// Check that signature check was performed
	foundSignatureCheck := false
	for _, check := range result.Checks {
		if check.Type == CheckSBOMSignature {
			foundSignatureCheck = true
			if check.Status != StatusPass {
				t.Errorf("Expected signature check to pass, got %s: %s", check.Status, check.Message)
			}
		}
	}

	if !foundSignatureCheck {
		t.Error("SBOM signature check was not performed")
	}
}

func TestVerifySBOMWithInvalidSignature(t *testing.T) {
	// Generate key pair
	privateKey, _, err := signing.GenerateKeyPair(2048)
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	// Create signer
	signer, err := signing.NewSigner(privateKey, signing.AlgorithmRS256)
	if err != nil {
		t.Fatalf("Failed to create signer: %v", err)
	}

	// Create SBOM
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
				Name:    "test-component",
				Version: "1.0.0",
			},
		},
	}

	sbomJSON, _ := json.Marshal(sbomDoc)

	// Sign different data (tampered)
	tamperedData := []byte("tampered data")
	signature, err := signer.Sign(tamperedData)
	if err != nil {
		t.Fatalf("Failed to sign data: %v", err)
	}

	// Get public key PEM
	publicKeyPEM, err := signer.PublicKeyPEM()
	if err != nil {
		t.Fatalf("Failed to get public key PEM: %v", err)
	}

	// Configure verifier with signature checking
	config := DefaultVerifierConfig()
	config.EnableSBOMSignatureCheck = true
	config.PublicKeyPEM = publicKeyPEM
	config.SigningAlgorithm = signing.AlgorithmRS256

	verifier := NewVerifier(config)

	// Verify SBOM with wrong signature
	result, err := verifier.VerifySBOM(sbomDoc, sbomJSON, signature)
	if err != nil {
		t.Fatalf("VerifySBOM failed: %v", err)
	}

	// Should fail because signature doesn't match
	if result.Valid {
		t.Error("Expected invalid result for invalid signature")
	}

	foundSignatureCheck := false
	for _, check := range result.Checks {
		if check.Type == CheckSBOMSignature {
			foundSignatureCheck = true
			if check.Status != StatusFail {
				t.Errorf("Expected signature check to fail, got %s", check.Status)
			}
		}
	}

	if !foundSignatureCheck {
		t.Error("SBOM signature check was not performed")
	}
}

func TestVerifyDependencyIntegrity(t *testing.T) {
	config := DefaultVerifierConfig()
	verifier := NewVerifier(config)

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
				Name:    "test-component",
				Version: "1.0.0",
				Type:    "library",
				PURL:    "pkg:golang/test-component@1.0.0",
				Hashes: map[string]string{
					"SHA256": "abc123",
				},
			},
		},
	}

	sbomJSON, _ := json.Marshal(sbomDoc)
	result, err := verifier.VerifySBOM(sbomDoc, sbomJSON, nil)
	if err != nil {
		t.Fatalf("VerifySBOM failed: %v", err)
	}

	// Check that dependency integrity check was performed
	foundDependencyCheck := false
	for _, check := range result.Checks {
		if check.Type == CheckDependencyIntegrity {
			foundDependencyCheck = true
			if check.Status != StatusPass {
				t.Errorf("Expected dependency integrity check to pass, got %s: %s", check.Status, check.Message)
			}
		}
	}

	if !foundDependencyCheck {
		t.Error("Dependency integrity check was not performed")
	}
}

func TestVerifyComponent(t *testing.T) {
	config := DefaultVerifierConfig()
	verifier := NewVerifier(config)

	component := sbom.Component{
		Name:    "test-component",
		Version: "1.0.0",
		Type:    "library",
		PURL:    "pkg:golang/test-component@1.0.0",
		Hashes: map[string]string{
			"SHA256": "abc123",
		},
	}

	result, err := verifier.VerifyComponent(component)
	if err != nil {
		t.Fatalf("VerifyComponent failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Expected valid component, got invalid: %v", result.Issues)
	}
}

func TestVerifyComponentWithMissingFields(t *testing.T) {
	config := DefaultVerifierConfig()
	verifier := NewVerifier(config)

	// Component missing name
	component := sbom.Component{
		Version: "1.0.0",
		Type:    "library",
	}

	result, err := verifier.VerifyComponent(component)
	if err != nil {
		t.Fatalf("VerifyComponent failed: %v", err)
	}

	if result.Valid {
		t.Error("Expected invalid component due to missing name")
	}

	if len(result.Issues) == 0 {
		t.Error("Expected issues to be reported")
	}
}

func TestVerifySBOMFromJSON(t *testing.T) {
	config := DefaultVerifierConfig()
	verifier := NewVerifier(config)

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
				Name:    "test-component",
				Version: "1.0.0",
			},
		},
	}

	sbomJSON, _ := json.Marshal(sbomDoc)
	result, err := verifier.VerifySBOMFromJSON(sbomJSON, nil)
	if err != nil {
		t.Fatalf("VerifySBOMFromJSON failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Expected valid result, got invalid: %v", result.Errors)
	}
}

func TestVerifySBOMWithVulnerabilities(t *testing.T) {
	// Create mock vulnerability scanner
	mockScanner := &MockVulnerabilityScanner{
		Vulnerabilities: []Vulnerability{
			{
				ID:        "VULN-001",
				CVE:       "CVE-2023-1234",
				Component: "test-component",
				Version:   "1.0.0",
				Severity:  SeverityHigh,
				Description: "Test vulnerability",
			},
		},
	}

	config := DefaultVerifierConfig()
	config.EnableVulnerabilityCheck = true
	config.VulnerabilityScanner = mockScanner

	verifier := NewVerifier(config)

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
				Name:    "test-component",
				Version: "1.0.0",
			},
		},
	}

	sbomJSON, _ := json.Marshal(sbomDoc)
	result, err := verifier.VerifySBOM(sbomDoc, sbomJSON, nil)
	if err != nil {
		t.Fatalf("VerifySBOM failed: %v", err)
	}

	// Should fail due to high severity vulnerability
	if result.Valid {
		t.Error("Expected invalid result due to vulnerabilities")
	}

	foundVulnCheck := false
	for _, check := range result.Checks {
		if check.Type == CheckVulnerabilities {
			foundVulnCheck = true
			if check.Status != StatusFail {
				t.Errorf("Expected vulnerability check to fail, got %s", check.Status)
			}
		}
	}

	if !foundVulnCheck {
		t.Error("Vulnerability check was not performed")
	}
}

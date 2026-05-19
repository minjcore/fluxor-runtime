package vpn

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
)

// TLSConfig wraps TLS configuration for VPN
type TLSConfig struct {
	CertFile    string
	KeyFile     string
	CAFile      string
	MinVersion  uint16
	MaxVersion  uint16
	CipherSuites []uint16
}

// LoadTLSConfig loads TLS configuration from files
func LoadTLSConfig(cfg TLSConfig) (*tls.Config, error) {
	if cfg.CertFile == "" || cfg.KeyFile == "" {
		return nil, fmt.Errorf("certificate and key files are required")
	}

	// Load certificate and key
	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   cfg.MinVersion,
		MaxVersion:   cfg.MaxVersion,
	}

	// Set default min version if not specified
	if tlsConfig.MinVersion == 0 {
		tlsConfig.MinVersion = tls.VersionTLS12
	}

	// Set cipher suites if specified
	if len(cfg.CipherSuites) > 0 {
		tlsConfig.CipherSuites = cfg.CipherSuites
	}

	// Load CA certificate if provided (for client verification)
	if cfg.CAFile != "" {
		caCert, err := ioutil.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig, nil
}

// ValidateTLSFiles validates that TLS certificate files exist
func ValidateTLSFiles(certFile, keyFile, caFile string) error {
	if certFile != "" {
		if _, err := os.Stat(certFile); os.IsNotExist(err) {
			return fmt.Errorf("certificate file not found: %s", certFile)
		}
	}

	if keyFile != "" {
		if _, err := os.Stat(keyFile); os.IsNotExist(err) {
			return fmt.Errorf("key file not found: %s", keyFile)
		}
	}

	if caFile != "" {
		if _, err := os.Stat(caFile); os.IsNotExist(err) {
			return fmt.Errorf("CA certificate file not found: %s", caFile)
		}
	}

	return nil
}

// GetDefaultTLSConfig returns a default TLS configuration
func GetDefaultTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion:               tls.VersionTLS12,
		MaxVersion:               tls.VersionTLS13,
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		},
	}
}

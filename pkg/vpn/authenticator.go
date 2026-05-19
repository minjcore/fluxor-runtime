package vpn

import (
	"fmt"
	"net"
	"sync"
)

// Authenticator implementations

// PasswordAuthenticator authenticates clients using username/password
type PasswordAuthenticator struct {
	mu        sync.RWMutex
	users     map[string]string // username -> password hash
	clientInfo map[string]*ClientInfo // username -> client info
}

// NewPasswordAuthenticator creates a new password authenticator
func NewPasswordAuthenticator() *PasswordAuthenticator {
	auth := &PasswordAuthenticator{
		users:      make(map[string]string),
		clientInfo: make(map[string]*ClientInfo),
	}

	// Add default test user for development
	auth.AddUser("test", "test", &ClientInfo{
		ID:       "test-user",
		Username: "test",
		Metadata: make(map[string]interface{}),
	})

	return auth
}

// Authenticate authenticates a client with username and password
func (a *PasswordAuthenticator) Authenticate(username, password string) (bool, error) {
	if username == "" || password == "" {
		return false, fmt.Errorf("username and password are required")
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	storedPassword, exists := a.users[username]
	if !exists {
		return false, fmt.Errorf("invalid credentials")
	}

	// Simple comparison (in production, use bcrypt or similar)
	if storedPassword != password {
		return false, fmt.Errorf("invalid credentials")
	}

	return true, nil
}

// GetClientInfo retrieves client information after authentication
func (a *PasswordAuthenticator) GetClientInfo(username string) (*ClientInfo, error) {
	if username == "" {
		return nil, fmt.Errorf("username is required")
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	info, exists := a.clientInfo[username]
	if !exists {
		return nil, fmt.Errorf("client info not found for username: %s", username)
	}

	// Return a copy
	infoCopy := &ClientInfo{
		ID:       info.ID,
		Username: info.Username,
		Metadata: make(map[string]interface{}),
	}
	if info.IP != nil {
		infoCopy.IP = make([]byte, len(info.IP))
		copy(infoCopy.IP, info.IP)
	}
	for k, v := range info.Metadata {
		infoCopy.Metadata[k] = v
	}

	return infoCopy, nil
}

// AddUser adds a user to the authenticator
func (a *PasswordAuthenticator) AddUser(username, password string, info *ClientInfo) error {
	if username == "" || password == "" {
		return fmt.Errorf("username and password are required")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.users[username] = password
	if info != nil {
		a.clientInfo[username] = info
	} else {
		a.clientInfo[username] = &ClientInfo{
			ID:       username,
			Username: username,
			Metadata: make(map[string]interface{}),
		}
	}

	return nil
}

// RemoveUser removes a user from the authenticator
func (a *PasswordAuthenticator) RemoveUser(username string) error {
	if username == "" {
		return fmt.Errorf("username is required")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.users, username)
	delete(a.clientInfo, username)

	return nil
}

// CertificateAuthenticator authenticates clients using certificates
type CertificateAuthenticator struct {
	mu        sync.RWMutex
	certificates map[string]*ClientInfo // certificate fingerprint -> client info
}

// NewCertificateAuthenticator creates a new certificate authenticator
func NewCertificateAuthenticator() *CertificateAuthenticator {
	return &CertificateAuthenticator{
		certificates: make(map[string]*ClientInfo),
	}
}

// Authenticate authenticates a client using certificate
// For certificate auth, username should be the certificate CN or fingerprint
func (a *CertificateAuthenticator) Authenticate(username, password string) (bool, error) {
	// Certificate authentication doesn't use password
	// Username is typically the certificate CN or fingerprint
	if username == "" {
		return false, fmt.Errorf("username (certificate CN/fingerprint) is required")
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	// Check if certificate exists
	for fingerprint, info := range a.certificates {
		if fingerprint == username || info.Username == username {
			return true, nil
		}
	}

	return false, fmt.Errorf("certificate not found or invalid")
}

// GetClientInfo retrieves client information from certificate
func (a *CertificateAuthenticator) GetClientInfo(username string) (*ClientInfo, error) {
	// In certificate auth, username might be CN from certificate or fingerprint
	if username == "" {
		return nil, fmt.Errorf("username (certificate CN/fingerprint) is required")
	}

	a.mu.RLock()
	defer a.mu.RUnlock()

	// Try to find by fingerprint first, then by username
	var foundInfo *ClientInfo
	for fingerprint, info := range a.certificates {
		if fingerprint == username || info.Username == username {
			foundInfo = info
			break
		}
	}

	if foundInfo == nil {
		return nil, fmt.Errorf("client info not found for certificate: %s", username)
	}

	// Return a copy
	infoCopy := &ClientInfo{
		ID:       foundInfo.ID,
		Username: foundInfo.Username,
		Metadata: make(map[string]interface{}),
	}
	if foundInfo.IP != nil {
		infoCopy.IP = make(net.IP, len(foundInfo.IP))
		copy(infoCopy.IP, foundInfo.IP)
	}
	for k, v := range foundInfo.Metadata {
		infoCopy.Metadata[k] = v
	}

	return infoCopy, nil
}

// AddCertificate adds a certificate to the authenticator
func (a *CertificateAuthenticator) AddCertificate(fingerprint string, info *ClientInfo) error {
	if fingerprint == "" {
		return fmt.Errorf("fingerprint is required")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.certificates[fingerprint] = info
	return nil
}

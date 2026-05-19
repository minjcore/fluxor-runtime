package vault

import "fmt"

// ErrSecretNotFound is returned when a secret path is not found
type ErrSecretNotFound struct {
	Path string
}

func (e *ErrSecretNotFound) Error() string {
	return fmt.Sprintf("secret not found: %s", e.Path)
}

// IsSecretNotFound checks if an error is ErrSecretNotFound
func IsSecretNotFound(err error) bool {
	_, ok := err.(*ErrSecretNotFound)
	return ok
}

// ErrVaultConfig is returned when the vault configuration is invalid
type ErrVaultConfig struct {
	Field string
	Err   error
}

func (e *ErrVaultConfig) Error() string {
	return fmt.Sprintf("vault config %s: %v", e.Field, e.Err)
}

func (e *ErrVaultConfig) Unwrap() error {
	return e.Err
}

// ErrVaultConnection is returned when the provider cannot connect to Vault
type ErrVaultConnection struct {
	Err error
}

func (e *ErrVaultConnection) Error() string {
	return fmt.Sprintf("vault connection: %v", e.Err)
}

func (e *ErrVaultConnection) Unwrap() error {
	return e.Err
}

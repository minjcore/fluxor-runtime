package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// DecryptSecrets decrypts encrypted/sealed config values
// Currently supports basic secret masking in logs
// Future: Integration with HashiCorp Vault, AWS Secrets Manager
// Fail-fast: Validates inputs before processing
func DecryptSecrets(target interface{}) error {
	// Fail-fast: target cannot be nil
	failfast.NotNil(target, "target")

	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("fail-fast: target must be a pointer to a struct, got %s", val.Kind())
	}
	if val.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("fail-fast: target must be a pointer to a struct, got pointer to %s", val.Elem().Kind())
	}

	return decryptSecretsRecursive(val.Elem())
}

// decryptSecretsRecursive recursively processes secrets in struct fields
func decryptSecretsRecursive(val reflect.Value) error {
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Skip unexported fields
		if !field.CanSet() {
			continue
		}

		// Check for secret tag
		secretTag := fieldType.Tag.Get("secret")
		if secretTag == "true" {
			// Mark as secret (for log masking)
			// In future, could decrypt here if value starts with {encrypted:...}
			if field.Kind() == reflect.String {
				value := field.String()
				// Check if it's an encrypted value
				if strings.HasPrefix(value, "{encrypted:") || strings.HasPrefix(value, "{sealed:") {
					// TODO: Implement decryption
					// For now, just log that it needs decryption
					// In production, integrate with secret management system
				}
			}
		}

		// Handle nested structs
		if field.Kind() == reflect.Struct {
			if err := decryptSecretsRecursive(field); err != nil {
				return err
			}
		} else if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
			if !field.IsNil() {
				if err := decryptSecretsRecursive(field.Elem()); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// MaskSecret masks a secret value for logging
func MaskSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "****" + value[len(value)-4:]
}

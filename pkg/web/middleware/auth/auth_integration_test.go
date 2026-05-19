package auth_test

import (
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/web/middleware/auth"
	"github.com/golang-jwt/jwt/v5"
)

func TestJWTMiddleware(t *testing.T) {
	// Create a test JWT token
	secretKey := "test-secret-key"
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": "123",
		"roles":   []string{"user", "admin"},
		"exp":     time.Now().Add(time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Test JWT middleware configuration
	config := auth.DefaultJWTConfig(secretKey)
	if config.SecretKey != secretKey {
		t.Errorf("SecretKey = %v, want %v", config.SecretKey, secretKey)
	}

	// Test token string is valid
	if tokenString == "" {
		t.Error("Token string should not be empty")
	}
}

func TestRBACMiddleware(t *testing.T) {
	// Test RequireRole
	middleware := auth.RequireRole("admin")
	if middleware == nil {
		t.Error("RequireRole should return middleware")
	}

	// Test RequireAnyRole
	middleware = auth.RequireAnyRole("admin", "user")
	if middleware == nil {
		t.Error("RequireAnyRole should return middleware")
	}

	// Test RequireAllRoles
	middleware = auth.RequireAllRoles("admin", "user")
	if middleware == nil {
		t.Error("RequireAllRoles should return middleware")
	}
}

func TestAPIKeyMiddleware(t *testing.T) {
	// Test API key validator
	validKeys := map[string]map[string]interface{}{
		"test-key": {
			"user_id": "123",
			"roles":   []string{"user"},
		},
	}

	validator := auth.SimpleAPIKeyValidator(validKeys)
	claims, err := validator("test-key")
	if err != nil {
		t.Errorf("Validator should accept valid key: %v", err)
	}
	if claims == nil {
		t.Error("Validator should return claims")
	}

	_, err = validator("invalid-key")
	if err == nil {
		t.Error("Validator should reject invalid key")
	}
}

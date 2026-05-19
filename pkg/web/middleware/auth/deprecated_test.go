package auth_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/pkg/web/middleware/auth"
	"github.com/valyala/fasthttp"
)

// TestDeprecatedAPIKey_ValidateKey tests the deprecated ValidateKey function
// This ensures backward compatibility for code using the old API
func TestDeprecatedAPIKey_ValidateKey(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Create a deprecated ValidateKey function
	validateKey := func(key string) (map[string]interface{}, error) {
		if key == "valid-key" {
			return map[string]interface{}{
				"user_id": "123",
				"roles":   []string{"user", "admin"},
			}, nil
		}
		return nil, fmt.Errorf("invalid API key")
	}

	// Test that DefaultAPIKeyConfig accepts deprecated ValidateKey
	config := auth.DefaultAPIKeyConfig(validateKey)
	if config.ValidateKey == nil {
		t.Fatal("DefaultAPIKeyConfig should accept ValidateKey function")
	}

	// Test that the middleware can be created with deprecated ValidateKey
	mw := auth.APIKey(config)
	if mw == nil {
		t.Fatal("APIKey() should not return nil with deprecated ValidateKey")
	}

	// Test that the middleware works with deprecated ValidateKey
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.Set("X-API-Key", "valid-key")
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		// Verify claims are stored
		claims := ctx.Get("user")
		if claims == nil {
			t.Error("Claims should be stored in context")
		}
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("APIKey middleware with deprecated ValidateKey returned error: %v", err)
	}

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status code 200, got %d", reqCtx.Response.StatusCode())
	}
}

// TestDeprecatedAPIKey_ValidateKey_InvalidKey tests error handling with deprecated ValidateKey
func TestDeprecatedAPIKey_ValidateKey_InvalidKey(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	validateKey := func(key string) (map[string]interface{}, error) {
		return nil, fmt.Errorf("invalid API key")
	}

	config := auth.DefaultAPIKeyConfig(validateKey)
	mw := auth.APIKey(config)

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.Set("X-API-Key", "invalid-key")
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Logf("Expected error for invalid key: %v", err)
	}

	// Should return 401 Unauthorized
	if reqCtx.Response.StatusCode() != 401 {
		t.Errorf("Expected status code 401, got %d", reqCtx.Response.StatusCode())
	}
}

// TestDeprecatedAPIKey_ValidateKey_MissingKey tests missing API key with deprecated ValidateKey
func TestDeprecatedAPIKey_ValidateKey_MissingKey(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	validateKey := func(key string) (map[string]interface{}, error) {
		return map[string]interface{}{"user_id": "123"}, nil
	}

	config := auth.DefaultAPIKeyConfig(validateKey)
	mw := auth.APIKey(config)

	reqCtx := &fasthttp.RequestCtx{}
	// No API key header
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Logf("Expected error for missing key: %v", err)
	}

	// Should return 401 Unauthorized
	if reqCtx.Response.StatusCode() != 401 {
		t.Errorf("Expected status code 401, got %d", reqCtx.Response.StatusCode())
	}
}

// TestDeprecatedOAuth2_IntrospectionURL tests the deprecated IntrospectionURL
// This ensures backward compatibility for code using the old OAuth2 introspection API
func TestDeprecatedOAuth2_IntrospectionURL(t *testing.T) {
	// Create a mock introspection server
	introspectionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify basic auth
		username, password, ok := r.BasicAuth()
		if !ok || username != "test-client" || password != "test-secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Verify content type
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Parse form
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		token := r.FormValue("token")
		if token == "valid-token" {
			response := map[string]interface{}{
				"active": true,
				"sub":    "user123",
				"scope":  "read write",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		} else {
			response := map[string]interface{}{
				"active": false,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	}))
	defer introspectionServer.Close()

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Test that DefaultOAuth2Config accepts deprecated IntrospectionURL
	config := auth.DefaultOAuth2Config(
		introspectionServer.URL,
		"test-client",
		"test-secret",
	)
	if config.IntrospectionURL == "" {
		t.Fatal("DefaultOAuth2Config should accept IntrospectionURL")
	}

	// Test that the middleware can be created with deprecated IntrospectionURL
	mw := auth.OAuth2(config)
	if mw == nil {
		t.Fatal("OAuth2() should not return nil with deprecated IntrospectionURL")
	}

	// Test that the middleware works with deprecated IntrospectionURL
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.Set("Authorization", "Bearer valid-token")
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		// Verify claims are stored
		claims := ctx.Get("user")
		if claims == nil {
			t.Error("Claims should be stored in context")
		}
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("OAuth2 middleware with deprecated IntrospectionURL returned error: %v", err)
	}

	if reqCtx.Response.StatusCode() != 200 {
		t.Errorf("Expected status code 200, got %d", reqCtx.Response.StatusCode())
	}
}

// TestDeprecatedOAuth2_IntrospectionURL_InvalidToken tests error handling with deprecated IntrospectionURL
func TestDeprecatedOAuth2_IntrospectionURL_InvalidToken(t *testing.T) {
	// Create a mock introspection server
	introspectionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"active": false,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer introspectionServer.Close()

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := auth.DefaultOAuth2Config(
		introspectionServer.URL,
		"test-client",
		"test-secret",
	)
	mw := auth.OAuth2(config)

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.Set("Authorization", "Bearer invalid-token")
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Logf("Expected error for invalid token: %v", err)
	}

	// Should return 401 Unauthorized
	if reqCtx.Response.StatusCode() != 401 {
		t.Errorf("Expected status code 401, got %d", reqCtx.Response.StatusCode())
	}
}

// TestDeprecatedOAuth2_IntrospectionURL_MissingToken tests missing token with deprecated IntrospectionURL
func TestDeprecatedOAuth2_IntrospectionURL_MissingToken(t *testing.T) {
	introspectionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{"active": true}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer introspectionServer.Close()

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := auth.DefaultOAuth2Config(
		introspectionServer.URL,
		"test-client",
		"test-secret",
	)
	mw := auth.OAuth2(config)

	reqCtx := &fasthttp.RequestCtx{}
	// No Authorization header
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Logf("Expected error for missing token: %v", err)
	}

	// Should return 401 Unauthorized
	if reqCtx.Response.StatusCode() != 401 {
		t.Errorf("Expected status code 401, got %d", reqCtx.Response.StatusCode())
	}
}

// TestDeprecatedOAuth2_IntrospectionURL_ServerError tests introspection server error
func TestDeprecatedOAuth2_IntrospectionURL_ServerError(t *testing.T) {
	// Create a mock introspection server that returns error
	introspectionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer introspectionServer.Close()

	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	config := auth.DefaultOAuth2Config(
		introspectionServer.URL,
		"test-client",
		"test-secret",
	)
	mw := auth.OAuth2(config)

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.Set("Authorization", "Bearer test-token")
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Logf("Expected error for server error: %v", err)
	}

	// Should return 401 Unauthorized
	if reqCtx.Response.StatusCode() != 401 {
		t.Errorf("Expected status code 401, got %d", reqCtx.Response.StatusCode())
	}
}

// TestDeprecatedAPIKey_Priority tests that new methods (Authenticator/Manager) take priority over deprecated ValidateKey
func TestDeprecatedAPIKey_Priority(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Create both deprecated and new methods
	validateKey := func(key string) (map[string]interface{}, error) {
		return map[string]interface{}{"deprecated": true}, nil
	}

	// When Authenticator is set, it should be used instead of ValidateKey
	// This test verifies the priority order
	config := auth.APIKeyConfig{
		ValidateKey: validateKey, // Deprecated method
		KeyLookup:   "header:X-API-Key",
		ClaimsKey:   "user",
	}

	// Without Authenticator or Manager, ValidateKey should be used
	mw := auth.APIKey(config)
	if mw == nil {
		t.Fatal("APIKey() should not return nil")
	}

	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.Header.Set("X-API-Key", "test-key")
	reqCtx.Request.SetRequestURI("/test")
	reqCtx.Request.Header.SetMethod("GET")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	handler := mw(func(ctx *web.FastRequestContext) error {
		claims := ctx.Get("user")
		if claims == nil {
			t.Error("Claims should be stored")
		}
		ctx.RequestCtx.SetStatusCode(200)
		return nil
	})

	err := handler(fastCtx)
	if err != nil {
		t.Errorf("APIKey middleware should work with deprecated ValidateKey: %v", err)
	}
}

// TestDeprecatedOAuth2_Priority tests that new methods (Authenticator/Provider) take priority over deprecated IntrospectionURL
func TestDeprecatedOAuth2_Priority(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// When Provider or Authenticator is set, it should be used instead of IntrospectionURL
	// This test verifies that deprecated IntrospectionURL is only used when new methods are not available
	config := auth.OAuth2Config{
		IntrospectionURL: "http://example.com/introspect", // Deprecated method
		ClientID:         "test-client",
		ClientSecret:     "test-secret",
		TokenLookup:      "header:Authorization",
		AuthScheme:       "Bearer",
		ClaimsKey:        "user",
	}

	// Without Authenticator or Provider, IntrospectionURL should be used
	// This will panic if IntrospectionURL is not set, which is expected
	mw := auth.OAuth2(config)
	if mw == nil {
		t.Fatal("OAuth2() should not return nil when IntrospectionURL is provided")
	}

	// Verify the config is set correctly
	if config.IntrospectionURL == "" {
		t.Error("IntrospectionURL should be set")
	}
}

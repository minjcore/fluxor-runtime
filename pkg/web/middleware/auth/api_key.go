package auth

import (
	"fmt"
	"strings"

	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn"
	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn/apikey"
)

// APIKeyConfig configures API key authentication
type APIKeyConfig struct {
	// Authenticator is the authn authenticator to use (preferred)
	// If set, this will be used instead of ValidateKey
	Authenticator authn.Authenticator

	// Manager is the API key manager to use (alternative to Authenticator)
	// If set, an authenticator will be created automatically
	Manager *apikey.Manager

	// ValidateKey validates an API key (deprecated: use Authenticator or Manager)
	// This is kept for backward compatibility
	ValidateKey func(key string) (map[string]interface{}, error)

	// KeyLookup is the key lookup pattern (default: "header:X-API-Key")
	// Format: "header:<name>", "query:<name>", "cookie:<name>"
	KeyLookup string

	// ClaimsKey is the key to store claims in request context
	ClaimsKey string

	// PrincipalKey is the key to store Principal in request context (default: ClaimsKey + "_principal")
	// If empty and using Authenticator/Manager, defaults to ClaimsKey + "_principal"
	PrincipalKey string

	// SkipPaths is a list of paths to skip authentication
	SkipPaths []string

	// OnError is called when authentication fails
	OnError func(ctx *web.FastRequestContext, err error) error
}

// DefaultAPIKeyConfig returns a default API key configuration
func DefaultAPIKeyConfig(validateKey func(string) (map[string]interface{}, error)) APIKeyConfig {
	return APIKeyConfig{
		ValidateKey: validateKey,
		KeyLookup:   "header:X-API-Key",
		ClaimsKey:   "user",
		SkipPaths:   []string{},
	}
}

// APIKeyWithManager creates an API key config using a key manager
func APIKeyWithManager(manager *apikey.Manager) APIKeyConfig {
	return APIKeyConfig{
		Manager:    manager,
		KeyLookup:  "header:X-API-Key",
		ClaimsKey:  "user",
		SkipPaths:  []string{},
	}
}

// APIKeyWithAuthenticator creates an API key config using an authenticator
func APIKeyWithAuthenticator(authenticator authn.Authenticator) APIKeyConfig {
	return APIKeyConfig{
		Authenticator: authenticator,
		KeyLookup:     "header:X-API-Key",
		ClaimsKey:     "user",
		SkipPaths:     []string{},
	}
}

// APIKey middleware validates API keys
func APIKey(config APIKeyConfig) web.FastMiddleware {
	// Determine which authenticator to use
	var authenticator authn.Authenticator
	if config.Authenticator != nil {
		authenticator = config.Authenticator
	} else if config.Manager != nil {
		authenticator = apikey.NewAuthenticator(config.Manager)
	} else if config.ValidateKey == nil {
		panic("APIKey: ValidateKey function, Authenticator, or Manager must be provided")
	}

	// Default claims key
	claimsKey := config.ClaimsKey
	if claimsKey == "" {
		claimsKey = "user"
	}

	// Default principal key
	principalKey := config.PrincipalKey
	if principalKey == "" && authenticator != nil {
		principalKey = claimsKey + "_principal"
	}

	// Default key lookup
	keyLookup := config.KeyLookup
	if keyLookup == "" {
		keyLookup = "header:X-API-Key"
	}

	// Parse key lookup
	lookupParts := strings.Split(keyLookup, ":")
	if len(lookupParts) != 2 {
		panic("APIKey: invalid KeyLookup format, expected 'source:name'")
	}
	lookupSource := lookupParts[0]
	lookupName := lookupParts[1]

	// Default error handler
	onError := config.OnError
	if onError == nil {
		onError = func(ctx *web.FastRequestContext, err error) error {
			ctx.RequestCtx.SetStatusCode(401)
			ctx.RequestCtx.SetContentType("application/json")
			if _, werr := ctx.RequestCtx.WriteString(fmt.Sprintf(`{"error":"unauthorized","message":"%s"}`, err.Error())); werr != nil {
				// Best-effort response write; ignore on error.
			}
			return nil
		}
	}

	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			// Check if path should be skipped
			path := string(ctx.Path())
			for _, skipPath := range config.SkipPaths {
				if path == skipPath || strings.HasPrefix(path, skipPath) {
					return next(ctx)
				}
			}

			// Extract API key
			var apiKey string
			switch lookupSource {
			case "header":
				apiKey = string(ctx.RequestCtx.Request.Header.Peek(lookupName))
				if apiKey == "" {
					return onError(ctx, fmt.Errorf("API key header missing"))
				}
			case "query":
				apiKey = ctx.Query(lookupName)
				if apiKey == "" {
					return onError(ctx, fmt.Errorf("API key query parameter missing"))
				}
			case "cookie":
				cookieValue := ctx.RequestCtx.Request.Header.Cookie(lookupName)
				if len(cookieValue) == 0 {
					return onError(ctx, fmt.Errorf("API key cookie missing"))
				}
				apiKey = string(cookieValue)
			default:
				return onError(ctx, fmt.Errorf("unsupported key lookup source: %s", lookupSource))
			}

			// Authenticate using the configured method
			if authenticator != nil {
				// Use new authn package
				principal, err := authenticator.Authenticate(ctx.Context(), apiKey)
				if err != nil {
					return onError(ctx, fmt.Errorf("authentication failed: %w", err))
				}

				// Store principal
				if principalKey != "" {
					ctx.Set(principalKey, principal)
				}

				// Convert principal to claims for backward compatibility
				claims := principalToClaims(principal)
				ctx.Set(claimsKey, claims)
			} else {
				// Use legacy ValidateKey function
				claims, err := config.ValidateKey(apiKey)
				if err != nil {
					return onError(ctx, fmt.Errorf("invalid API key: %w", err))
				}

				// Store claims in context
				ctx.Set(claimsKey, claims)
			}

			return next(ctx)
		}
	}
}

// SimpleAPIKeyValidator creates a simple API key validator from a map
func SimpleAPIKeyValidator(validKeys map[string]map[string]interface{}) func(string) (map[string]interface{}, error) {
	return func(key string) (map[string]interface{}, error) {
		claims, ok := validKeys[key]
		if !ok {
			return nil, fmt.Errorf("invalid API key")
		}
		return claims, nil
	}
}

// principalToClaims converts a Principal to a claims map for backward compatibility
func principalToClaims(p *authn.Principal) map[string]interface{} {
	claims := make(map[string]interface{})
	claims["id"] = p.ID
	claims["type"] = p.Type

	// Copy all attributes
	for k, v := range p.Attributes {
		claims[k] = v
	}

	// Add standard fields
	if p.ExpiresAt != nil {
		claims["exp"] = p.ExpiresAt.Unix()
	}
	claims["authenticated_at"] = p.AuthenticatedAt.Unix()

	return claims
}

// GetPrincipal extracts a Principal from request context
func GetPrincipal(ctx *web.FastRequestContext, key string) (*authn.Principal, bool) {
	p, ok := ctx.Get(key).(*authn.Principal)
	return p, ok
}

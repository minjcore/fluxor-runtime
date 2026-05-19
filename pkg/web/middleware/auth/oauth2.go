package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn"
	"github.com/fluxorio/fluxor/pkg/web/middleware/auth/authn/oidc"
)

// OAuth2Config configures OAuth2/OIDC authentication
type OAuth2Config struct {
	// Authenticator is the authn authenticator to use (preferred)
	// If set, this will be used instead of introspection
	Authenticator authn.Authenticator

	// Provider is the OIDC provider to use (alternative to Authenticator)
	// If set, an authenticator will be created automatically
	Provider *oidc.Provider

	// IntrospectionURL is the OAuth2 token introspection endpoint (deprecated: use Provider)
	IntrospectionURL string

	// ClientID is the OAuth2 client ID
	ClientID string

	// ClientSecret is the OAuth2 client secret
	ClientSecret string

	// TokenLookup is the token lookup pattern (default: "header:Authorization")
	TokenLookup string

	// AuthScheme is the authorization scheme (default: "Bearer")
	AuthScheme string

	// ClaimsKey is the key to store claims in request context
	ClaimsKey string

	// PrincipalKey is the key to store Principal in request context (default: ClaimsKey + "_principal")
	// If empty and using Authenticator/Provider, defaults to ClaimsKey + "_principal"
	PrincipalKey string

	// SkipPaths is a list of paths to skip authentication
	SkipPaths []string

	// OnError is called when authentication fails
	OnError func(ctx *web.FastRequestContext, err error) error

	// HTTPClient is the HTTP client for introspection (optional)
	HTTPClient *http.Client
}

// DefaultOAuth2Config returns a default OAuth2 configuration
func DefaultOAuth2Config(introspectionURL, clientID, clientSecret string) OAuth2Config {
	return OAuth2Config{
		IntrospectionURL: introspectionURL,
		ClientID:         clientID,
		ClientSecret:     clientSecret,
		TokenLookup:      "header:Authorization",
		AuthScheme:       "Bearer",
		ClaimsKey:        "user",
		SkipPaths:        []string{},
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// OAuth2WithProvider creates an OAuth2 config using an OIDC provider
func OAuth2WithProvider(provider *oidc.Provider) OAuth2Config {
	return OAuth2Config{
		Provider:    provider,
		TokenLookup: "header:Authorization",
		AuthScheme:  "Bearer",
		ClaimsKey:   "user",
		SkipPaths:   []string{},
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// OAuth2WithAuthenticator creates an OAuth2 config using an authenticator
func OAuth2WithAuthenticator(authenticator authn.Authenticator) OAuth2Config {
	return OAuth2Config{
		Authenticator: authenticator,
		TokenLookup:   "header:Authorization",
		AuthScheme:    "Bearer",
		ClaimsKey:     "user",
		SkipPaths:     []string{},
		HTTPClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// OAuth2 middleware validates OAuth2 tokens via introspection
func OAuth2(config OAuth2Config) web.FastMiddleware {
	// Determine which authenticator to use
	var authenticator authn.Authenticator
	if config.Authenticator != nil {
		authenticator = config.Authenticator
	} else if config.Provider != nil {
		authenticator = oidc.NewAuthenticator(config.Provider)
	} else {
		// Use legacy introspection
		if config.IntrospectionURL == "" {
			panic("OAuth2: IntrospectionURL, Provider, or Authenticator must be provided")
		}
		if config.ClientID == "" {
			panic("OAuth2: ClientID must be provided")
		}
		if config.ClientSecret == "" {
			panic("OAuth2: ClientSecret must be provided")
		}
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

	// Default token lookup
	tokenLookup := config.TokenLookup
	if tokenLookup == "" {
		tokenLookup = "header:Authorization"
	}

	// Parse token lookup
	lookupParts := strings.Split(tokenLookup, ":")
	if len(lookupParts) != 2 {
		panic("OAuth2: invalid TokenLookup format, expected 'source:name'")
	}
	lookupSource := lookupParts[0]
	lookupName := lookupParts[1]

	// Default auth scheme
	authScheme := config.AuthScheme
	if authScheme == "" {
		authScheme = "Bearer"
	}

	// Default HTTP client
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 5 * time.Second,
		}
	}

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

			// Extract token
			var tokenString string
			switch lookupSource {
			case "header":
				authHeader := string(ctx.RequestCtx.Request.Header.Peek(lookupName))
				if authHeader == "" {
					return onError(ctx, fmt.Errorf("authorization header missing"))
				}
				parts := strings.Split(authHeader, " ")
				if len(parts) != 2 || parts[0] != authScheme {
					return onError(ctx, fmt.Errorf("invalid authorization header format"))
				}
				tokenString = parts[1]
			case "query":
				tokenString = ctx.Query(lookupName)
				if tokenString == "" {
					return onError(ctx, fmt.Errorf("token query parameter missing"))
				}
			case "cookie":
				cookieValue := ctx.RequestCtx.Request.Header.Cookie(lookupName)
				if len(cookieValue) == 0 {
					return onError(ctx, fmt.Errorf("token cookie missing"))
				}
				tokenString = string(cookieValue)
			default:
				return onError(ctx, fmt.Errorf("unsupported token lookup source: %s", lookupSource))
			}

			// Authenticate using the configured method
			if authenticator != nil {
				// Use new authn package
				principal, err := authenticator.Authenticate(ctx.Context(), tokenString)
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
				// Use legacy introspection
				claims, err := introspectToken(httpClient, config.IntrospectionURL, config.ClientID, config.ClientSecret, tokenString)
				if err != nil {
					return onError(ctx, fmt.Errorf("token introspection failed: %w", err))
				}

				// Check if token is active
				active, ok := claims["active"].(bool)
				if !ok || !active {
					return onError(ctx, fmt.Errorf("token is not active"))
				}

				// Store claims in context
				ctx.Set(claimsKey, claims)
			}

			return next(ctx)
		}
	}
}

// introspectToken performs OAuth2 token introspection
func introspectToken(client *http.Client, url, clientID, clientSecret, token string) (map[string]interface{}, error) {
	// Create introspection request
	reqBody := fmt.Sprintf("token=%s&token_type_hint=access_token", token)
	req, err := http.NewRequest("POST", url, bytes.NewBufferString(reqBody))
	if err != nil {
		return nil, err
	}

	// Set headers
	req.SetBasicAuth(clientID, clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("introspection failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var claims map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, fmt.Errorf("failed to decode introspection response: %w", err)
	}

	return claims, nil
}

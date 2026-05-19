package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/golang-jwt/jwt/v5"
)

// JWTConfig configures JWT authentication
type JWTConfig struct {
	// SecretKey is the secret key for signing/verifying tokens
	SecretKey string

	// SecretKeyFunc is a function that returns the secret key (alternative to SecretKey)
	SecretKeyFunc func(token *jwt.Token) (interface{}, error)

	// ValidMethods is the list of accepted JWT signing algorithms (e.g. ["HS256"]).
	// Strongly recommended to set to avoid alg-confusion attacks.
	// Default: ["HS256"] when SecretKey is used.
	ValidMethods []string

	// Issuer requires a matching `iss` claim when set.
	Issuer string

	// Audience requires a matching `aud` claim when set.
	Audience []string

	// Leeway allows small clock skew for exp/nbf/iat validation.
	Leeway time.Duration

	// ClaimsKey is the key to store claims in request context
	ClaimsKey string

	// TokenLookup is the token lookup pattern (default: "header:Authorization")
	// Format: "header:<name>", "query:<name>", "cookie:<name>"
	TokenLookup string

	// AuthScheme is the authorization scheme (default: "Bearer")
	AuthScheme string

	// SkipPaths is a list of paths to skip authentication
	SkipPaths []string

	// OnError is called when authentication fails
	OnError func(ctx *web.FastRequestContext, err error) error
}

// DefaultJWTConfig returns a default JWT configuration
func DefaultJWTConfig(secretKey string) JWTConfig {
	return JWTConfig{
		SecretKey:    secretKey,
		ClaimsKey:    "user",
		TokenLookup:  "header:Authorization",
		AuthScheme:   "Bearer",
		SkipPaths:    []string{},
		ValidMethods: []string{"HS256"},
	}
}

// JWT middleware validates JWT tokens
func JWT(config JWTConfig) web.FastMiddleware {
	if config.SecretKey == "" && config.SecretKeyFunc == nil {
		panic("JWT: SecretKey or SecretKeyFunc must be provided")
	}

	validMethods := config.ValidMethods
	if len(validMethods) == 0 && config.SecretKey != "" {
		validMethods = []string{"HS256"}
	}

	// Default secret key function
	keyFunc := config.SecretKeyFunc
	if keyFunc == nil {
		keyFunc = func(token *jwt.Token) (interface{}, error) {
			// Validate signing method family for HMAC secrets.
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(config.SecretKey), nil
		}
	}

	// Default token lookup
	tokenLookup := config.TokenLookup
	if tokenLookup == "" {
		tokenLookup = "header:Authorization"
	}

	// Parse token lookup
	lookupParts := strings.Split(tokenLookup, ":")
	if len(lookupParts) != 2 {
		panic("JWT: invalid TokenLookup format, expected 'source:name'")
	}
	lookupSource := lookupParts[0]
	lookupName := lookupParts[1]

	// Default auth scheme
	authScheme := config.AuthScheme
	if authScheme == "" {
		authScheme = "Bearer"
	}

	// Default error handler
	onError := config.OnError
	if onError == nil {
		onError = func(ctx *web.FastRequestContext, err error) error {
			ctx.RequestCtx.SetStatusCode(401)
			ctx.RequestCtx.Response.Header.Set("WWW-Authenticate", fmt.Sprintf(`%s realm="fluxor", error="invalid_token"`, authScheme))
			ctx.RequestCtx.SetContentType("application/json")
			// Do not reflect internal errors to the caller by default.
			if _, werr := ctx.RequestCtx.WriteString(`{"error":"unauthorized","message":"invalid or missing token"}`); werr != nil {
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

			// Extract token: prefer Authorization header (API clients), then cookie (browser navigation)
			var tokenString string
			var err error

			authHeader := string(ctx.RequestCtx.Request.Header.Peek("Authorization"))
			if authHeader != "" {
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 && parts[0] == authScheme {
					tokenString = strings.TrimSpace(parts[1])
				}
			}
			if tokenString == "" {
				cookieValue := ctx.RequestCtx.Request.Header.Cookie("token")
				if len(cookieValue) > 0 {
					tokenString = string(cookieValue)
				}
			}
			
			// If still no token, try the configured lookup source
			if tokenString == "" {
				switch lookupSource {
				case "header":
					authHeader := string(ctx.RequestCtx.Request.Header.Peek(lookupName))
					if authHeader == "" {
						return onError(ctx, fmt.Errorf("authorization header missing"))
					}
					// Extract token from "Bearer <token>"
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
			}
			
			if tokenString == "" {
				return onError(ctx, fmt.Errorf("token not found in cookie or authorization header"))
			}

			// Parse and validate token
			options := make([]jwt.ParserOption, 0, 4)
			if len(validMethods) > 0 {
				options = append(options, jwt.WithValidMethods(validMethods))
			}
			if config.Leeway > 0 {
				options = append(options, jwt.WithLeeway(config.Leeway))
			}
			if config.Issuer != "" {
				options = append(options, jwt.WithIssuer(config.Issuer))
			}
			if len(config.Audience) > 0 {
				options = append(options, jwt.WithAudience(config.Audience...))
			}

			token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, keyFunc, options...)
			if err != nil {
				return onError(ctx, fmt.Errorf("invalid token: %w", err))
			}

			if !token.Valid {
				return onError(ctx, fmt.Errorf("token is not valid"))
			}

			// Extract claims
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return onError(ctx, fmt.Errorf("invalid token claims"))
			}

			// Store claims in context
			ctx.Set(config.ClaimsKey, claims)

			return next(ctx)
		}
	}
}

// GetClaims extracts JWT claims from request context
func GetClaims(ctx *web.FastRequestContext, key string) (jwt.MapClaims, error) {
	claims, ok := ctx.Get(key).(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("claims not found in context")
	}
	return claims, nil
}

// GetUserID extracts user ID from JWT claims
func GetUserID(ctx *web.FastRequestContext, key string) (string, error) {
	claims, err := GetClaims(ctx, key)
	if err != nil {
		return "", err
	}

	// Try common claim names
	userID, ok := claims["user_id"].(string)
	if ok {
		return userID, nil
	}

	userID, ok = claims["sub"].(string) // JWT standard "subject"
	if ok {
		return userID, nil
	}

	userID, ok = claims["id"].(string)
	if ok {
		return userID, nil
	}

	return "", fmt.Errorf("user ID not found in claims")
}

// JWTTokenGenerator generates JWT tokens
type JWTTokenGenerator struct {
	secret []byte
}

// NewJWTTokenGenerator creates a new JWT token generator
func NewJWTTokenGenerator(secret []byte) *JWTTokenGenerator {
	return &JWTTokenGenerator{secret: secret}
}

// Generate creates a new JWT token with the given claims and expiration
func (g *JWTTokenGenerator) Generate(claims map[string]interface{}, expiresIn time.Duration) (string, error) {
	if claims == nil {
		claims = make(map[string]interface{})
	}

	// Add standard claims
	now := time.Now()
	claims["iat"] = now.Unix()
	claims["exp"] = now.Add(expiresIn).Unix()

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims(claims))

	// Sign token
	tokenString, err := token.SignedString(g.secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

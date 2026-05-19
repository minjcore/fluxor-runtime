package web

// Context keys for Gin / BaseRequestContext. Middleware should set typed values
// instead of ad-hoc map[string]interface{} claim blobs.
const (
	ContextKeyJWTClaims = "claims"
	ContextKeyFloxID    = "flox_id"
)

// JWTClaims is the preferred contract for authenticated subject identity.
// Middleware: c.Set(ContextKeyJWTClaims, &JWTClaims{...}) or gin: c.Set(ContextKeyJWTClaims, claims).
type JWTClaims struct {
	UserID string `json:"sub"`
}

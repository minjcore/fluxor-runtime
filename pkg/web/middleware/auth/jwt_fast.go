package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"hash"
	"sync"
	"time"
	"unsafe"

	"github.com/fluxorio/fluxor/pkg/web"
)

var (
	ErrJWTMalformed = errors.New("jwt: malformed token")
	ErrJWTExpired   = errors.New("jwt: token expired")
	ErrJWTBadSig    = errors.New("jwt: bad signature")
	ErrJWTBadAlg    = errors.New("jwt: algorithm must be HS256")
)

// FastClaims holds decoded JWT payload fields.
// Typed struct avoids the map allocation that jwt.MapClaims incurs per request.
type FastClaims struct {
	Sub      string `json:"sub"`
	Username string `json:"username"`
	Exp      int64  `json:"exp"`
	Iat      int64  `json:"iat"`
}

// FastJWTVerifier validates HS256 JWT tokens with minimal heap allocations.
// Safe for concurrent use.
type FastJWTVerifier struct {
	key     []byte
	macPool sync.Pool // pools hash.Hash (HMAC-SHA256) instances
}

// NewFastJWTVerifier creates a verifier for the given HMAC secret.
func NewFastJWTVerifier(secret string) *FastJWTVerifier {
	key := []byte(secret)
	v := &FastJWTVerifier{key: key}
	v.macPool.New = func() any { return hmac.New(sha256.New, key) }
	return v
}

// knownHS256Header is base64url({"alg":"HS256","typ":"JWT"}) as produced by golang-jwt.
// Fast-path: when the header matches exactly, skip decode+unmarshal entirely.
const knownHS256Header = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9"

// Verify parses and validates an HS256 JWT. Returns FastClaims on success.
// Allocates at most one []byte for the base64-decoded payload.
func (v *FastJWTVerifier) Verify(token string) (FastClaims, error) {
	// Locate the two separator dots without allocating.
	dot1, dot2 := -1, -1
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			if dot1 < 0 {
				dot1 = i
			} else {
				dot2 = i
				break
			}
		}
	}
	if dot1 < 0 || dot2 < 0 || dot2 == len(token)-1 {
		return FastClaims{}, ErrJWTMalformed
	}

	headerB64  := token[:dot1]
	payloadB64 := token[dot1+1 : dot2]
	sigB64     := token[dot2+1:]

	// Algorithm check — skip decode for the standard golang-jwt HS256 header.
	if headerB64 != knownHS256Header {
		hdrJSON, err := base64.RawURLEncoding.DecodeString(headerB64)
		if err != nil {
			return FastClaims{}, ErrJWTMalformed
		}
		var h struct {
			Alg string `json:"alg"`
		}
		if json.Unmarshal(hdrJSON, &h) != nil || h.Alg != "HS256" {
			return FastClaims{}, ErrJWTBadAlg
		}
	}

	// HMAC-SHA256 the signed part (header.payload).
	// Pool avoids allocating a new Hash per request.
	// unsafe.Slice converts the string to []byte without copying.
	signedPart := token[:dot2]
	mac := v.macPool.Get().(hash.Hash)
	mac.Reset()
	mac.Write(unsafe.Slice(unsafe.StringData(signedPart), len(signedPart)))
	var wantBuf [sha256.Size]byte
	want := mac.Sum(wantBuf[:0]) // appends in-place — no heap alloc
	v.macPool.Put(mac)

	// Decode signature into a stack buffer — SHA256 is always 32 bytes.
	var gotBuf [sha256.Size]byte
	n, err := base64.RawURLEncoding.Decode(gotBuf[:], unsafe.Slice(unsafe.StringData(sigB64), len(sigB64)))
	if err != nil {
		return FastClaims{}, ErrJWTMalformed
	}
	if n != sha256.Size || !hmac.Equal(want, gotBuf[:n]) {
		return FastClaims{}, ErrJWTBadSig
	}

	// Decode payload into a typed struct.
	// One allocation here (DecodeString) — unavoidable without a pooled buffer.
	payloadJSON, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return FastClaims{}, ErrJWTMalformed
	}
	var claims FastClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return FastClaims{}, ErrJWTMalformed
	}

	if claims.Exp > 0 && time.Now().Unix() > claims.Exp {
		return FastClaims{}, ErrJWTExpired
	}

	return claims, nil
}

// FastJWTConfig configures the fast JWT middleware.
type FastJWTConfig struct {
	Secret    string
	ClaimsKey string // context key for FastClaims (default: "user")
	OnError   func(ctx *web.FastRequestContext, err error) error
}

var bearerPrefix = []byte("Bearer ")

// FastJWT returns a middleware that validates HS256 JWT tokens.
// Checks Authorization header first, then falls back to "token" cookie.
// Drop-in replacement for JWT() for HMAC-only deployments.
func FastJWT(cfg FastJWTConfig) web.FastMiddleware {
	if cfg.Secret == "" {
		panic("FastJWT: Secret must be set")
	}
	if cfg.ClaimsKey == "" {
		cfg.ClaimsKey = "user"
	}

	v := NewFastJWTVerifier(cfg.Secret)

	onError := cfg.OnError
	if onError == nil {
		onError = func(ctx *web.FastRequestContext, err error) error {
			ctx.RequestCtx.SetStatusCode(401)
			ctx.RequestCtx.SetContentType("application/json")
			_, _ = ctx.RequestCtx.WriteString(`{"error":"unauthorized","message":"invalid or missing token"}`)
			return nil
		}
	}

	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			// fasthttp.Header.Peek returns a slice into the header buffer — no alloc.
			authHeader := ctx.RequestCtx.Request.Header.Peek("Authorization")

			var token string
			if bytes.HasPrefix(authHeader, bearerPrefix) {
				// Convert slice to string without copying using unsafe.
				// Safe: token is only used within this request's lifetime.
				raw := authHeader[len(bearerPrefix):]
				token = unsafe.String(unsafe.SliceData(raw), len(raw))
			} else {
				// Fallback: raw token in "token" cookie (browser sessions).
				cookieVal := ctx.RequestCtx.Request.Header.Cookie("token")
				if len(cookieVal) == 0 {
					return onError(ctx, ErrJWTMalformed)
				}
				token = unsafe.String(unsafe.SliceData(cookieVal), len(cookieVal))
			}

			claims, err := v.Verify(token)
			if err != nil {
				return onError(ctx, err)
			}
			ctx.Set(cfg.ClaimsKey, claims)
			return next(ctx)
		}
	}
}

// GetFastClaims retrieves FastClaims stored by FastJWT middleware.
func GetFastClaims(ctx *web.FastRequestContext, key string) (FastClaims, bool) {
	if key == "" {
		key = "user"
	}
	c, ok := ctx.Get(key).(FastClaims)
	return c, ok
}

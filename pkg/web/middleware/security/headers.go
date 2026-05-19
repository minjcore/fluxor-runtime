package security

import (
	"fmt"

	"github.com/fluxorio/fluxor/pkg/web"
)

// HeadersConfig configures security headers
type HeadersConfig struct {
	// HSTS (HTTP Strict Transport Security)
	HSTS           bool
	HSTSMaxAge     int // in seconds, default 31536000 (1 year)
	HSTSIncludeSub bool

	// CSP (Content Security Policy)
	CSP string

	// X-Frame-Options
	XFrameOptions string // DENY, SAMEORIGIN, or ALLOW-FROM uri

	// X-Content-Type-Options
	XContentTypeOptions bool // nosniff

	// X-XSS-Protection (legacy; modern browsers ignore it). Leave empty unless required.
	XXSSProtection string

	// Referrer-Policy
	ReferrerPolicy string // no-referrer, no-referrer-when-downgrade, origin, etc.

	// Permissions-Policy (formerly Feature-Policy)
	PermissionsPolicy string

	// X-DNS-Prefetch-Control
	XDNSPrefetchControl bool // "off"

	// X-Permitted-Cross-Domain-Policies
	XPermittedCrossDomainPolicies string // e.g. "none"

	// Cross-Origin-Opener-Policy
	CrossOriginOpenerPolicy string // e.g. "same-origin"

	// Cross-Origin-Resource-Policy
	CrossOriginResourcePolicy string // e.g. "same-origin"

	// Cross-Origin-Embedder-Policy
	CrossOriginEmbedderPolicy string // e.g. "require-corp"

	// Custom headers
	CustomHeaders map[string]string
}

// DefaultHeadersConfig returns a default security headers configuration
func DefaultHeadersConfig() HeadersConfig {
	return HeadersConfig{
		HSTS:                true,
		HSTSMaxAge:          31536000, // 1 year
		HSTSIncludeSub:      true,
		XContentTypeOptions: true,
		// Safe-by-default for APIs. If you serve HTML, configure CSP appropriately.
		CSP:                           "default-src 'none'; frame-ancestors 'none'; base-uri 'none'",
		ReferrerPolicy:                "no-referrer",
		XFrameOptions:                 "DENY",
		XDNSPrefetchControl:           true,
		XPermittedCrossDomainPolicies: "none",
		CrossOriginOpenerPolicy:       "same-origin",
		CrossOriginResourcePolicy:     "same-origin",
		// COEP is intentionally left unset by default because it can break embeddings.
		CustomHeaders: make(map[string]string),
	}
}

// Headers middleware adds security headers to responses
func Headers(config HeadersConfig) web.FastMiddleware {
	return func(next web.FastRequestHandler) web.FastRequestHandler {
		return func(ctx *web.FastRequestContext) error {
			// HSTS
			if config.HSTS {
				hstsValue := "max-age="
				if config.HSTSMaxAge > 0 {
					hstsValue += fmt.Sprintf("%d", config.HSTSMaxAge)
				} else {
					hstsValue += "31536000"
				}
				if config.HSTSIncludeSub {
					hstsValue += "; includeSubDomains"
				}
				ctx.RequestCtx.Response.Header.Set("Strict-Transport-Security", hstsValue)
			}

			// CSP
			if config.CSP != "" {
				ctx.RequestCtx.Response.Header.Set("Content-Security-Policy", config.CSP)
			}

			// X-Frame-Options
			if config.XFrameOptions != "" {
				ctx.RequestCtx.Response.Header.Set("X-Frame-Options", config.XFrameOptions)
			}

			// X-Content-Type-Options
			if config.XContentTypeOptions {
				ctx.RequestCtx.Response.Header.Set("X-Content-Type-Options", "nosniff")
			}

			// X-XSS-Protection
			if config.XXSSProtection != "" {
				ctx.RequestCtx.Response.Header.Set("X-XSS-Protection", config.XXSSProtection)
			}

			// Referrer-Policy
			if config.ReferrerPolicy != "" {
				ctx.RequestCtx.Response.Header.Set("Referrer-Policy", config.ReferrerPolicy)
			}

			// Permissions-Policy
			if config.PermissionsPolicy != "" {
				ctx.RequestCtx.Response.Header.Set("Permissions-Policy", config.PermissionsPolicy)
			}

			// X-DNS-Prefetch-Control
			if config.XDNSPrefetchControl {
				ctx.RequestCtx.Response.Header.Set("X-DNS-Prefetch-Control", "off")
			}

			// X-Permitted-Cross-Domain-Policies
			if config.XPermittedCrossDomainPolicies != "" {
				ctx.RequestCtx.Response.Header.Set("X-Permitted-Cross-Domain-Policies", config.XPermittedCrossDomainPolicies)
			}

			// Cross-origin isolation related headers
			if config.CrossOriginOpenerPolicy != "" {
				ctx.RequestCtx.Response.Header.Set("Cross-Origin-Opener-Policy", config.CrossOriginOpenerPolicy)
			}
			if config.CrossOriginResourcePolicy != "" {
				ctx.RequestCtx.Response.Header.Set("Cross-Origin-Resource-Policy", config.CrossOriginResourcePolicy)
			}
			if config.CrossOriginEmbedderPolicy != "" {
				ctx.RequestCtx.Response.Header.Set("Cross-Origin-Embedder-Policy", config.CrossOriginEmbedderPolicy)
			}

			// Custom headers
			for key, value := range config.CustomHeaders {
				ctx.RequestCtx.Response.Header.Set(key, value)
			}

			return next(ctx)
		}
	}
}

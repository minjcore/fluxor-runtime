package bff

import (
	"log"
	"net/http"
	"net/url"
	"strings"
)

// OriginChecker returns true if the origin is allowed.
type OriginChecker func(origin string) bool

// DefaultOriginChecker allows localhost, 127.0.0.1, and extra origins.
func DefaultOriginChecker(extraOrigins []string) OriginChecker {
	return func(origin string) bool {
		if allowLocalhostOrigin(origin) {
			return true
		}
		for _, o := range extraOrigins {
			if o == origin {
				return true
			}
		}
		return false
	}
}

// GofluxorOriginChecker allows *.gofluxor.com, gofluxor.com, localhost, and extra.
func GofluxorOriginChecker(extraOrigins []string) OriginChecker {
	return func(origin string) bool {
		if allowGofluxorOrigin(origin) {
			return true
		}
		if allowLocalhostOrigin(origin) {
			return true
		}
		for _, o := range extraOrigins {
			if o == origin {
				return true
			}
		}
		return false
	}
}

func allowGofluxorOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	host := u.Hostname()
	host = strings.TrimPrefix(host, ".")
	return host == "gofluxor.com" || strings.HasSuffix(host, ".gofluxor.com")
}

// BaoInsureOriginChecker is deprecated; use GofluxorOriginChecker.
func BaoInsureOriginChecker(extraOrigins []string) OriginChecker {
	return GofluxorOriginChecker(extraOrigins)
}

func allowLocalhostOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Scheme != "http" || u.Host == "" {
		return false
	}
	host := u.Hostname()
	return host == "localhost" || host == "127.0.0.1"
}

// CORSOptions configures CORS middleware.
type CORSOptions struct {
	// Checker determines allowed origins.
	Checker OriginChecker
	// UseRefererFallback: when Origin is empty (e.g. LB strips it), derive from Referer.
	UseRefererFallback bool
	// LogPreflight logs OPTIONS requests (can be noisy at high traffic).
	LogPreflight bool
}

// CORS wraps the handler with CORS headers. Uses checker to allow origin.
func CORS(next http.Handler, opts CORSOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" && opts.UseRefererFallback {
			if ref := r.Header.Get("Referer"); ref != "" {
				if u, err := url.Parse(ref); err == nil && u.Scheme != "" && u.Host != "" {
					origin = u.Scheme + "://" + u.Host
				}
			}
		}
		if origin != "" && opts.Checker(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			if opts.LogPreflight {
				log.Printf("CORS preflight OPTIONS %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

package bff

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

// getClientIP extracts the original client IP from the request.
// Checks X-Forwarded-For (leftmost), X-Real-IP, then RemoteAddr.
func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.Index(xff, ","); idx >= 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	if r.RemoteAddr != "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return r.RemoteAddr
		}
		return host
	}
	return ""
}

// getImmediateClientIP returns the IP of the immediate client (for appending to X-Forwarded-For chain).
func getImmediateClientIP(r *http.Request) string {
	if r.RemoteAddr != "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return r.RemoteAddr
		}
		return host
	}
	return ""
}

// ForwardClientIP sets X-Forwarded-For and X-Real-IP on r so upstream biết IP gọi tới (original client).
// Gọi trong Director trước khi proxy request đi. Mỗi proxy append IP kết nối tới vào chain.
func ForwardClientIP(r *http.Request) {
	orig := getClientIP(r)
	immediate := getImmediateClientIP(r)
	if orig == "" && immediate == "" {
		return
	}
	if orig == "" {
		orig = immediate
	}
	existing := r.Header.Get("X-Forwarded-For")
	if existing != "" {
		toAppend := immediate
		if toAppend == "" {
			toAppend = orig
		}
		r.Header.Set("X-Forwarded-For", existing+", "+toAppend)
	} else {
		r.Header.Set("X-Forwarded-For", orig)
	}
	r.Header.Set("X-Real-IP", orig)
}

// StripCORSFromUpstream removes Access-Control-* headers from upstream response
// so only BFF CORS middleware sets them (avoids duplicate Access-Control-Allow-Origin).
func StripCORSFromUpstream(resp *http.Response) error {
	for k := range resp.Header {
		if strings.HasPrefix(k, "Access-Control-") {
			resp.Header.Del(k)
		}
	}
	return nil
}

// backendHeaderNames — strip bất kể casing (upstream có thể gửi x-served-by hoặc X-Served-By).
var backendHeaderNames = []string{
	"x-served-by", "via", "server", "alt-svc",
	"x-powered-by", "x-backend-server",
}

// StripBackendHeaders removes headers that expose upstream server identity (e.g. X-Served-By, Via, Server).
func StripBackendHeaders(resp *http.Response) error {
	for k := range resp.Header {
		lower := strings.ToLower(k)
		for _, strip := range backendHeaderNames {
			if lower == strip {
				resp.Header.Del(k)
				break
			}
		}
	}
	return nil
}

// StripCORSAndBackendFromUpstream removes CORS and backend-identifying headers so client only sees BFF.
func StripCORSAndBackendFromUpstream(resp *http.Response) error {
	StripCORSFromUpstream(resp)
	return StripBackendHeaders(resp)
}

// StripCORSAndBackendFromUpstreamWithServer strips CORS/backend headers then sets Server to serverName (e.g. "agent-bff").
// Nếu serverName rỗng thì chỉ strip, không set Server.
func StripCORSAndBackendFromUpstreamWithServer(serverName string) func(*http.Response) error {
	return func(resp *http.Response) error {
		StripCORSAndBackendFromUpstream(resp)
		if serverName != "" {
			resp.Header.Set("Server", serverName)
		}
		return nil
	}
}

// NewReverseProxy creates a SingleHostReverseProxy for the given target.
// If modifyResponse is non-nil, it is set (e.g. StripCORSFromUpstream).
// Tự động forward client IP qua X-Forwarded-For, X-Real-IP để IAM/upstream biết IP gọi tới.
func NewReverseProxy(target *url.URL, transport http.RoundTripper, modifyResponse func(*http.Response) error) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Director = func(r *http.Request) {
		r.URL.Scheme = target.Scheme
		r.URL.Host = target.Host
		r.Host = target.Host
		if r.URL.RawPath == "" {
			r.URL.RawPath = r.URL.EscapedPath()
		}
		ForwardClientIP(r)
	}
	if transport != nil {
		proxy.Transport = transport
	}
	if modifyResponse != nil {
		proxy.ModifyResponse = modifyResponse
	}
	return proxy
}

// ResponseLogger wraps http.ResponseWriter to capture status code for logging.
type ResponseLogger struct {
	http.ResponseWriter
	Status int
	Method string
	Path   string
}

// WriteHeader records status and delegates.
func (rw *ResponseLogger) WriteHeader(code int) {
	rw.Status = code
	rw.ResponseWriter.WriteHeader(code)
}

// NewUpstreamProxy creates a reverse proxy that injects an API key header.
// Chuẩn hóa luồng: forward user token (Bearer) sang X-User-Token trước khi ghi đè Authorization bằng API key.
// Response từ upstream được strip CORS + backend headers; nếu serverName != "" thì set Server header (vd. "agent-bff").
func NewUpstreamProxy(target *url.URL, transport http.RoundTripper, apiKey, headerName string, serverName ...string) *httputil.ReverseProxy {
	mod := StripCORSAndBackendFromUpstream
	if len(serverName) > 0 && serverName[0] != "" {
		mod = StripCORSAndBackendFromUpstreamWithServer(serverName[0])
	}
	proxy := NewReverseProxy(target, transport, mod)
	if headerName == "" {
		headerName = "Authorization"
	}
	baseDirector := proxy.Director
	key := apiKey
	header := headerName
	proxy.Director = func(r *http.Request) {
		baseDirector(r)
		// Forward user token (Bearer) sang upstream để xác định context user (trước khi ghi đè Authorization)
		if userAuth := r.Header.Get("Authorization"); userAuth != "" && strings.HasPrefix(userAuth, "Bearer ") {
			r.Header.Set("X-User-Token", userAuth)
		}
		if key != "" {
			r.Header.Set(header, key)
		}
	}
	return proxy
}

// MaskPhone returns last 4 digits for logging (e.g. ***1234).
func MaskPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if len(phone) <= 4 {
		return "****"
	}
	return "***" + phone[len(phone)-4:]
}

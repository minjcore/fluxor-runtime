package bff

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestForwardClientIP(t *testing.T) {
	target, _ := url.Parse("http://upstream.example.com")
	proxy := NewReverseProxy(target, nil, nil)
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if xff := r.Header.Get("X-Forwarded-For"); xff == "" {
			t.Error("X-Forwarded-For should be set by proxy")
		}
		if xri := r.Header.Get("X-Real-IP"); xri == "" {
			t.Error("X-Real-IP should be set by proxy")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()
	backendURL, _ := url.Parse(backend.URL)
	proxy = NewReverseProxy(backendURL, nil, nil)

	req := httptest.NewRequest("GET", "http://bff.example.com/", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("proxy ServeHTTP: got status %d", rec.Code)
	}
}

func TestForwardClientIP_XForwardedFor(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xff := r.Header.Get("X-Forwarded-For")
		// Proxy forwards original client (203.0.113.50) and may append immediate client (192.168.1.1)
		if !strings.HasPrefix(xff, "203.0.113.50") {
			t.Errorf("X-Forwarded-For should contain original client: got %q", xff)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()
	backendURL, _ := url.Parse(backend.URL)
	proxy := NewReverseProxy(backendURL, nil, nil)

	req := httptest.NewRequest("GET", "http://bff/", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.50")
	req.RemoteAddr = "192.168.1.1:45678"
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("proxy: got status %d", rec.Code)
	}
}

func TestStripCORSFromUpstream(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Access-Control-Allow-Origin", "https://evil.com")
	resp.Header.Set("Content-Type", "application/json")
	err := StripCORSFromUpstream(resp)
	if err != nil {
		t.Fatalf("StripCORSFromUpstream: %v", err)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Error("Access-Control-* should be stripped")
	}
	if resp.Header.Get("Content-Type") != "application/json" {
		t.Error("Content-Type should remain")
	}
}

func TestStripBackendHeaders(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("X-Served-By", "upstream")
	resp.Header.Set("Server", "nginx")
	resp.Header.Set("Via", "1.1 cache")
	resp.Header.Set("Content-Type", "application/json")
	err := StripBackendHeaders(resp)
	if err != nil {
		t.Fatalf("StripBackendHeaders: %v", err)
	}
	if resp.Header.Get("X-Served-By") != "" {
		t.Error("X-Served-By should be stripped")
	}
	if resp.Header.Get("Server") != "" {
		t.Error("Server should be stripped")
	}
	if resp.Header.Get("Via") != "" {
		t.Error("Via should be stripped")
	}
	if resp.Header.Get("Content-Type") != "application/json" {
		t.Error("Content-Type should remain")
	}
}

func TestStripCORSAndBackendFromUpstreamWithServer(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Access-Control-Allow-Origin", "x")
	resp.Header.Set("Server", "nginx")
	mod := StripCORSAndBackendFromUpstreamWithServer("agent-bff")
	err := mod(resp)
	if err != nil {
		t.Fatalf("ModifyResponse: %v", err)
	}
	if resp.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS header should be stripped")
	}
	if resp.Header.Get("Server") != "agent-bff" {
		t.Errorf("Server header: got %q", resp.Header.Get("Server"))
	}
}

func TestStripCORSAndBackendFromUpstreamWithServer_EmptyName(t *testing.T) {
	resp := &http.Response{Header: http.Header{}}
	resp.Header.Set("Server", "nginx")
	mod := StripCORSAndBackendFromUpstreamWithServer("")
	err := mod(resp)
	if err != nil {
		t.Fatalf("ModifyResponse: %v", err)
	}
	if resp.Header.Get("Server") != "" {
		t.Error("empty serverName should not set Server header")
	}
}

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"0901234567", "***4567"},
		{" 0901234567 ", "***4567"},
		{"1234", "****"},
		{"12", "****"},
		{"", "****"},
	}
	for _, tt := range tests {
		got := MaskPhone(tt.in)
		if got != tt.want {
			t.Errorf("MaskPhone(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestResponseLogger(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := &ResponseLogger{ResponseWriter: rec, Method: "GET", Path: "/api"}
	rw.WriteHeader(http.StatusCreated)
	if rw.Status != http.StatusCreated {
		t.Errorf("ResponseLogger.Status = %d, want 201", rw.Status)
	}
	if rec.Code != http.StatusCreated {
		t.Errorf("recorder Code = %d, want 201", rec.Code)
	}
}

func TestNewReverseProxy_ServeHTTP(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer backend.Close()
	backendURL, _ := url.Parse(backend.URL)
	proxy := NewReverseProxy(backendURL, nil, nil)

	req := httptest.NewRequest("GET", "http://bff/foo", nil)
	req.RemoteAddr = "127.0.0.1:9999"
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("got status %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body = %q", rec.Body.String())
	}
}

func TestNewReverseProxy_ModifyResponse(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "https://backend.com")
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()
	backendURL, _ := url.Parse(backend.URL)
	proxy := NewReverseProxy(backendURL, nil, StripCORSFromUpstream)

	req := httptest.NewRequest("GET", "http://bff/", nil)
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("got status %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("ModifyResponse should have stripped CORS header")
	}
}

func TestNewUpstreamProxy_InjectsAuth(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Apikey secret" {
			t.Errorf("Authorization: got %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()
	backendURL, _ := url.Parse(backend.URL)
	proxy := NewUpstreamProxy(backendURL, nil, "Apikey secret", "Authorization", "bff")

	req := httptest.NewRequest("GET", "http://bff/", nil)
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("got status %d", rec.Code)
	}
}

func TestNewUpstreamProxy_ForwardsBearerAsXUserToken(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-User-Token") != "Bearer user-jwt" {
			t.Errorf("X-User-Token: got %q", r.Header.Get("X-User-Token"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()
	backendURL, _ := url.Parse(backend.URL)
	proxy := NewUpstreamProxy(backendURL, nil, "Apikey key", "Authorization", "bff")

	req := httptest.NewRequest("GET", "http://bff/", nil)
	req.Header.Set("Authorization", "Bearer user-jwt")
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("got status %d", rec.Code)
	}
}

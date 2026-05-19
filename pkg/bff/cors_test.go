package bff

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultOriginChecker_Localhost(t *testing.T) {
	check := DefaultOriginChecker(nil)
	if !check("http://localhost:3000") {
		t.Error("http://localhost:3000 should be allowed")
	}
	if !check("http://127.0.0.1:8080") {
		t.Error("http://127.0.0.1:8080 should be allowed")
	}
	if check("http://other.example.com") {
		t.Error("other origin should not be allowed without extra")
	}
}

func TestDefaultOriginChecker_ExtraOrigins(t *testing.T) {
	check := DefaultOriginChecker([]string{"https://app.example.com", "https://admin.example.com"})
	if !check("https://app.example.com") {
		t.Error("extra origin should be allowed")
	}
	if !check("https://admin.example.com") {
		t.Error("extra origin should be allowed")
	}
	if check("https://evil.example.com") {
		t.Error("non-extra origin should not be allowed")
	}
}

func TestGofluxorOriginChecker(t *testing.T) {
	check := GofluxorOriginChecker(nil)
	if !check("https://gofluxor.com") {
		t.Error("gofluxor.com should be allowed")
	}
	if !check("https://app.gofluxor.com") {
		t.Error("*.gofluxor.com should be allowed")
	}
	if !check("http://localhost:3000") {
		t.Error("localhost should be allowed")
	}
	if check("https://other.com") {
		t.Error("other.com should not be allowed")
	}
}

func TestCORS_Options(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := CORS(next, CORSOptions{Checker: DefaultOriginChecker(nil)})

	req := httptest.NewRequest("OPTIONS", "http://bff/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("OPTIONS: got status %d, want 204", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("CORS methods header should be set")
	}
	if rec.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("CORS headers should be set")
	}
}

func TestCORS_AllowedOrigin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	handler := CORS(next, CORSOptions{Checker: DefaultOriginChecker(nil)})

	req := httptest.NewRequest("GET", "http://bff/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET: got status %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("Access-Control-Allow-Origin: got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body: got %q", rec.Body.String())
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	handler := CORS(next, CORSOptions{Checker: DefaultOriginChecker(nil)})

	req := httptest.NewRequest("GET", "http://bff/", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("disallowed origin should not get Access-Control-Allow-Origin")
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body: got %q", rec.Body.String())
	}
}

func TestCORS_RefererFallback(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	handler := CORS(next, CORSOptions{
		Checker:            DefaultOriginChecker(nil),
		UseRefererFallback: true,
	})

	req := httptest.NewRequest("GET", "http://bff/", nil)
	req.Header.Set("Referer", "http://localhost:3000/page")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Errorf("Referer fallback: got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

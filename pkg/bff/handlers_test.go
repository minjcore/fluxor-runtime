package bff

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCoreConfig_CheckRole_NoAuth(t *testing.T) {
	iam := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("IAM should not be called when no Authorization")
	}))
	defer iam.Close()
	cfg := &CoreConfig{
		IAMBaseURL: iam.URL,
		HTTPClient: iam.Client(),
		AllowedRole: "agent",
		LogPrefix:   "test",
	}
	req := httptest.NewRequest("GET", "http://bff/api/profile", nil)
	allowed, code, _ := cfg.CheckRole(req)
	if allowed {
		t.Error("allowed should be false when no auth")
	}
	if code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", code)
	}
}

func TestCoreConfig_CheckRole_Forbidden(t *testing.T) {
	iam := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"role_name": "user"})
	}))
	defer iam.Close()
	cfg := &CoreConfig{
		IAMBaseURL:  iam.URL,
		HTTPClient:  iam.Client(),
		AllowedRole: "agent",
		LogPrefix:   "test",
	}
	req := httptest.NewRequest("GET", "http://bff/", nil)
	req.Header.Set("Authorization", "Bearer token")
	allowed, code, _ := cfg.CheckRole(req)
	if allowed {
		t.Error("allowed should be false when role != AllowedRole")
	}
	if code != http.StatusForbidden {
		t.Errorf("status: got %d, want 403", code)
	}
}

func TestCoreConfig_CheckRole_Allowed(t *testing.T) {
	iam := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"role_name": "agent"})
	}))
	defer iam.Close()
	cfg := &CoreConfig{
		IAMBaseURL:  iam.URL,
		HTTPClient:  iam.Client(),
		AllowedRole: "agent",
		LogPrefix:   "test",
	}
	req := httptest.NewRequest("GET", "http://bff/", nil)
	req.Header.Set("Authorization", "Bearer token")
	allowed, code, body := cfg.CheckRole(req)
	if !allowed {
		t.Error("allowed should be true")
	}
	if code != 0 {
		t.Errorf("code should be 0 when allowed, got %d", code)
	}
	if len(body) == 0 {
		t.Error("body should contain profile JSON")
	}
}

func TestCoreConfig_HandleProfile_Unauthorized(t *testing.T) {
	cfg := &CoreConfig{LogPrefix: "test"}
	req := httptest.NewRequest("GET", "http://bff/api/auth/profile", nil)
	rec := httptest.NewRecorder()
	cfg.HandleProfile(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("HandleProfile no auth: got %d", rec.Code)
	}
}

func TestCoreConfig_HandleProfile_Forbidden(t *testing.T) {
	iam := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"role_name": "user"})
	}))
	defer iam.Close()
	cfg := &CoreConfig{
		IAMBaseURL:        iam.URL,
		HTTPClient:        iam.Client(),
		AllowedRole:       "agent",
		LogPrefix:         "test",
		RoleForbiddenMsg:  "Agent only",
	}
	req := httptest.NewRequest("GET", "http://bff/", nil)
	req.Header.Set("Authorization", "Bearer x")
	rec := httptest.NewRecorder()
	cfg.HandleProfile(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("HandleProfile wrong role: got %d", rec.Code)
	}
	var m map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&m)
	if m["message"] != "Agent only" {
		t.Errorf("message: got %q", m["message"])
	}
}

func TestCoreConfig_HandleLogout_NoBody(t *testing.T) {
	cfg := &CoreConfig{LogPrefix: "test"}
	req := httptest.NewRequest("POST", "http://bff/api/auth/logout", nil)
	rec := httptest.NewRecorder()
	cfg.HandleLogout(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("HandleLogout no body: got %d", rec.Code)
	}
	var m map[string]interface{}
	_ = json.NewDecoder(rec.Body).Decode(&m)
	if m["ok"] != true {
		t.Errorf("expected ok:true, got %v", m)
	}
}

func TestCoreConfig_HandleLogout_NoRefreshToken(t *testing.T) {
	cfg := &CoreConfig{LogPrefix: "test"}
	req := httptest.NewRequest("POST", "http://bff/api/auth/logout", nil)
	req.Body = http.NoBody
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	cfg.HandleLogout(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("HandleLogout no refresh_token: got %d", rec.Code)
	}
}

func TestCoreConfig_HandleOTPSignup_InvalidBody(t *testing.T) {
	cfg := &CoreConfig{LogPrefix: "test"}
	req := httptest.NewRequest("POST", "http://bff/api/v1/otp/signup", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = http.NoBody
	rec := httptest.NewRecorder()
	cfg.HandleOTPSignup(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleOTPSignup invalid body: got %d", rec.Code)
	}
}

func TestCoreConfig_HandleOTPSignup_MissingPhone(t *testing.T) {
	cfg := &CoreConfig{LogPrefix: "test"}
	req := httptest.NewRequest("POST", "http://bff/api/v1/otp/signup", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Body = http.NoBody
	rec := httptest.NewRecorder()
	cfg.HandleOTPSignup(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleOTPSignup missing phone: got %d", rec.Code)
	}
}

func TestCoreConfig_HandleOTPSignup_EmptyPhone(t *testing.T) {
	cfg := &CoreConfig{LogPrefix: "test"}
	body := []byte(`{"phone": "   "}`)
	req := httptest.NewRequest("POST", "http://bff/api/v1/otp/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	cfg.HandleOTPSignup(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("HandleOTPSignup empty phone: got %d", rec.Code)
	}
}

func TestCoreConfig_HandleOTPSignup_Success(t *testing.T) {
	iam := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
	}))
	defer iam.Close()
	cfg := &CoreConfig{
		IAMBaseURL: iam.URL,
		HTTPClient: iam.Client(),
		LogPrefix:  "test",
	}
	body := []byte(`{"phone": "0901234567"}`)
	req := httptest.NewRequest("POST", "http://bff/api/v1/otp/signup", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	cfg.HandleOTPSignup(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("HandleOTPSignup success: got %d", rec.Code)
	}
	var m map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&m); err != nil {
		t.Fatalf("response JSON: %v", err)
	}
	if m["otp_session"] != "0901234567" {
		t.Errorf("otp_session: got %v", m["otp_session"])
	}
	if _, ok := m["new_user"]; !ok {
		t.Error("new_user should be present")
	}
}
/*</think>
Fixing the handlers test: adding the missing import and correcting the request body.
<｜tool▁calls▁begin｜><｜tool▁call▁begin｜>
StrReplace
*/
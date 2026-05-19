package bff

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// CheckRole calls IAM GET /api/auth/profile and returns (allowed, statusCode, body).
// allowed is true only when IAM returns 200 and role_name matches AllowedRole.
func (c *CoreConfig) CheckRole(r *http.Request) (allowed bool, statusCode int, body []byte) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return false, http.StatusUnauthorized, nil
	}
	iamReq, err := http.NewRequestWithContext(r.Context(), "GET", c.IAMBaseURL+"/api/auth/profile", nil)
	if err != nil {
		return false, http.StatusInternalServerError, nil
	}
	iamReq.Header.Set("Authorization", auth)
	iamReq.Header.Set("Accept", "application/json")
	resp, err := c.HTTPClient.Do(iamReq)
	if err != nil {
		log.Printf("[%s] CheckRole IAM request: %v", c.LogPrefix, err)
		return false, http.StatusBadGateway, nil
	}
	defer resp.Body.Close()
	body, _ = io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return false, resp.StatusCode, body
	}
	var profile struct {
		RoleName string `json:"role_name"`
	}
	_ = json.Unmarshal(body, &profile)
	if profile.RoleName != c.AllowedRole {
		log.Printf("[%s] CheckRole role=%s required=%s -> 403", c.LogPrefix, profile.RoleName, c.AllowedRole)
		return false, http.StatusForbidden, nil
	}
	return true, 0, body
}

// HandleOAuth2Token proxies POST /oauth2/token to IAM, injecting client_id and client_secret.
func (c *CoreConfig) HandleOAuth2Token(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[%s] oauth2/token read body: %v", c.LogPrefix, err)
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}
	vals, err := url.ParseQuery(string(body))
	if err != nil {
		log.Printf("[%s] oauth2/token parse form: %v", c.LogPrefix, err)
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}
	if c.ClientID != "" {
		vals.Set("client_id", c.ClientID)
		if c.ClientSecret != "" {
			vals.Set("client_secret", c.ClientSecret)
		}
	}
	proxyBody := vals.Encode()
	iamReq, err := http.NewRequestWithContext(r.Context(), "POST", c.IAMBaseURL+"/oauth2/token", strings.NewReader(proxyBody))
	if err != nil {
		log.Printf("[%s] oauth2/token build request: %v", c.LogPrefix, err)
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	iamReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	iamReq.Header.Set("Accept", "application/json")
	if r.Header.Get("Authorization") != "" {
		iamReq.Header.Set("Authorization", r.Header.Get("Authorization"))
	}
	resp, err := c.HTTPClient.Do(iamReq)
	if err != nil {
		log.Printf("[%s] oauth2/token IAM request: %v", c.LogPrefix, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"iam_unavailable"}`))
		return
	}
	defer resp.Body.Close()
	copyResponseHeaders(w, resp)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// HandleProfile proxies GET /api/auth/profile; returns 403 if role != AllowedRole.
func (c *CoreConfig) HandleProfile(w http.ResponseWriter, r *http.Request) {
	allowed, code, body := c.CheckRole(r)
	if !allowed {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		if code == http.StatusForbidden {
			msg := c.RoleForbiddenMsg
			if msg == "" {
				msg = "This BFF is restricted to role " + c.AllowedRole
			}
			b, _ := json.Marshal(map[string]string{"error": "role_required", "message": msg})
			_, _ = w.Write(b)
		} else if len(body) > 0 {
			_, _ = w.Write(body)
		} else {
			_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// HandleLogout proxies POST /api/auth/logout to IAM. If no refresh_token, returns 200 so client can clear state.
func (c *CoreConfig) HandleLogout(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[%s] logout read body: %v", c.LogPrefix, err)
		writeOK(w)
		return
	}
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.Unmarshal(body, &req)
	if req.RefreshToken == "" {
		log.Printf("[%s] logout no refresh_token -> 200 ok (client-side only)", c.LogPrefix)
		writeOK(w)
		return
	}
	iamReq, err := http.NewRequestWithContext(r.Context(), "POST", c.IAMBaseURL+"/api/auth/logout", bytes.NewReader(body))
	if err != nil {
		log.Printf("[%s] logout build request: %v", c.LogPrefix, err)
		writeOK(w)
		return
	}
	iamReq.Header.Set("Content-Type", "application/json")
	iamReq.Header.Set("Accept", "application/json")
	resp, err := c.HTTPClient.Do(iamReq)
	if err != nil {
		log.Printf("[%s] logout IAM request: %v", c.LogPrefix, err)
		writeOK(w)
		return
	}
	defer resp.Body.Close()
	copyResponseHeaders(w, resp)
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// HandleOTPSignup handles POST /api/v1/otp/signup. Body: { "phone": "<number>" }.
// Calls IAM send-otp with purpose=login; on USER_NOT_FOUND/404, retries with purpose=register.
func (c *CoreConfig) HandleOTPSignup(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[%s] otp/signup read body: %v", c.LogPrefix, err)
		http.Error(w, `{"error":"bad request"}`, http.StatusBadRequest)
		return
	}
	var req struct {
		Phone string `json:"phone"`
	}
	if err := json.Unmarshal(body, &req); err != nil || strings.TrimSpace(req.Phone) == "" {
		log.Printf("[%s] otp/signup invalid body or missing phone", c.LogPrefix)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"phone required"}`))
		return
	}
	username := strings.TrimSpace(req.Phone)
	log.Printf("[%s] otp/signup phone=%s -> IAM send-otp purpose=login", c.LogPrefix, MaskPhone(username))

	iamBody, _ := json.Marshal(map[string]string{"username": username, "purpose": "login"})
	iamReq, err := http.NewRequestWithContext(r.Context(), "POST", c.IAMBaseURL+"/api/auth/send-otp", bytes.NewReader(iamBody))
	if err != nil {
		log.Printf("[%s] otp/signup build request: %v", c.LogPrefix, err)
		http.Error(w, `{"error":"internal"}`, http.StatusInternalServerError)
		return
	}
	iamReq.Header.Set("Content-Type", "application/json")
	iamReq.Header.Set("Accept", "application/json")
	resp, err := c.HTTPClient.Do(iamReq)
	if err != nil {
		log.Printf("[%s] otp/signup IAM request failed: %v", c.LogPrefix, err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		detail := err.Error()
		if len(detail) > 200 {
			detail = detail[:200] + "..."
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "iam_unavailable", "detail": detail})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Printf("[%s] otp/signup IAM send-otp phone=%s status=%d", c.LogPrefix, MaskPhone(username), resp.StatusCode)
		if resp.StatusCode == 404 || resp.StatusCode == 401 || resp.StatusCode == 400 {
			var errBody struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&errBody)
			if resp.StatusCode == 404 || errBody.Code == "USER_NOT_FOUND" || strings.Contains(strings.ToLower(errBody.Message), "not found") {
				log.Printf("[%s] otp/signup retry purpose=register phone=%s", c.LogPrefix, MaskPhone(username))
				iamBody2, _ := json.Marshal(map[string]string{"username": username, "purpose": "register"})
				iamReq2, err2 := http.NewRequestWithContext(r.Context(), "POST", c.IAMBaseURL+"/api/auth/send-otp", bytes.NewReader(iamBody2))
				if err2 != nil {
					log.Printf("[%s] otp/signup retry build request: %v", c.LogPrefix, err2)
				} else {
					iamReq2.Header.Set("Content-Type", "application/json")
					resp2, err2 := c.HTTPClient.Do(iamReq2)
					if err2 == nil && resp2.StatusCode == http.StatusOK {
						resp2.Body.Close()
						log.Printf("[%s] otp/signup success phone=%s new_user=true", c.LogPrefix, MaskPhone(username))
						writeOTPSignupResponse(w, username, true)
						return
					}
					if resp2 != nil {
						resp2.Body.Close()
						log.Printf("[%s] otp/signup retry register status=%d", c.LogPrefix, resp2.StatusCode)
					}
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
		return
	}
	log.Printf("[%s] otp/signup success phone=%s new_user=false", c.LogPrefix, MaskPhone(username))
	writeOTPSignupResponse(w, username, false)
}

func writeOTPSignupResponse(w http.ResponseWriter, otpSession string, newUser bool) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"otp_session": otpSession,
		"new_user":    newUser,
	})
}

func writeOK(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func copyResponseHeaders(w http.ResponseWriter, resp *http.Response) {
	for k, v := range resp.Header {
		if k == "Content-Type" || k == "Content-Length" {
			for _, vv := range v {
				w.Header().Add(k, vv)
			}
		}
	}
}

package bff

import "net/http"

// CoreConfig is the minimal config for BFF auth handlers (OAuth2, OTP, profile, logout).
type CoreConfig struct {
	IAMBaseURL       string
	HTTPClient       *http.Client
	ClientID         string
	ClientSecret     string
	AllowedRole      string
	ServiceName      string
	RoleForbiddenMsg string // e.g. "Chỉ tài khoản đại lý mới truy cập được ứng dụng này"
	LogPrefix        string // e.g. "agent-bff" or "bff" for logs
}

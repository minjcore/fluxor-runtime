// Package bff provides shared Backend-for-Frontend logic: reverse proxy, CORS,
// OAuth2 token injection, role check, and auth handlers (OTP signup, profile, logout).
//
// Proxy tự động forward client IP (X-Forwarded-For, X-Real-IP) để IAM/upstream biết IP gọi tới.
//
// Used by agent-bff and identity-saas bff-template to avoid code duplication.
package bff

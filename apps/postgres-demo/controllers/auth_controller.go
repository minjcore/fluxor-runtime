package controllers

import (
	"log"

	"github.com/fluxorio/fluxor/apps/postgres-demo/models"
	"github.com/fluxorio/fluxor/apps/postgres-demo/services"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

// AuthController handles authentication requests
type AuthController struct {
	authService *services.AuthService
}

// NewAuthController creates a new auth controller
func NewAuthController(authService *services.AuthService) *AuthController {
	return &AuthController{
		authService: authService,
	}
}

// Login handles login requests
func (c *AuthController) Login(ctx *web.FastRequestContext) error {
	remoteIP := ctx.RequestCtx.RemoteAddr().String()
	log.Printf("[AUTH_CONTROLLER] Received login request from IP: %s", remoteIP)
	
	var req models.LoginRequest
	if err := ctx.BindJSON(&req); err != nil {
		log.Printf("[AUTH_CONTROLLER] Failed to bind JSON: %v", err)
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_request",
			"message": "Invalid JSON",
		})
	}

	log.Printf("[AUTH_CONTROLLER] Login request parsed - Username: %s, Password length: %d", req.Username, len(req.Password))

	resp, err := c.authService.Login(req)
	if err != nil {
		statusCode := 401
		if authErr, ok := err.(*services.AuthError); ok {
			log.Printf("[AUTH_CONTROLLER] Authentication failed - Code: %s, Message: %s", authErr.Code, authErr.Message)
			if authErr.Code == "TOKEN_ERROR" {
				statusCode = 500
			} else if authErr.Code == "ACCOUNT_INACTIVE" {
				statusCode = 403
			}
		} else {
			log.Printf("[AUTH_CONTROLLER] Authentication failed with unknown error: %v", err)
		}
		return ctx.JSON(statusCode, map[string]interface{}{
			"error":   "authentication_failed",
			"message": err.Error(),
		})
	}

	log.Printf("[AUTH_CONTROLLER] Login successful - Username: %s, Token generated", req.Username)
	
	// Set token in HTTP-only cookie for browser navigation
	// This allows the JWT middleware to read the token from cookies
	var cookie fasthttp.Cookie
	cookie.SetKey("token")
	cookie.SetValue(resp.Token)
	cookie.SetPath("/")
	cookie.SetMaxAge(86400) // 24 hours
	cookie.SetHTTPOnly(true)
	cookie.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	ctx.RequestCtx.Response.Header.SetCookie(&cookie)
	
	return ctx.JSON(200, resp)
}

// Logout handles logout requests
func (c *AuthController) Logout(ctx *web.FastRequestContext) error {
	return ctx.JSON(200, map[string]interface{}{
		"message": "Logged out successfully",
	})
}

// Register handles user registration requests
func (c *AuthController) Register(ctx *web.FastRequestContext) error {
	remoteIP := ctx.RequestCtx.RemoteAddr().String()
	log.Printf("[AUTH_CONTROLLER] Received registration request from IP: %s", remoteIP)
	
	var req services.RegisterRequest
	if err := ctx.BindJSON(&req); err != nil {
		log.Printf("[AUTH_CONTROLLER] Failed to bind JSON: %v", err)
		return ctx.JSON(400, map[string]interface{}{
			"error":   "invalid_request",
			"message": "Invalid JSON",
		})
	}

	log.Printf("[AUTH_CONTROLLER] Registration request parsed - Username: %s, Email: %s, Password length: %d", req.Username, req.Email, len(req.Password))

	resp, err := c.authService.Register(ctx.Context(), req)
	if err != nil {
		statusCode := 400
		if authErr, ok := err.(*services.AuthError); ok {
			log.Printf("[AUTH_CONTROLLER] Registration failed - Code: %s, Message: %s", authErr.Code, authErr.Message)
			switch authErr.Code {
			case "USER_EXISTS":
				statusCode = 409 // Conflict
			case "CONFIG_ERROR", "INTERNAL_ERROR":
				statusCode = 500
			}
		} else {
			log.Printf("[AUTH_CONTROLLER] Registration failed with unknown error: %v", err)
		}
		return ctx.JSON(statusCode, map[string]interface{}{
			"error":   "registration_failed",
			"message": err.Error(),
		})
	}

	log.Printf("[AUTH_CONTROLLER] Registration successful - Username: %s, UserID: %s", req.Username, resp.UserID)
	return ctx.JSON(201, resp)
}

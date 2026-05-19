package zalo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	oauthTokenPath       = "https://oauth.zaloapp.com/v4/oa/access_token"
	messageTemplatePath  = "/message/template"
	templateInfoPath     = "/template/info"
	messageQuotaPath     = "/message/quota"
	// refreshBuffer is the safe time window before token expiry when we trigger refresh.
	// Ensures we never use an expired token (no "lost" window) even if refresh takes a few seconds.
	refreshBuffer = 5 * time.Minute
)

// TokenStore can be implemented by callers to persist/retrieve tokens (e.g. DB or cache).
type TokenStore interface {
	GetAccessToken(ctx context.Context) (accessToken, refreshToken string, expiresAt time.Time, err error)
	StoreTokens(ctx context.Context, accessToken, refreshToken string, expiresIn int) error
}

// Client is the Zalo API client (OAuth + ZNS).
type Client struct {
	config    Config
	httpClient *http.Client
	mu        sync.RWMutex
	tokenStore TokenStore
	// in-memory fallback when no TokenStore is set
	accessToken  string
	refreshToken string
	expiresAt    time.Time
}

// NewClient creates a new Zalo client. Config must be validated before calling.
func NewClient(cfg Config) *Client {
	return &Client{
		config:    cfg,
		httpClient: &http.Client{Timeout: cfg.GetTimeout()},
	}
}

// SetTokenStore sets an optional token store for refresh persistence.
func (c *Client) SetTokenStore(store TokenStore) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tokenStore = store
}

// ExchangeCodeForToken exchanges an OAuth authorization code for access and refresh tokens.
// See apps/fluxor-mail zns_client.exchange_code_for_token / zalo_zns.exchange_code_for_token.
func (c *Client) ExchangeCodeForToken(ctx context.Context, code string) (*TokenResponse, error) {
	form := url.Values{}
	form.Set("code", code)
	form.Set("app_id", c.config.AppID)
	form.Set("grant_type", "authorization_code")
	if c.config.CodeVerifier != "" {
		form.Set("code_verifier", c.config.CodeVerifier)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, oauthTokenPath, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("secret_key", c.config.AppSecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	tr, err := parseTokenResponse(body)
	if err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	if tr.Error != 0 && tr.Message != "" {
		return tr, fmt.Errorf("zalo api error %d: %s", tr.Error, tr.Message)
	}
	if tr.AccessToken == "" {
		return tr, fmt.Errorf("no access_token in response")
	}
	c.mu.Lock()
	c.accessToken = tr.AccessToken
	c.refreshToken = tr.RefreshToken
	if tr.ExpiresIn > 0 {
		c.expiresAt = time.Now().Add(time.Duration(int(tr.ExpiresIn)) * time.Second)
	}
	c.mu.Unlock()
	if c.tokenStore != nil {
		_ = c.tokenStore.StoreTokens(ctx, tr.AccessToken, tr.RefreshToken, int(tr.ExpiresIn))
	}
	return tr, nil
}

// RefreshAccessToken refreshes the access token using the refresh token.
func (c *Client) RefreshAccessToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	refreshToken := c.refreshToken
	c.mu.RUnlock()
	if refreshToken == "" && c.tokenStore != nil {
		_, rt, _, _ := c.tokenStore.GetAccessToken(ctx)
		refreshToken = rt
	}
	if refreshToken == "" {
		return "", fmt.Errorf("refresh token not available")
	}
	form := url.Values{}
	form.Set("app_id", c.config.AppID)
	form.Set("refresh_token", refreshToken)
	form.Set("grant_type", "refresh_token")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, oauthTokenPath, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("secret_key", c.config.AppSecret)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	tr, err := parseTokenResponse(body)
	if err != nil {
		return "", fmt.Errorf("parse refresh response: %w", err)
	}
	// Zalo returns error+message when refresh fails (e.g. invalid/expired refresh token).
	if tr.Error != 0 && tr.Message != "" {
		return "", fmt.Errorf("zalo refresh error %d: %s", tr.Error, tr.Message)
	}
	if tr.AccessToken == "" {
		return "", fmt.Errorf("no access_token in refresh response (body: %s)", string(body))
	}
	c.mu.Lock()
	c.accessToken = tr.AccessToken
	c.refreshToken = tr.RefreshToken
	if tr.ExpiresIn > 0 {
		c.expiresAt = time.Now().Add(time.Duration(int(tr.ExpiresIn)) * time.Second)
	}
	c.mu.Unlock()
	if c.tokenStore != nil {
		_ = c.tokenStore.StoreTokens(ctx, tr.AccessToken, tr.RefreshToken, int(tr.ExpiresIn))
	}
	return tr.AccessToken, nil
}

// parseTokenResponse unmarshals token JSON; Zalo may return expires_in as string or number.
func parseTokenResponse(body []byte) (*TokenResponse, error) {
	var raw struct {
		AccessToken  string          `json:"access_token"`
		RefreshToken string          `json:"refresh_token"`
		ExpiresIn    json.RawMessage `json:"expires_in"`
		Error        int             `json:"error,omitempty"`
		Message      string          `json:"message,omitempty"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	tr := &TokenResponse{
		AccessToken:  raw.AccessToken,
		RefreshToken: raw.RefreshToken,
		Error:        raw.Error,
		Message:      raw.Message,
	}
	if len(raw.ExpiresIn) > 0 {
		if raw.ExpiresIn[0] == '"' {
			var s string
			if err := json.Unmarshal(raw.ExpiresIn, &s); err != nil {
				return nil, err
			}
			if n, err := strconv.Atoi(s); err == nil {
				tr.ExpiresIn = flexInt(n)
			}
		} else {
			var n int
			if err := json.Unmarshal(raw.ExpiresIn, &n); err != nil {
				return nil, err
			}
			tr.ExpiresIn = flexInt(n)
		}
	}
	return tr, nil
}

// getAccessToken returns a valid access token, refreshing if needed.
// GetAccessToken returns the current valid access token, refreshing if needed.
func (c *Client) GetAccessToken(ctx context.Context) (string, error) {
	return c.getAccessToken(ctx)
}

func (c *Client) getAccessToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	tok := c.config.AccessToken
	if tok == "" {
		tok = c.accessToken
	}
	exp := c.expiresAt
	c.mu.RUnlock()
	if tok != "" && exp.IsZero() {
		return tok, nil
	}
	if tok != "" && time.Until(exp) > refreshBuffer {
		return tok, nil
	}
	if c.tokenStore != nil {
		at, rt, expAt, err := c.tokenStore.GetAccessToken(ctx)
		if err == nil && at != "" && time.Until(expAt) > refreshBuffer {
			c.mu.Lock()
			c.accessToken = at
			c.refreshToken = rt
			c.expiresAt = expAt
			c.mu.Unlock()
			return at, nil
		}
	}
	return c.RefreshAccessToken(ctx)
}

// Zalo error codes: -124 = invalid token (use refresh), -216 = expired token.
const (
	zaloErrInvalidToken  = -124
	zaloErrExpiredToken  = -216
)

// SendZNSMessage sends a ZNS templated message. Phone is normalized to 84xxxxxxxxx.
// On -124/-216 (invalid/expired token), automatically refreshes and retries once.
func (c *Client) SendZNSMessage(ctx context.Context, in SendZNSInput) (*SendZNSResult, error) {
	res, err := c.sendZNSMessageWithToken(ctx, in, "")
	if err != nil {
		return &SendZNSResult{Success: false, Error: -1, Message: err.Error()}, nil
	}
	if res.Error == zaloErrInvalidToken || res.Error == zaloErrExpiredToken {
		newToken, refreshErr := c.RefreshAccessToken(ctx)
		if refreshErr != nil {
			return &SendZNSResult{Success: false, Error: res.Error, Message: fmt.Sprintf("%s; refresh failed: %v", res.Message, refreshErr)}, nil
		}
		res, _ = c.sendZNSMessageWithToken(ctx, in, newToken)
	}
	return res, nil
}

func (c *Client) sendZNSMessageWithToken(ctx context.Context, in SendZNSInput, accessToken string) (*SendZNSResult, error) {
	if accessToken == "" {
		var err error
		accessToken, err = c.getAccessToken(ctx)
		if err != nil {
			return &SendZNSResult{Success: false, Error: -1, Message: err.Error()}, err
		}
	}
	phone := NormalizePhone(in.Phone)
	if phone == "" {
		return &SendZNSResult{Success: false, Error: -1, Message: "invalid phone"}, nil
	}
	baseURL := strings.TrimSuffix(c.config.BaseURL, "/")
	reqURL := baseURL + messageTemplatePath
	payload := map[string]interface{}{
		"phone":         phone,
		"template_id":   in.TemplateID,
		"template_data": in.TemplateData,
	}
	if in.TrackingID != "" {
		payload["tracking_id"] = in.TrackingID
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, bytes.NewReader(body))
	if err != nil {
		return &SendZNSResult{Success: false, Error: -1, Message: err.Error()}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("access_token", accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return &SendZNSResult{Success: false, Error: -1, Message: err.Error()}, err
	}
	defer resp.Body.Close()
	resBody, _ := io.ReadAll(resp.Body)
	var apiResp struct {
		Error   int    `json:"error"`
		Message string `json:"message"`
		Data    struct {
			MsgID    string `json:"msg_id"`
			SentTime int64  `json:"sent_time"`
		} `json:"data"`
	}
	_ = json.Unmarshal(resBody, &apiResp)
	if apiResp.Error != 0 {
		return &SendZNSResult{Success: false, Error: apiResp.Error, Message: apiResp.Message}, nil
	}
	return &SendZNSResult{
		MsgID: apiResp.Data.MsgID, SentTime: apiResp.Data.SentTime, Success: true,
	}, nil
}

// GetTemplateInfo returns ZNS template information.
// See apps/fluxor-mail zalo_zns.get_zns_template_info.
func (c *Client) GetTemplateInfo(ctx context.Context, templateID string) (*TemplateInfo, error) {
	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	baseURL := strings.TrimSuffix(c.config.BaseURL, "/")
	reqURL := fmt.Sprintf("%s%s?template_id=%s", baseURL, templateInfoPath, url.QueryEscape(templateID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("access_token", accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	resBody, _ := io.ReadAll(resp.Body)
	var apiResp struct {
		Error int                    `json:"error"`
		Data  map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resBody, &apiResp); err != nil {
		return nil, err
	}
	if apiResp.Error != 0 {
		return nil, fmt.Errorf("zalo api error %d", apiResp.Error)
	}
	ti := &TemplateInfo{Data: apiResp.Data}
	if id, ok := apiResp.Data["template_id"].(string); ok {
		ti.TemplateID = id
	}
	if name, ok := apiResp.Data["name"].(string); ok {
		ti.Name = name
	}
	return ti, nil
}

// GetQuota returns ZNS message quota.
// See apps/fluxor-mail zalo_zns.get_zns_quota.
func (c *Client) GetQuota(ctx context.Context) (*QuotaInfo, error) {
	accessToken, err := c.getAccessToken(ctx)
	if err != nil {
		return nil, err
	}
	baseURL := strings.TrimSuffix(c.config.BaseURL, "/")
	reqURL := baseURL + messageQuotaPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("access_token", accessToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	resBody, _ := io.ReadAll(resp.Body)
	var apiResp struct {
		Error int                    `json:"error"`
		Data  map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resBody, &apiResp); err != nil {
		return nil, err
	}
	if apiResp.Error != 0 {
		return nil, fmt.Errorf("zalo api error %d", apiResp.Error)
	}
	return &QuotaInfo{Data: apiResp.Data}, nil
}

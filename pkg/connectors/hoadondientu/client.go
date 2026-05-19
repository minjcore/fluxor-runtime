package hoadondientu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Client is the Vietnam e-invoice API client interface.
type Client interface {
	Invoices() InvoicesClient
	// EnsureToken ensures a valid Bearer token (refresh if needed).
	EnsureToken(ctx context.Context) error
}

// InvoicesClient provides create, get, list, update, delete, status, download, send email.
type InvoicesClient interface {
	Create(ctx context.Context, params *CreateInvoiceParams) (*Invoice, error)
	Get(ctx context.Context, id string) (*Invoice, error)
	List(ctx context.Context, params *ListInvoicesParams) (*ListInvoicesResult, error)
	Update(ctx context.Context, id string, params *UpdateInvoiceParams) (*Invoice, error)
	Delete(ctx context.Context, id string) error
	GetStatus(ctx context.Context, id string) (InvoiceStatus, error)
	Publish(ctx context.Context, id string) (*Invoice, error)
	DownloadPDF(ctx context.Context, id string) ([]byte, error)
	DownloadXML(ctx context.Context, id string) ([]byte, error)
	SendEmail(ctx context.Context, id string, email string) error
	// ListTemplates returns available invoice templates (mẫu số / ký hiệu).
	ListTemplates(ctx context.Context) ([]InvoiceTemplate, error)
}

type hoadondientuClient struct {
	config   Config
	http     *http.Client
	invoices *invoicesClient
	mu       sync.RWMutex
	token    string
	tokenExp time.Time
}

type invoicesClient struct {
	client *hoadondientuClient
}

// NewClient creates a new Vietnam e-invoice API client.
func NewClient(config Config) (Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	c := &hoadondientuClient{
		config: config,
		http: &http.Client{
			Timeout: config.GetTimeout(),
		},
	}
	c.invoices = &invoicesClient{client: c}
	return c, nil
}

func (c *hoadondientuClient) Invoices() InvoicesClient { return c.invoices }

// EnsureToken sets or refreshes Bearer token. If config has Token, use it; else login with Username/Password.
func (c *hoadondientuClient) EnsureToken(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.token != "" && time.Now().Before(c.tokenExp) {
		return nil
	}

	if c.config.Token != "" {
		c.token = c.config.Token
		c.tokenExp = time.Now().Add(24 * time.Hour)
		return nil
	}

	if c.config.Username != "" && c.config.Password != "" {
		token, exp, err := c.login(ctx)
		if err != nil {
			return err
		}
		c.token = token
		c.tokenExp = exp
		return nil
	}

	return fmt.Errorf("hoadondientu: no token and no username/password")
}

// login calls provider auth endpoint. MISA meInvoice: POST .../api/integration/auth/login (or similar).
func (c *hoadondientuClient) login(ctx context.Context) (token string, exp time.Time, err error) {
	base := strings.TrimSuffix(c.config.BaseURL, "/")
	// MISA path; other providers may use different path - config could add AuthPath later
	authURL := base + "/api/integration/auth/login"
	body := map[string]string{
		"username": c.config.Username,
		"password": c.config.Password,
	}
	raw, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, "POST", authURL, bytes.NewReader(raw))
	if err != nil {
		return "", time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", time.Time{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", time.Time{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("hoadondientu login %d: %s", resp.StatusCode, string(data))
	}
	var tr TokenResponse
	if err := json.Unmarshal(data, &tr); err != nil {
		return "", time.Time{}, err
	}
	if tr.AccessToken == "" {
		return "", time.Time{}, fmt.Errorf("hoadondientu: no accessToken in response")
	}
	exp = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	if tr.ExpiresIn <= 0 {
		exp = time.Now().Add(24 * time.Hour)
	}
	return tr.AccessToken, exp, nil
}

func (c *hoadondientuClient) do(ctx context.Context, method, path string, body interface{}, out interface{}) error {
	if err := c.EnsureToken(ctx); err != nil {
		return err
	}

	base := strings.TrimSuffix(c.config.BaseURL, "/")
	// MISA meInvoice: /api/integration/invoice/...
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasPrefix(path, "/api/") {
		path = "/api/integration/invoice" + path
	}
	reqURL := base + path

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("hoadondientu marshal: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)

	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
		resp, err := c.http.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if out != nil && len(data) > 0 {
				if err := json.Unmarshal(data, out); err != nil {
					return fmt.Errorf("hoadondientu parse: %w", err)
				}
			}
			return nil
		}
		if resp.StatusCode == 401 {
			c.mu.Lock()
			c.token = ""
			c.mu.Unlock()
			lastErr = fmt.Errorf("hoadondientu unauthorized: refresh token and retry")
			continue
		}
		var apiErr APIError
		_ = json.Unmarshal(data, &apiErr)
		if apiErr.Message != "" {
			lastErr = &apiErr
		} else {
			lastErr = fmt.Errorf("hoadondientu %d: %s", resp.StatusCode, string(data))
		}
	}
	return lastErr
}

func (c *hoadondientuClient) doGet(ctx context.Context, path string, params url.Values, out interface{}) error {
	if err := c.EnsureToken(ctx); err != nil {
		return err
	}
	base := strings.TrimSuffix(c.config.BaseURL, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasPrefix(path, "/api/") {
		path = "/api/integration/invoice" + path
	}
	reqURL := base + path
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var apiErr APIError
		_ = json.Unmarshal(data, &apiErr)
		if apiErr.Message != "" {
			return &apiErr
		}
		return fmt.Errorf("hoadondientu %d: %s", resp.StatusCode, string(data))
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

func (c *hoadondientuClient) doBinary(ctx context.Context, method, path string) ([]byte, error) {
	if err := c.EnsureToken(ctx); err != nil {
		return nil, err
	}
	base := strings.TrimSuffix(c.config.BaseURL, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasPrefix(path, "/api/") {
		path = "/api/integration/invoice" + path
	}
	reqURL := base + path
	req, err := http.NewRequestWithContext(ctx, method, reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

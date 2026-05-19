package momo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Client is the MoMo Payment API client interface.
type Client interface {
	Payments() PaymentsClient
}

// PaymentsClient provides create payment and callback verification.
type PaymentsClient interface {
	Create(ctx context.Context, params *CreatePaymentParams) (*CreatePaymentResponse, error)
	VerifyCallback(cb *CallbackPayload) bool
	BuildIpnResponse(cb *CallbackPayload, resultCode int, message string) *IpnResponse
}

type momoClient struct {
	config  Config
	http    *http.Client
	payments *paymentsClient
}

type paymentsClient struct {
	client *momoClient
}

// NewClient creates a new MoMo API client.
func NewClient(config Config) (Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	c := &momoClient{
		config: config,
		http: &http.Client{
			Timeout: config.GetTimeout(),
		},
	}
	c.payments = &paymentsClient{client: c}
	return c, nil
}

func (c *momoClient) Payments() PaymentsClient { return c.payments }

func (c *momoClient) do(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("momo: marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	url := c.config.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("momo: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("momo: request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("momo: read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &MomoError{
			StatusCode: resp.StatusCode,
			Body:       string(data),
		}
	}
	return data, nil
}

// MomoError represents an error from MoMo API.
type MomoError struct {
	StatusCode int
	Body       string
}

func (e *MomoError) Error() string {
	return fmt.Sprintf("momo api error %d: %s", e.StatusCode, e.Body)
}

package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

type stripeClient struct {
	config             Config
	httpClient         *http.Client
	rateLimiter        *rate.Limiter
	customersImpl      *customersClient
	paymentIntentsImpl *paymentIntentsClient
	subscriptionsImpl  *subscriptionsClient
	invoicesImpl       *invoicesClient
	productsImpl       *productsClient
	pricesImpl         *pricesClient
}

func NewClient(config Config) (Client, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	httpClient := &http.Client{
		Timeout: config.GetTimeout(),
	}

	limiter := rate.NewLimiter(rate.Limit(config.RateLimit), 10)

	client := &stripeClient{
		config:      config,
		httpClient:  httpClient,
		rateLimiter: limiter,
	}

	client.customersImpl = &customersClient{client: client}
	client.paymentIntentsImpl = &paymentIntentsClient{client: client}
	client.subscriptionsImpl = &subscriptionsClient{client: client}
	client.invoicesImpl = &invoicesClient{client: client}
	client.productsImpl = &productsClient{client: client}
	client.pricesImpl = &pricesClient{client: client}

	return client, nil
}

func (c *stripeClient) Customers() CustomersClient           { return c.customersImpl }
func (c *stripeClient) PaymentIntents() PaymentIntentsClient { return c.paymentIntentsImpl }
func (c *stripeClient) Subscriptions() SubscriptionsClient   { return c.subscriptionsImpl }
func (c *stripeClient) Invoices() InvoicesClient             { return c.invoicesImpl }
func (c *stripeClient) Products() ProductsClient             { return c.productsImpl }
func (c *stripeClient) Prices() PricesClient                 { return c.pricesImpl }

func (c *stripeClient) doRequest(ctx context.Context, method, path string, params url.Values) ([]byte, error) {
	if err := c.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter error: %w", err)
	}

	var reqBody io.Reader
	if params != nil {
		reqBody = strings.NewReader(params.Encode())
	}

	reqURL := c.config.BaseURL + "/v1" + path

	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(c.config.SecretKey, "")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Stripe-Version", c.config.Version)

	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, nil
		}

		if resp.StatusCode == 429 {
			lastErr = fmt.Errorf("rate limited")
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
			continue
		}

		var apiError APIError
		if err := json.Unmarshal(respBody, &apiError); err == nil && apiError.Error.Message != "" {
			return nil, &StripeError{
				Type:    apiError.Error.Type,
				Code:    apiError.Error.Code,
				Message: apiError.Error.Message,
				Param:   apiError.Error.Param,
			}
		}

		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", c.config.MaxRetries, lastErr)
}

// StripeError represents an error response from Stripe API
type StripeError struct {
	Type    string
	Code    string
	Message string
	Param   string
}

func (e *StripeError) Error() string {
	return fmt.Sprintf("stripe error (%s): %s", e.Type, e.Message)
}

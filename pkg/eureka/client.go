package eureka

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// ClientConfig configures the Eureka client
type ClientConfig struct {
	// Registry server URL (e.g., "http://localhost:8761")
	RegistryURL string

	// Instance to register
	Instance *ServiceInstance

	// Renewal interval (how often to send heartbeat)
	// Default: 30 seconds
	RenewalInterval time.Duration

	// Request timeout
	// Default: 5 seconds
	RequestTimeout time.Duration
}

// DefaultClientConfig returns default client configuration
func DefaultClientConfig(registryURL string, instance *ServiceInstance) *ClientConfig {
	return &ClientConfig{
		RegistryURL:    registryURL,
		Instance:       instance,
		RenewalInterval: 30 * time.Second,
		RequestTimeout:  5 * time.Second,
	}
}

// Client is a Eureka client for service registration and discovery
type Client struct {
	config         *ClientConfig
	httpClient     *http.Client
	registered     bool
	renewalStop    chan struct{}
	renewalStarted bool
	mu             sync.RWMutex
}

// NewClient creates a new Eureka client
func NewClient(config *ClientConfig) *Client {
	failfast.NotNil(config, "config")
	failfast.If(config.RegistryURL != "", "registryURL cannot be empty")
	failfast.NotNil(config.Instance, "instance")

	if config.RenewalInterval <= 0 {
		config.RenewalInterval = 30 * time.Second
	}
	if config.RequestTimeout <= 0 {
		config.RequestTimeout = 5 * time.Second
	}

	// Generate instance ID if not provided
	if config.Instance.InstanceID == "" {
		config.Instance.InstanceID = fmt.Sprintf("%s:%s:%d",
			config.Instance.ServiceName,
			config.Instance.Host,
			config.Instance.Port)
	}

	return &Client{
		config:      config,
		httpClient:  &http.Client{Timeout: config.RequestTimeout},
		renewalStop: make(chan struct{}),
	}
}

// Register registers the service instance with the registry
func (c *Client) Register(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.registered {
		return fmt.Errorf("instance already registered")
	}

	url := fmt.Sprintf("%s/eureka/apps/%s", c.config.RegistryURL, c.config.Instance.ServiceName)

	reqBody := RegistrationRequest{
		Instance: c.config.Instance,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to register: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 && resp.StatusCode != 200 {
		return fmt.Errorf("registration failed with status %d", resp.StatusCode)
	}

	c.registered = true
	return nil
}

// Unregister unregisters the service instance
func (c *Client) Unregister(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.registered {
		return nil // Already unregistered
	}

	url := fmt.Sprintf("%s/eureka/apps/%s/%s",
		c.config.RegistryURL,
		c.config.Instance.ServiceName,
		c.config.Instance.InstanceID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to unregister: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 404 {
		return fmt.Errorf("unregistration failed with status %d", resp.StatusCode)
	}

	c.registered = false
	return nil
}

// StartHeartbeat starts the heartbeat/renewal process
func (c *Client) StartHeartbeat(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.renewalStarted {
		return fmt.Errorf("heartbeat already started")
	}

	c.renewalStarted = true

	// Start renewal goroutine
	go func() {
		ticker := time.NewTicker(c.config.RenewalInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := c.renew(ctx); err != nil {
					// Log error but continue renewing
					// In production, you might want to retry or handle differently
					fmt.Printf("Failed to renew lease: %v\n", err)
				}
			case <-c.renewalStop:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// StopHeartbeat stops the heartbeat process
func (c *Client) StopHeartbeat() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.renewalStarted {
		return
	}

	close(c.renewalStop)
	c.renewalStop = make(chan struct{})
	c.renewalStarted = false
}

// renew renews the lease (heartbeat)
func (c *Client) renew(ctx context.Context) error {
	c.mu.RLock()
	instanceID := c.config.Instance.InstanceID
	serviceName := c.config.Instance.ServiceName
	c.mu.RUnlock()

	url := fmt.Sprintf("%s/eureka/apps/%s/%s",
		c.config.RegistryURL,
		serviceName,
		instanceID)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to renew: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("renewal failed with status %d", resp.StatusCode)
	}

	return nil
}

// UpdateStatus updates the instance status
func (c *Client) UpdateStatus(ctx context.Context, status InstanceStatus) error {
	c.mu.RLock()
	instanceID := c.config.Instance.InstanceID
	serviceName := c.config.Instance.ServiceName
	c.mu.RUnlock()

	url := fmt.Sprintf("%s/eureka/apps/%s/%s/status?value=%s",
		c.config.RegistryURL,
		serviceName,
		instanceID,
		status)

	req, err := http.NewRequestWithContext(ctx, "PUT", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("status update failed with status %d", resp.StatusCode)
	}

	// Update local status
	c.mu.Lock()
	c.config.Instance.UpdateStatus(status)
	c.mu.Unlock()

	return nil
}

// Discover discovers instances of a service
func (c *Client) Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	failfast.If(serviceName != "", "serviceName cannot be empty")

	url := fmt.Sprintf("%s/eureka/apps/%s", c.config.RegistryURL, serviceName)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to discover: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("discovery failed with status %d", resp.StatusCode)
	}

	var discoveryResp DiscoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&discoveryResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return discoveryResp.Instances, nil
}

// IsRegistered returns true if the instance is registered
func (c *Client) IsRegistered() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.registered
}

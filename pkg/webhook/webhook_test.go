package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/fluxorio/fluxor/pkg/core"
	"github.com/fluxorio/fluxor/pkg/web"
	"github.com/valyala/fasthttp"
)

func TestHMACSignatureValidator_Validate(t *testing.T) {
	secret := "test-secret"
	payload := []byte("test-payload")

	// Compute expected signature
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	tests := []struct {
		name      string
		validator *HMACSignatureValidator
		payload   []byte
		signature string
		headers   map[string]string
		wantErr   bool
	}{
		{
			name:      "valid signature",
			validator: NewHMACSignatureValidator(secret, SignatureAlgorithmHMACSHA256, "X-Signature", "sha256="),
			payload:   payload,
			signature: "sha256=" + expectedSignature,
			headers:   map[string]string{"X-Signature": "sha256=" + expectedSignature},
			wantErr:   false,
		},
		{
			name:      "invalid signature",
			validator: NewHMACSignatureValidator(secret, SignatureAlgorithmHMACSHA256, "X-Signature", "sha256="),
			payload:   payload,
			signature: "sha256=invalid",
			headers:   map[string]string{"X-Signature": "sha256=invalid"},
			wantErr:   true,
		},
		{
			name:      "missing header",
			validator: NewHMACSignatureValidator(secret, SignatureAlgorithmHMACSHA256, "X-Signature", "sha256="),
			payload:   payload,
			signature: "",
			headers:   map[string]string{},
			wantErr:   true,
		},
		{
			name:      "signature without prefix",
			validator: NewHMACSignatureValidator(secret, SignatureAlgorithmHMACSHA256, "X-Signature", ""),
			payload:   payload,
			signature: expectedSignature,
			headers:   map[string]string{"X-Signature": expectedSignature},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.validator.Validate(tt.payload, tt.signature, tt.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("HMACSignatureValidator.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHMACSignatureValidator_SHA1(t *testing.T) {
	secret := "test-secret"
	payload := []byte("test-payload")

	// Compute expected signature (HMAC-SHA1)
	h := hmac.New(sha1.New, []byte(secret))
	h.Write(payload)
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	validator := NewHMACSignatureValidator(secret, SignatureAlgorithmHMACSHA1, "X-Signature", "sha1=")
	headers := map[string]string{
		"X-Signature": "sha1=" + expectedSignature,
	}

	err := validator.Validate(payload, "", headers)
	if err != nil {
		t.Errorf("HMACSignatureValidator SHA1.Validate() error = %v", err)
	}
}

func TestHMACSignatureValidator_SHA512(t *testing.T) {
	secret := "test-secret"
	payload := []byte("test-payload")

	// Compute expected signature (HMAC-SHA512)
	h := hmac.New(sha512.New, []byte(secret))
	h.Write(payload)
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	validator := NewHMACSignatureValidator(secret, SignatureAlgorithmHMACSHA512, "X-Signature", "sha512=")
	headers := map[string]string{
		"X-Signature": "sha512=" + expectedSignature,
	}

	err := validator.Validate(payload, "", headers)
	if err != nil {
		t.Errorf("HMACSignatureValidator SHA512.Validate() error = %v", err)
	}
}

func TestHMACSignatureValidator_UnsupportedAlgorithm(t *testing.T) {
	secret := "test-secret"
	payload := []byte("test-payload")

	validator := &HMACSignatureValidator{
		secret:     []byte(secret),
		algorithm:  SignatureAlgorithm("unsupported"),
		headerName: "X-Signature",
		prefix:     "",
	}

	err := validator.Validate(payload, "signature", map[string]string{})
	if err == nil {
		t.Error("HMACSignatureValidator.Validate() expected error for unsupported algorithm")
	}
}

func TestGitHubSignatureValidator(t *testing.T) {
	secret := "test-secret"
	payload := []byte("test-payload")

	// Compute expected signature (HMAC-SHA256)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	validator := NewGitHubSignatureValidator(secret)
	headers := map[string]string{
		"X-Hub-Signature-256": "sha256=" + expectedSignature,
	}

	err := validator.Validate(payload, "", headers)
	if err != nil {
		t.Errorf("GitHubSignatureValidator.Validate() error = %v", err)
	}
}

func TestStripeSignatureValidator(t *testing.T) {
	secret := "test-secret"
	payload := []byte("test-payload")

	// Compute expected signature (HMAC-SHA256)
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	expectedSignature := hex.EncodeToString(h.Sum(nil))

	validator := NewStripeSignatureValidator(secret)
	headers := map[string]string{
		"Stripe-Signature": expectedSignature,
	}

	err := validator.Validate(payload, "", headers)
	if err != nil {
		t.Errorf("StripeSignatureValidator.Validate() error = %v", err)
	}
}

func TestEndpointConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  EndpointConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: EndpointConfig{
				Path:            "/github",
				Secret:          "test-secret",
				EventBusAddress: "webhook.github",
			},
			wantErr: false,
		},
		{
			name: "missing path",
			config: EndpointConfig{
				Secret:          "test-secret",
				EventBusAddress: "webhook.github",
			},
			wantErr: true,
		},
		{
			name: "missing secret",
			config: EndpointConfig{
				Path:            "/github",
				EventBusAddress: "webhook.github",
			},
			wantErr: true,
		},
		{
			name: "missing eventBusAddress",
			config: EndpointConfig{
				Path:   "/github",
				Secret: "test-secret",
			},
			wantErr: true,
		},
		{
			name: "skip validation",
			config: EndpointConfig{
				Path:            "/github",
				EventBusAddress: "webhook.github",
				SkipValidation: true,
			},
			wantErr: false,
		},
		{
			name: "custom validator",
			config: EndpointConfig{
				Path:            "/github",
				EventBusAddress: "webhook.github",
				CustomValidator: NewGitHubSignatureValidator("test"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("EndpointConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEndpointConfig_SetDefaults(t *testing.T) {
	config := EndpointConfig{
		Path:            "/test",
		Secret:          "secret",
		EventBusAddress: "webhook.test",
	}

	config.SetDefaults()

	if config.Algorithm != SignatureAlgorithmHMACSHA256 {
		t.Errorf("SetDefaults() Algorithm = %v, want %v", config.Algorithm, SignatureAlgorithmHMACSHA256)
	}

	if config.SignatureHeader != "X-Webhook-Signature" {
		t.Errorf("SetDefaults() SignatureHeader = %v, want X-Webhook-Signature", config.SignatureHeader)
	}

	if config.SignaturePrefix != "sha256=" {
		t.Errorf("SetDefaults() SignaturePrefix = %v, want sha256=", config.SignaturePrefix)
	}
}

func TestEndpointConfig_GetValidator(t *testing.T) {
	tests := []struct {
		name     string
		config   EndpointConfig
		wantNil  bool
		wantType string
	}{
		{
			name: "custom validator",
			config: EndpointConfig{
				CustomValidator: NewGitHubSignatureValidator("secret"),
			},
			wantNil:  false,
			wantType: "HMACSignatureValidator",
		},
		{
			name: "skip validation",
			config: EndpointConfig{
				SkipValidation: true,
			},
			wantNil: true,
		},
		{
			name: "default validator",
			config: EndpointConfig{
				Secret: "test-secret",
			},
			wantNil:  false,
			wantType: "HMACSignatureValidator",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := tt.config.GetValidator()
			if (validator == nil) != tt.wantNil {
				t.Errorf("GetValidator() validator = %v, wantNil %v", validator, tt.wantNil)
			}
		})
	}
}

func TestReceiverConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *ReceiverConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &ReceiverConfig{
				Prefix: "/webhooks",
				Endpoints: []EndpointConfig{
					{
						Path:            "/github",
						Secret:          "secret",
						EventBusAddress: "webhook.github",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing prefix",
			config: &ReceiverConfig{
				Endpoints: []EndpointConfig{
					{
						Path:            "/github",
						Secret:          "secret",
						EventBusAddress: "webhook.github",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no endpoints",
			config: &ReceiverConfig{
				Prefix:    "/webhooks",
				Endpoints: []EndpointConfig{},
			},
			wantErr: true,
		},
		{
			name: "invalid endpoint",
			config: &ReceiverConfig{
				Prefix: "/webhooks",
				Endpoints: []EndpointConfig{
					{
						Path: "", // Invalid: missing path
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ReceiverConfig.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReceiver_Start_Stop(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := core.NewFluxorContext(ctx, gocmd)

	config := &ReceiverConfig{
		Prefix: "/webhooks",
		Endpoints: []EndpointConfig{
			{
				Path:            "/github",
				Secret:          "test-secret",
				EventBusAddress: "webhook.github",
			},
		},
	}

	receiver := NewReceiver(config)

	// Test Start
	if err := receiver.Start(fluxorCtx); err != nil {
		t.Fatalf("Receiver.Start() error = %v", err)
	}

	if !receiver.IsStarted() {
		t.Error("Receiver.IsStarted() should return true after Start()")
	}

	// Test Stop
	if err := receiver.Stop(fluxorCtx); err != nil {
		t.Fatalf("Receiver.Stop() error = %v", err)
	}

	if receiver.IsStarted() {
		t.Error("Receiver.IsStarted() should return false after Stop()")
	}
}

func TestReceiver_Start_InvalidConfig(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := core.NewFluxorContext(ctx, gocmd)

	config := &ReceiverConfig{
		Prefix:    "", // Invalid: empty prefix
		Endpoints: []EndpointConfig{},
	}

	receiver := NewReceiver(config)

	err := receiver.Start(fluxorCtx)
	if err == nil {
		t.Error("Receiver.Start() expected error for invalid config")
	}
}

func TestReceiver_HandleRequest(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := core.NewFluxorContext(ctx, gocmd)

	config := &ReceiverConfig{
		Prefix: "/webhooks",
		Endpoints: []EndpointConfig{
			{
				Path:            "/github",
				Secret:          "test-secret",
				EventBusAddress: "webhook.github",
			},
		},
	}

	receiver := NewReceiver(config)
	if err := receiver.Start(fluxorCtx); err != nil {
		t.Fatalf("Receiver.Start() error = %v", err)
	}
	defer receiver.Stop(fluxorCtx)

	// Create webhook request with valid signature
	payload := []byte("test-payload")
	h := hmac.New(sha256.New, []byte("test-secret"))
	h.Write(payload)
	signature := hex.EncodeToString(h.Sum(nil))

	req := &WebhookRequest{
		Path:    "/webhooks/github",
		Payload: payload,
		Headers: map[string]string{
			"X-Webhook-Signature": "sha256=" + signature,
		},
		QueryParams: make(map[string]string),
		Method:      "POST",
	}

	// This should succeed (publish to EventBus)
	err := receiver.HandleRequest(req)
	if err != nil {
		t.Errorf("Receiver.HandleRequest() error = %v", err)
	}
}

func TestReceiver_HandleRequest_InvalidSignature(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := core.NewFluxorContext(ctx, gocmd)

	config := &ReceiverConfig{
		Prefix: "/webhooks",
		Endpoints: []EndpointConfig{
			{
				Path:            "/github",
				Secret:          "test-secret",
				EventBusAddress: "webhook.github",
			},
		},
	}

	receiver := NewReceiver(config)
	if err := receiver.Start(fluxorCtx); err != nil {
		t.Fatalf("Receiver.Start() error = %v", err)
	}
	defer receiver.Stop(fluxorCtx)

	req := &WebhookRequest{
		Path:    "/webhooks/github",
		Payload: []byte("test-payload"),
		Headers: map[string]string{
			"X-Webhook-Signature": "sha256=invalid",
		},
		QueryParams: make(map[string]string),
		Method:      "POST",
	}

	// This should fail due to invalid signature
	err := receiver.HandleRequest(req)
	if err == nil {
		t.Error("Receiver.HandleRequest() expected error for invalid signature")
	}
}

func TestReceiver_HandleRequest_NotFound(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := core.NewFluxorContext(ctx, gocmd)

	config := &ReceiverConfig{
		Prefix: "/webhooks",
		Endpoints: []EndpointConfig{
			{
				Path:            "/github",
				Secret:          "test-secret",
				EventBusAddress: "webhook.github",
			},
		},
	}

	receiver := NewReceiver(config)
	if err := receiver.Start(fluxorCtx); err != nil {
		t.Fatalf("Receiver.Start() error = %v", err)
	}
	defer receiver.Stop(fluxorCtx)

	req := &WebhookRequest{
		Path:        "/webhooks/unknown",
		Payload:     []byte("test-payload"),
		Headers:     make(map[string]string),
		QueryParams: make(map[string]string),
		Method:      "POST",
	}

	err := receiver.HandleRequest(req)
	if err == nil {
		t.Error("Receiver.HandleRequest() expected error for unknown endpoint")
	}
}

func TestReceiver_HandleRequest_SkipValidation(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := core.NewFluxorContext(ctx, gocmd)

	config := &ReceiverConfig{
		Prefix: "/webhooks",
		Endpoints: []EndpointConfig{
			{
				Path:            "/insecure",
				EventBusAddress: "webhook.insecure",
				SkipValidation: true,
			},
		},
	}

	receiver := NewReceiver(config)
	if err := receiver.Start(fluxorCtx); err != nil {
		t.Fatalf("Receiver.Start() error = %v", err)
	}
	defer receiver.Stop(fluxorCtx)

	req := &WebhookRequest{
		Path:        "/webhooks/insecure",
		Payload:     []byte("test-payload"),
		Headers:     make(map[string]string),
		QueryParams: make(map[string]string),
		Method:      "POST",
	}

	// Should succeed without signature validation
	err := receiver.HandleRequest(req)
	if err != nil {
		t.Errorf("Receiver.HandleRequest() error = %v", err)
	}
}

func TestReceiver_HandleRequest_OnError(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := core.NewFluxorContext(ctx, gocmd)

	errorHandled := false
	config := &ReceiverConfig{
		Prefix: "/webhooks",
		Endpoints: []EndpointConfig{
			{
				Path:            "/github",
				Secret:          "test-secret",
				EventBusAddress: "webhook.github",
			},
		},
		OnError: func(path string, err error) error {
			errorHandled = true
			return nil // Suppress error
		},
	}

	receiver := NewReceiver(config)
	if err := receiver.Start(fluxorCtx); err != nil {
		t.Fatalf("Receiver.Start() error = %v", err)
	}
	defer receiver.Stop(fluxorCtx)

	req := &WebhookRequest{
		Path:    "/webhooks/github",
		Payload: []byte("test-payload"),
		Headers: map[string]string{
			"X-Webhook-Signature": "sha256=invalid",
		},
		QueryParams: make(map[string]string),
		Method:      "POST",
	}

	// Error handler should be called
	err := receiver.HandleRequest(req)
	if err != nil {
		t.Errorf("Receiver.HandleRequest() error = %v (expected nil due to OnError)", err)
	}

	if !errorHandled {
		t.Error("OnError handler was not called")
	}
}

func TestReceiver_GetEndpoint(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := core.NewFluxorContext(ctx, gocmd)

	config := &ReceiverConfig{
		Prefix: "/webhooks",
		Endpoints: []EndpointConfig{
			{
				Path:            "/github",
				Secret:          "test-secret",
				EventBusAddress: "webhook.github",
			},
			{
				Path:            "/stripe",
				Secret:          "stripe-secret",
				EventBusAddress: "webhook.stripe",
			},
		},
	}

	receiver := NewReceiver(config)
	if err := receiver.Start(fluxorCtx); err != nil {
		t.Fatalf("Receiver.Start() error = %v", err)
	}
	defer receiver.Stop(fluxorCtx)

	// Test existing endpoint
	endpoint, ok := receiver.GetEndpoint("/webhooks/github")
	if !ok {
		t.Error("GetEndpoint() should return true for existing endpoint")
	}
	if endpoint == nil {
		t.Error("GetEndpoint() should return non-nil endpoint")
	}
	if endpoint.EventBusAddress != "webhook.github" {
		t.Errorf("GetEndpoint() EventBusAddress = %v, want webhook.github", endpoint.EventBusAddress)
	}

	// Test non-existing endpoint
	_, ok = receiver.GetEndpoint("/webhooks/unknown")
	if ok {
		t.Error("GetEndpoint() should return false for non-existing endpoint")
	}
}

func TestReceiver_GetEndpoints(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := core.NewFluxorContext(ctx, gocmd)

	config := &ReceiverConfig{
		Prefix: "/webhooks",
		Endpoints: []EndpointConfig{
			{
				Path:            "/github",
				Secret:          "test-secret",
				EventBusAddress: "webhook.github",
			},
			{
				Path:            "/stripe",
				Secret:          "stripe-secret",
				EventBusAddress: "webhook.stripe",
			},
		},
	}

	receiver := NewReceiver(config)
	if err := receiver.Start(fluxorCtx); err != nil {
		t.Fatalf("Receiver.Start() error = %v", err)
	}
	defer receiver.Stop(fluxorCtx)

	endpoints := receiver.GetEndpoints()
	if len(endpoints) != 2 {
		t.Errorf("GetEndpoints() returned %d endpoints, want 2", len(endpoints))
	}

	// Check that both endpoints are present
	found := make(map[string]bool)
	for _, path := range endpoints {
		found[path] = true
	}

	if !found["/webhooks/github"] {
		t.Error("GetEndpoints() missing /webhooks/github")
	}
	if !found["/webhooks/stripe"] {
		t.Error("GetEndpoints() missing /webhooks/stripe")
	}
}

func TestReceiver_PathNormalization(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := core.NewFluxorContext(ctx, gocmd)

	config := &ReceiverConfig{
		Prefix: "/webhooks",
		Endpoints: []EndpointConfig{
			{
				Path:            "github", // No leading slash
				Secret:          "test-secret",
				EventBusAddress: "webhook.github",
			},
			{
				Path:            "/stripe", // With leading slash
				Secret:          "stripe-secret",
				EventBusAddress: "webhook.stripe",
			},
		},
	}

	receiver := NewReceiver(config)
	if err := receiver.Start(fluxorCtx); err != nil {
		t.Fatalf("Receiver.Start() error = %v", err)
	}
	defer receiver.Stop(fluxorCtx)

	// Both should be accessible with normalized paths
	_, ok1 := receiver.GetEndpoint("/webhooks/github")
	_, ok2 := receiver.GetEndpoint("/webhooks/stripe")

	if !ok1 {
		t.Error("GetEndpoint() should find /webhooks/github")
	}
	if !ok2 {
		t.Error("GetEndpoint() should find /webhooks/stripe")
	}
}

func TestParseWebhookRequest(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()

	// Create a mock FastRequestContext
	reqCtx := &fasthttp.RequestCtx{}
	reqCtx.Request.SetRequestURI("/webhooks/github?param=value")
	reqCtx.Request.Header.SetMethod("POST")
	reqCtx.Request.Header.Set("X-Webhook-Signature", "sha256=test")
	reqCtx.Request.SetBodyString("test-payload")

	fastCtx := &web.FastRequestContext{
		BaseRequestContext: core.NewBaseRequestContext(),
		RequestCtx:         reqCtx,
		GoCMD:              gocmd,
		EventBus:           gocmd.EventBus(),
		Params:             make(map[string]string),
	}

	req, err := parseWebhookRequest(fastCtx, "/webhooks/github")
	if err != nil {
		t.Fatalf("parseWebhookRequest() error = %v", err)
	}

	if req.Path != "/webhooks/github" {
		t.Errorf("parseWebhookRequest() Path = %v, want %v", req.Path, "/webhooks/github")
	}

	if string(req.Payload) != "test-payload" {
		t.Errorf("parseWebhookRequest() Payload = %v, want %v", string(req.Payload), "test-payload")
	}

	if req.Headers["X-Webhook-Signature"] != "sha256=test" {
		t.Errorf("parseWebhookRequest() Headers[X-Webhook-Signature] = %v, want %v", req.Headers["X-Webhook-Signature"], "sha256=test")
	}

	if req.QueryParams["param"] != "value" {
		t.Errorf("parseWebhookRequest() QueryParams[param] = %v, want %v", req.QueryParams["param"], "value")
	}

	if req.Method != "POST" {
		t.Errorf("parseWebhookRequest() Method = %v, want %v", req.Method, "POST")
	}
}

func TestNewReceiver_NilConfig(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("NewReceiver() with nil config should panic")
		}
	}()

	NewReceiver(nil)
}

func TestDefaultReceiverConfig(t *testing.T) {
	config := DefaultReceiverConfig()

	if config.Prefix != "/webhooks" {
		t.Errorf("DefaultReceiverConfig() Prefix = %v, want /webhooks", config.Prefix)
	}

	if len(config.Endpoints) != 0 {
		t.Errorf("DefaultReceiverConfig() Endpoints length = %v, want 0", len(config.Endpoints))
	}
}

func TestReceiver_HandleRequest_EventBusError(t *testing.T) {
	ctx := context.Background()
	gocmd := core.NewGoCMD(ctx)
	defer gocmd.Close()
	fluxorCtx := core.NewFluxorContext(ctx, gocmd)

	// Create a mock event bus that returns an error
	mockEventBus := &mockEventBus{shouldError: true}

	config := &ReceiverConfig{
		Prefix: "/webhooks",
		Endpoints: []EndpointConfig{
			{
				Path:            "/github",
				Secret:          "test-secret",
				EventBusAddress: "webhook.github",
				SkipValidation:  true, // Skip validation for this test
			},
		},
	}

	receiver := NewReceiver(config)
	if err := receiver.Start(fluxorCtx); err != nil {
		t.Fatalf("Receiver.Start() error = %v", err)
	}
	defer receiver.Stop(fluxorCtx)

	// Replace event bus with mock
	receiver.eventBus = mockEventBus

	req := &WebhookRequest{
		Path:        "/webhooks/github",
		Payload:     []byte("test-payload"),
		Headers:     make(map[string]string),
		QueryParams: make(map[string]string),
		Method:      "POST",
	}

	err := receiver.HandleRequest(req)
	if err == nil {
		t.Error("Receiver.HandleRequest() expected error when EventBus.Publish fails")
	}
}

// mockEventBus is a simple mock for testing
type mockEventBus struct {
	shouldError bool
}

func (m *mockEventBus) Publish(address string, body interface{}) error {
	if m.shouldError {
		return errors.New("mock event bus error")
	}
	return nil
}

func (m *mockEventBus) Send(address string, body interface{}, timeout interface{}) (interface{}, error) {
	return nil, errors.New("not implemented")
}

func (m *mockEventBus) Consumer(address string) core.Consumer {
	return nil
}

func (m *mockEventBus) Close() error {
	return nil
}
package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type paymentIntentsClient struct {
	client *stripeClient
}

func (p *paymentIntentsClient) Create(ctx context.Context, params *PaymentIntentParams) (*PaymentIntent, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}

	values := url.Values{}
	if params.Amount > 0 {
		values.Set("amount", strconv.FormatInt(params.Amount, 10))
	}
	if params.Currency != "" {
		values.Set("currency", params.Currency)
	}
	if params.Customer != "" {
		values.Set("customer", params.Customer)
	}
	if params.Description != "" {
		values.Set("description", params.Description)
	}
	if params.PaymentMethod != "" {
		values.Set("payment_method", params.PaymentMethod)
	}
	if params.CaptureMethod != "" {
		values.Set("capture_method", params.CaptureMethod)
	}
	if params.Confirm {
		values.Set("confirm", "true")
	}
	if params.ReturnURL != "" {
		values.Set("return_url", params.ReturnURL)
	}
	for i, pm := range params.PaymentMethodTypes {
		values.Set(fmt.Sprintf("payment_method_types[%d]", i), pm)
	}
	for k, v := range params.Metadata {
		values.Set("metadata["+k+"]", v)
	}

	respBody, err := p.client.doRequest(ctx, "POST", "/payment_intents", values)
	if err != nil {
		return nil, fmt.Errorf("failed to create payment intent: %w", err)
	}

	var intent PaymentIntent
	if err := json.Unmarshal(respBody, &intent); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &intent, nil
}

func (p *paymentIntentsClient) Get(ctx context.Context, id string) (*PaymentIntent, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	respBody, err := p.client.doRequest(ctx, "GET", "/payment_intents/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment intent: %w", err)
	}

	var intent PaymentIntent
	if err := json.Unmarshal(respBody, &intent); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &intent, nil
}

func (p *paymentIntentsClient) Update(ctx context.Context, id string, params *PaymentIntentParams) (*PaymentIntent, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	values := url.Values{}
	if params != nil {
		if params.Description != "" {
			values.Set("description", params.Description)
		}
		for k, v := range params.Metadata {
			values.Set("metadata["+k+"]", v)
		}
	}

	respBody, err := p.client.doRequest(ctx, "POST", "/payment_intents/"+id, values)
	if err != nil {
		return nil, fmt.Errorf("failed to update payment intent: %w", err)
	}

	var intent PaymentIntent
	if err := json.Unmarshal(respBody, &intent); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &intent, nil
}

func (p *paymentIntentsClient) Confirm(ctx context.Context, id string, params *PaymentIntentConfirmParams) (*PaymentIntent, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	values := url.Values{}
	if params != nil {
		if params.PaymentMethod != "" {
			values.Set("payment_method", params.PaymentMethod)
		}
		if params.ReturnURL != "" {
			values.Set("return_url", params.ReturnURL)
		}
	}

	respBody, err := p.client.doRequest(ctx, "POST", "/payment_intents/"+id+"/confirm", values)
	if err != nil {
		return nil, fmt.Errorf("failed to confirm payment intent: %w", err)
	}

	var intent PaymentIntent
	if err := json.Unmarshal(respBody, &intent); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &intent, nil
}

func (p *paymentIntentsClient) Cancel(ctx context.Context, id string) (*PaymentIntent, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	respBody, err := p.client.doRequest(ctx, "POST", "/payment_intents/"+id+"/cancel", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel payment intent: %w", err)
	}

	var intent PaymentIntent
	if err := json.Unmarshal(respBody, &intent); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &intent, nil
}

func (p *paymentIntentsClient) Capture(ctx context.Context, id string, params *PaymentIntentCaptureParams) (*PaymentIntent, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	values := url.Values{}
	if params != nil && params.AmountToCapture > 0 {
		values.Set("amount_to_capture", strconv.FormatInt(params.AmountToCapture, 10))
	}

	respBody, err := p.client.doRequest(ctx, "POST", "/payment_intents/"+id+"/capture", values)
	if err != nil {
		return nil, fmt.Errorf("failed to capture payment intent: %w", err)
	}

	var intent PaymentIntent
	if err := json.Unmarshal(respBody, &intent); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &intent, nil
}

func (p *paymentIntentsClient) List(ctx context.Context, params *ListParams) (*PaymentIntentList, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	values := url.Values{}
	if params != nil {
		if params.Limit > 0 {
			values.Set("limit", strconv.Itoa(params.Limit))
		}
		if params.StartingAfter != "" {
			values.Set("starting_after", params.StartingAfter)
		}
		if params.EndingBefore != "" {
			values.Set("ending_before", params.EndingBefore)
		}
	}

	path := "/payment_intents"
	if len(values) > 0 {
		path += "?" + values.Encode()
	}

	respBody, err := p.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list payment intents: %w", err)
	}

	var list PaymentIntentList
	if err := json.Unmarshal(respBody, &list); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &list, nil
}

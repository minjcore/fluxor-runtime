package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type subscriptionsClient struct {
	client *stripeClient
}

func (s *subscriptionsClient) Create(ctx context.Context, params *SubscriptionParams) (*Subscription, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}

	values := url.Values{}
	if params.Customer != "" {
		values.Set("customer", params.Customer)
	}
	for i, item := range params.Items {
		if item.Price != "" {
			values.Set(fmt.Sprintf("items[%d][price]", i), item.Price)
		}
		if item.Quantity > 0 {
			values.Set(fmt.Sprintf("items[%d][quantity]", i), strconv.FormatInt(item.Quantity, 10))
		}
	}
	if params.DefaultPaymentMethod != "" {
		values.Set("default_payment_method", params.DefaultPaymentMethod)
	}
	if params.TrialPeriodDays > 0 {
		values.Set("trial_period_days", strconv.Itoa(params.TrialPeriodDays))
	}
	for k, v := range params.Metadata {
		values.Set("metadata["+k+"]", v)
	}

	respBody, err := s.client.doRequest(ctx, "POST", "/subscriptions", values)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	var sub Subscription
	if err := json.Unmarshal(respBody, &sub); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &sub, nil
}

func (s *subscriptionsClient) Get(ctx context.Context, id string) (*Subscription, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	respBody, err := s.client.doRequest(ctx, "GET", "/subscriptions/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	var sub Subscription
	if err := json.Unmarshal(respBody, &sub); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &sub, nil
}

func (s *subscriptionsClient) Update(ctx context.Context, id string, params *SubscriptionParams) (*Subscription, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	values := url.Values{}
	if params != nil {
		if params.DefaultPaymentMethod != "" {
			values.Set("default_payment_method", params.DefaultPaymentMethod)
		}
		if params.CancelAtPeriodEnd {
			values.Set("cancel_at_period_end", "true")
		}
		for k, v := range params.Metadata {
			values.Set("metadata["+k+"]", v)
		}
	}

	respBody, err := s.client.doRequest(ctx, "POST", "/subscriptions/"+id, values)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscription: %w", err)
	}

	var sub Subscription
	if err := json.Unmarshal(respBody, &sub); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &sub, nil
}

func (s *subscriptionsClient) Cancel(ctx context.Context, id string, params *SubscriptionCancelParams) (*Subscription, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	values := url.Values{}
	if params != nil {
		if params.InvoiceNow {
			values.Set("invoice_now", "true")
		}
		if params.Prorate {
			values.Set("prorate", "true")
		}
	}

	respBody, err := s.client.doRequest(ctx, "DELETE", "/subscriptions/"+id, values)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel subscription: %w", err)
	}

	var sub Subscription
	if err := json.Unmarshal(respBody, &sub); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &sub, nil
}

func (s *subscriptionsClient) List(ctx context.Context, params *SubscriptionListParams) (*SubscriptionList, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	values := url.Values{}
	if params != nil {
		if params.Customer != "" {
			values.Set("customer", params.Customer)
		}
		if params.Price != "" {
			values.Set("price", params.Price)
		}
		if params.Status != "" {
			values.Set("status", params.Status)
		}
		if params.Limit > 0 {
			values.Set("limit", strconv.Itoa(params.Limit))
		}
		if params.StartingAfter != "" {
			values.Set("starting_after", params.StartingAfter)
		}
	}

	path := "/subscriptions"
	if len(values) > 0 {
		path += "?" + values.Encode()
	}

	respBody, err := s.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	var list SubscriptionList
	if err := json.Unmarshal(respBody, &list); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &list, nil
}

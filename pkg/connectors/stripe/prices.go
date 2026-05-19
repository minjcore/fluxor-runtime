package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type pricesClient struct {
	client *stripeClient
}

func (p *pricesClient) Create(ctx context.Context, params *PriceParams) (*Price, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}

	values := url.Values{}
	if params.Currency != "" {
		values.Set("currency", params.Currency)
	}
	if params.Product != "" {
		values.Set("product", params.Product)
	}
	if params.UnitAmount > 0 {
		values.Set("unit_amount", strconv.FormatInt(params.UnitAmount, 10))
	}
	if params.Nickname != "" {
		values.Set("nickname", params.Nickname)
	}
	if params.Active != nil {
		values.Set("active", strconv.FormatBool(*params.Active))
	}
	if params.Recurring != nil {
		if params.Recurring.Interval != "" {
			values.Set("recurring[interval]", params.Recurring.Interval)
		}
		if params.Recurring.IntervalCount > 0 {
			values.Set("recurring[interval_count]", strconv.Itoa(params.Recurring.IntervalCount))
		}
		if params.Recurring.UsageType != "" {
			values.Set("recurring[usage_type]", params.Recurring.UsageType)
		}
	}
	for k, v := range params.Metadata {
		values.Set("metadata["+k+"]", v)
	}

	respBody, err := p.client.doRequest(ctx, "POST", "/prices", values)
	if err != nil {
		return nil, fmt.Errorf("failed to create price: %w", err)
	}

	var price Price
	if err := json.Unmarshal(respBody, &price); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &price, nil
}

func (p *pricesClient) Get(ctx context.Context, id string) (*Price, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	respBody, err := p.client.doRequest(ctx, "GET", "/prices/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get price: %w", err)
	}

	var price Price
	if err := json.Unmarshal(respBody, &price); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &price, nil
}

func (p *pricesClient) Update(ctx context.Context, id string, params *PriceParams) (*Price, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	values := url.Values{}
	if params != nil {
		if params.Nickname != "" {
			values.Set("nickname", params.Nickname)
		}
		if params.Active != nil {
			values.Set("active", strconv.FormatBool(*params.Active))
		}
		for k, v := range params.Metadata {
			values.Set("metadata["+k+"]", v)
		}
	}

	respBody, err := p.client.doRequest(ctx, "POST", "/prices/"+id, values)
	if err != nil {
		return nil, fmt.Errorf("failed to update price: %w", err)
	}

	var price Price
	if err := json.Unmarshal(respBody, &price); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &price, nil
}

func (p *pricesClient) List(ctx context.Context, params *PriceListParams) (*PriceList, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	values := url.Values{}
	if params != nil {
		if params.Product != "" {
			values.Set("product", params.Product)
		}
		if params.Currency != "" {
			values.Set("currency", params.Currency)
		}
		if params.Type != "" {
			values.Set("type", params.Type)
		}
		if params.Active != nil {
			values.Set("active", strconv.FormatBool(*params.Active))
		}
		if params.Limit > 0 {
			values.Set("limit", strconv.Itoa(params.Limit))
		}
		if params.StartingAfter != "" {
			values.Set("starting_after", params.StartingAfter)
		}
	}

	path := "/prices"
	if len(values) > 0 {
		path += "?" + values.Encode()
	}

	respBody, err := p.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list prices: %w", err)
	}

	var list PriceList
	if err := json.Unmarshal(respBody, &list); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &list, nil
}

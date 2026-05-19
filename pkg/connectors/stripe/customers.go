package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type customersClient struct {
	client *stripeClient
}

func (c *customersClient) Create(ctx context.Context, params *CustomerParams) (*Customer, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	values := url.Values{}
	if params != nil {
		if params.Email != "" {
			values.Set("email", params.Email)
		}
		if params.Name != "" {
			values.Set("name", params.Name)
		}
		if params.Description != "" {
			values.Set("description", params.Description)
		}
		if params.Phone != "" {
			values.Set("phone", params.Phone)
		}
		for k, v := range params.Metadata {
			values.Set("metadata["+k+"]", v)
		}
	}

	respBody, err := c.client.doRequest(ctx, "POST", "/customers", values)
	if err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	var customer Customer
	if err := json.Unmarshal(respBody, &customer); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &customer, nil
}

func (c *customersClient) Get(ctx context.Context, id string) (*Customer, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	respBody, err := c.client.doRequest(ctx, "GET", "/customers/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	var customer Customer
	if err := json.Unmarshal(respBody, &customer); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &customer, nil
}

func (c *customersClient) Update(ctx context.Context, id string, params *CustomerParams) (*Customer, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	values := url.Values{}
	if params != nil {
		if params.Email != "" {
			values.Set("email", params.Email)
		}
		if params.Name != "" {
			values.Set("name", params.Name)
		}
		if params.Description != "" {
			values.Set("description", params.Description)
		}
		if params.Phone != "" {
			values.Set("phone", params.Phone)
		}
		for k, v := range params.Metadata {
			values.Set("metadata["+k+"]", v)
		}
	}

	respBody, err := c.client.doRequest(ctx, "POST", "/customers/"+id, values)
	if err != nil {
		return nil, fmt.Errorf("failed to update customer: %w", err)
	}

	var customer Customer
	if err := json.Unmarshal(respBody, &customer); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &customer, nil
}

func (c *customersClient) Delete(ctx context.Context, id string) (*DeletedCustomer, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	respBody, err := c.client.doRequest(ctx, "DELETE", "/customers/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to delete customer: %w", err)
	}

	var deleted DeletedCustomer
	if err := json.Unmarshal(respBody, &deleted); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &deleted, nil
}

func (c *customersClient) List(ctx context.Context, params *ListParams) (*CustomerList, error) {
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

	path := "/customers"
	if len(values) > 0 {
		path += "?" + values.Encode()
	}

	respBody, err := c.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list customers: %w", err)
	}

	var list CustomerList
	if err := json.Unmarshal(respBody, &list); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &list, nil
}

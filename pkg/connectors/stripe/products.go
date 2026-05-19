package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type productsClient struct {
	client *stripeClient
}

func (p *productsClient) Create(ctx context.Context, params *ProductParams) (*Product, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if params == nil {
		return nil, fmt.Errorf("params cannot be nil")
	}
	if params.Name == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}

	values := url.Values{}
	values.Set("name", params.Name)
	if params.Description != "" {
		values.Set("description", params.Description)
	}
	if params.Active != nil {
		values.Set("active", strconv.FormatBool(*params.Active))
	}
	for i, img := range params.Images {
		values.Set(fmt.Sprintf("images[%d]", i), img)
	}
	for k, v := range params.Metadata {
		values.Set("metadata["+k+"]", v)
	}

	respBody, err := p.client.doRequest(ctx, "POST", "/products", values)
	if err != nil {
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	var product Product
	if err := json.Unmarshal(respBody, &product); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &product, nil
}

func (p *productsClient) Get(ctx context.Context, id string) (*Product, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	respBody, err := p.client.doRequest(ctx, "GET", "/products/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get product: %w", err)
	}

	var product Product
	if err := json.Unmarshal(respBody, &product); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &product, nil
}

func (p *productsClient) Update(ctx context.Context, id string, params *ProductParams) (*Product, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	values := url.Values{}
	if params != nil {
		if params.Name != "" {
			values.Set("name", params.Name)
		}
		if params.Description != "" {
			values.Set("description", params.Description)
		}
		if params.Active != nil {
			values.Set("active", strconv.FormatBool(*params.Active))
		}
		for k, v := range params.Metadata {
			values.Set("metadata["+k+"]", v)
		}
	}

	respBody, err := p.client.doRequest(ctx, "POST", "/products/"+id, values)
	if err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	var product Product
	if err := json.Unmarshal(respBody, &product); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &product, nil
}

func (p *productsClient) Delete(ctx context.Context, id string) (*DeletedProduct, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	respBody, err := p.client.doRequest(ctx, "DELETE", "/products/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to delete product: %w", err)
	}

	var deleted DeletedProduct
	if err := json.Unmarshal(respBody, &deleted); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &deleted, nil
}

func (p *productsClient) List(ctx context.Context, params *ListParams) (*ProductList, error) {
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

	path := "/products"
	if len(values) > 0 {
		path += "?" + values.Encode()
	}

	respBody, err := p.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list products: %w", err)
	}

	var list ProductList
	if err := json.Unmarshal(respBody, &list); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &list, nil
}

package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type invoicesClient struct {
	client *stripeClient
}

func (i *invoicesClient) Create(ctx context.Context, params *InvoiceParams) (*Invoice, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	values := url.Values{}
	if params != nil {
		if params.Customer != "" {
			values.Set("customer", params.Customer)
		}
		if params.Subscription != "" {
			values.Set("subscription", params.Subscription)
		}
		if params.Description != "" {
			values.Set("description", params.Description)
		}
		if params.CollectionMethod != "" {
			values.Set("collection_method", params.CollectionMethod)
		}
		if params.DaysUntilDue > 0 {
			values.Set("days_until_due", strconv.Itoa(params.DaysUntilDue))
		}
		for k, v := range params.Metadata {
			values.Set("metadata["+k+"]", v)
		}
	}

	respBody, err := i.client.doRequest(ctx, "POST", "/invoices", values)
	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	var invoice Invoice
	if err := json.Unmarshal(respBody, &invoice); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &invoice, nil
}

func (i *invoicesClient) Get(ctx context.Context, id string) (*Invoice, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	respBody, err := i.client.doRequest(ctx, "GET", "/invoices/"+id, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	var invoice Invoice
	if err := json.Unmarshal(respBody, &invoice); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &invoice, nil
}

func (i *invoicesClient) Update(ctx context.Context, id string, params *InvoiceParams) (*Invoice, error) {
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

	respBody, err := i.client.doRequest(ctx, "POST", "/invoices/"+id, values)
	if err != nil {
		return nil, fmt.Errorf("failed to update invoice: %w", err)
	}

	var invoice Invoice
	if err := json.Unmarshal(respBody, &invoice); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &invoice, nil
}

func (i *invoicesClient) Pay(ctx context.Context, id string) (*Invoice, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	respBody, err := i.client.doRequest(ctx, "POST", "/invoices/"+id+"/pay", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to pay invoice: %w", err)
	}

	var invoice Invoice
	if err := json.Unmarshal(respBody, &invoice); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &invoice, nil
}

func (i *invoicesClient) SendInvoice(ctx context.Context, id string) (*Invoice, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	respBody, err := i.client.doRequest(ctx, "POST", "/invoices/"+id+"/send", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to send invoice: %w", err)
	}

	var invoice Invoice
	if err := json.Unmarshal(respBody, &invoice); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &invoice, nil
}

func (i *invoicesClient) VoidInvoice(ctx context.Context, id string) (*Invoice, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if id == "" {
		return nil, fmt.Errorf("id cannot be empty")
	}

	respBody, err := i.client.doRequest(ctx, "POST", "/invoices/"+id+"/void", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to void invoice: %w", err)
	}

	var invoice Invoice
	if err := json.Unmarshal(respBody, &invoice); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &invoice, nil
}

func (i *invoicesClient) List(ctx context.Context, params *InvoiceListParams) (*InvoiceList, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	values := url.Values{}
	if params != nil {
		if params.Customer != "" {
			values.Set("customer", params.Customer)
		}
		if params.Subscription != "" {
			values.Set("subscription", params.Subscription)
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

	path := "/invoices"
	if len(values) > 0 {
		path += "?" + values.Encode()
	}

	respBody, err := i.client.doRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to list invoices: %w", err)
	}

	var list InvoiceList
	if err := json.Unmarshal(respBody, &list); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &list, nil
}

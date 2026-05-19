package hoadondientu

import (
	"context"
	"fmt"
	"net/url"
)

// Create creates a new e-invoice (draft). Use Publish to submit to tax authority.
func (i *invoicesClient) Create(ctx context.Context, params *CreateInvoiceParams) (*Invoice, error) {
	if params == nil {
		return nil, fmt.Errorf("hoadondientu: params cannot be nil")
	}
	var out Invoice
	if err := i.client.do(ctx, "POST", "/invoices", params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get returns an invoice by ID.
func (i *invoicesClient) Get(ctx context.Context, id string) (*Invoice, error) {
	if id == "" {
		return nil, fmt.Errorf("hoadondientu: id cannot be empty")
	}
	var out Invoice
	if err := i.client.doGet(ctx, "/invoices/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// List returns a paginated list of invoices.
func (i *invoicesClient) List(ctx context.Context, params *ListInvoicesParams) (*ListInvoicesResult, error) {
	q := url.Values{}
	if params != nil {
		if params.Page > 0 {
			q.Set("page", fmt.Sprintf("%d", params.Page))
		}
		if params.PageSize > 0 {
			q.Set("pageSize", fmt.Sprintf("%d", params.PageSize))
		}
		if params.FromDate != "" {
			q.Set("fromDate", params.FromDate)
		}
		if params.ToDate != "" {
			q.Set("toDate", params.ToDate)
		}
		if params.Status != "" {
			q.Set("status", params.Status)
		}
		if params.InvNo != "" {
			q.Set("invNo", params.InvNo)
		}
		if params.TemplateCode != "" {
			q.Set("templateCode", params.TemplateCode)
		}
	}
	var out ListInvoicesResult
	if err := i.client.doGet(ctx, "/invoices", q, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update updates a draft invoice.
func (i *invoicesClient) Update(ctx context.Context, id string, params *UpdateInvoiceParams) (*Invoice, error) {
	if id == "" {
		return nil, fmt.Errorf("hoadondientu: id cannot be empty")
	}
	var out Invoice
	if err := i.client.do(ctx, "PUT", "/invoices/"+url.PathEscape(id), params, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Delete deletes a draft invoice (or cancels per provider rules).
func (i *invoicesClient) Delete(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("hoadondientu: id cannot be empty")
	}
	return i.client.do(ctx, "DELETE", "/invoices/"+url.PathEscape(id), nil, nil)
}

// GetStatus returns the current status of an invoice.
func (i *invoicesClient) GetStatus(ctx context.Context, id string) (InvoiceStatus, error) {
	inv, err := i.Get(ctx, id)
	if err != nil {
		return "", err
	}
	return inv.Status, nil
}

// Publish publishes (phát hành) the invoice to tax authority.
func (i *invoicesClient) Publish(ctx context.Context, id string) (*Invoice, error) {
	if id == "" {
		return nil, fmt.Errorf("hoadondientu: id cannot be empty")
	}
	var out Invoice
	if err := i.client.do(ctx, "POST", "/invoices/"+url.PathEscape(id)+"/publish", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DownloadPDF returns the invoice PDF bytes.
func (i *invoicesClient) DownloadPDF(ctx context.Context, id string) ([]byte, error) {
	if id == "" {
		return nil, fmt.Errorf("hoadondientu: id cannot be empty")
	}
	path := "/invoices/" + url.PathEscape(id) + "/download?format=pdf"
	data, err := i.client.doBinary(ctx, "GET", path)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// DownloadXML returns the invoice XML bytes (signed XML per Circular 78/2021).
func (i *invoicesClient) DownloadXML(ctx context.Context, id string) ([]byte, error) {
	if id == "" {
		return nil, fmt.Errorf("hoadondientu: id cannot be empty")
	}
	path := "/invoices/" + url.PathEscape(id) + "/download?format=xml"
	data, err := i.client.doBinary(ctx, "GET", path)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// SendEmail sends the invoice to the given email address.
func (i *invoicesClient) SendEmail(ctx context.Context, id string, email string) error {
	if id == "" || email == "" {
		return fmt.Errorf("hoadondientu: id and email are required")
	}
	body := map[string]string{"email": email}
	return i.client.do(ctx, "POST", "/invoices/"+url.PathEscape(id)+"/send-email", body, nil)
}

// ListTemplates returns available invoice templates (mẫu số, ký hiệu).
func (i *invoicesClient) ListTemplates(ctx context.Context) ([]InvoiceTemplate, error) {
	var out []InvoiceTemplate
	if err := i.client.doGet(ctx, "/templates", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

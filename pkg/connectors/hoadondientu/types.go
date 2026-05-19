package hoadondientu

// InvoiceStatus represents Vietnam e-invoice status (draft, published, cancelled, etc.).
type InvoiceStatus string

const (
	InvoiceStatusDraft     InvoiceStatus = "draft"
	InvoiceStatusPublished InvoiceStatus = "published"
	InvoiceStatusCancelled InvoiceStatus = "cancelled"
	InvoiceStatusReplaced  InvoiceStatus = "replaced"
)

// InvoiceTemplate represents mẫu hóa đơn / ký hiệu (template/symbol).
type InvoiceTemplate struct {
	TemplateCode string `json:"templateCode,omitempty"` // Mẫu số
	Symbol       string `json:"symbol,omitempty"`      // Ký hiệu
	Name         string `json:"name,omitempty"`
}

// Buyer represents bên mua (customer).
type Buyer struct {
	TaxCode     string `json:"taxCode,omitempty"`     // Mã số thuế
	Name        string `json:"name,omitempty"`        // Tên
	Address     string `json:"address,omitempty"`     // Địa chỉ
	Email       string `json:"email,omitempty"`
	Phone       string `json:"phone,omitempty"`
	BankAccount string `json:"bankAccount,omitempty"` // Số tài khoản
	BankName    string `json:"bankName,omitempty"`
}

// Seller represents bên bán (your company).
type Seller struct {
	TaxCode     string `json:"taxCode,omitempty"`
	Name        string `json:"name,omitempty"`
	Address     string `json:"address,omitempty"`
	Phone       string `json:"phone,omitempty"`
	Email       string `json:"email,omitempty"`
	BankAccount string `json:"bankAccount,omitempty"`
	BankName    string `json:"bankName,omitempty"`
}

// InvoiceItem represents dòng hóa đơn (line item).
type InvoiceItem struct {
	LineNumber   int     `json:"lineNumber,omitempty"`
	Name         string  `json:"name,omitempty"`         // Tên hàng hóa/dịch vụ
	Quantity     float64 `json:"quantity,omitempty"`     // Số lượng
	Unit         string  `json:"unit,omitempty"`        // Đơn vị tính
	UnitPrice    float64 `json:"unitPrice,omitempty"`    // Đơn giá
	Amount       float64 `json:"amount,omitempty"`      // Thành tiền
	VATRate      float64 `json:"vatRate,omitempty"`      // Thuế suất (%)
	VATAmount    float64 `json:"vatAmount,omitempty"`   // Tiền thuế
	Discount     float64 `json:"discount,omitempty"`    // Chiết khấu
	TotalAmount  float64 `json:"totalAmount,omitempty"`  // Tổng cộng
}

// Invoice represents a Vietnam e-invoice (hóa đơn điện tử).
// Fields align with common providers (MISA meInvoice, VNPT, etc.) and Circular 78/2021/TT-BTC.
type Invoice struct {
	ID               string         `json:"id,omitempty"`
	InvNo            string         `json:"invNo,omitempty"`            // Số hóa đơn
	TemplateCode     string         `json:"templateCode,omitempty"`     // Mẫu số
	Symbol           string         `json:"symbol,omitempty"`           // Ký hiệu
	IssueDate        string         `json:"issueDate,omitempty"`        // Ngày phát hành (yyyy-MM-dd)
	KindOfService    string         `json:"kindOfService,omitempty"`    // Loại hóa đơn: 1-Bán hàng, 2-...
	PaymentMethod    string         `json:"paymentMethod,omitempty"`    // Hình thức thanh toán
	Currency         string         `json:"currency,omitempty"`        // VND
	ExchangeRate     float64        `json:"exchangeRate,omitempty"`
	Seller           *Seller        `json:"seller,omitempty"`
	Buyer            *Buyer         `json:"buyer,omitempty"`
	Items            []InvoiceItem  `json:"items,omitempty"`
	TotalAmount      float64        `json:"totalAmount,omitempty"`      // Tổng tiền trước thuế
	TotalVATAmount   float64        `json:"totalVatAmount,omitempty"`   // Tổng tiền thuế
	TotalDiscount    float64        `json:"totalDiscount,omitempty"`
	GrossAmount      float64        `json:"grossAmount,omitempty"`      // Tổng cộng (bằng chữ)
	AmountInWords    string         `json:"amountInWords,omitempty"`     // Số tiền bằng chữ
	Note             string         `json:"note,omitempty"`
	Status           InvoiceStatus  `json:"status,omitempty"`
	TaxAuthorityCode string         `json:"taxAuthorityCode,omitempty"` // Mã CQT
	TransactionID    string         `json:"transactionId,omitempty"`    // ID giao dịch từ CQT
	CreatedAt        string         `json:"createdAt,omitempty"`
	UpdatedAt        string         `json:"updatedAt,omitempty"`
	Extra            map[string]interface{} `json:"extra,omitempty"`
}

// CreateInvoiceParams parameters for creating an e-invoice.
type CreateInvoiceParams struct {
	TemplateCode  string        `json:"templateCode,omitempty"`
	Symbol        string        `json:"symbol,omitempty"`
	IssueDate     string        `json:"issueDate,omitempty"`     // yyyy-MM-dd
	KindOfService string        `json:"kindOfService,omitempty"` // 1, 2, ...
	PaymentMethod string        `json:"paymentMethod,omitempty"`
	Currency      string        `json:"currency,omitempty"`      // VND
	Seller        *Seller       `json:"seller,omitempty"`
	Buyer         *Buyer        `json:"buyer,omitempty"`
	Items         []InvoiceItem `json:"items,omitempty"`
	Note          string        `json:"note,omitempty"`
	Extra         map[string]interface{} `json:"extra,omitempty"`
}

// UpdateInvoiceParams parameters for updating an e-invoice (draft).
type UpdateInvoiceParams struct {
	IssueDate     string        `json:"issueDate,omitempty"`
	PaymentMethod string        `json:"paymentMethod,omitempty"`
	Buyer         *Buyer        `json:"buyer,omitempty"`
	Items         []InvoiceItem `json:"items,omitempty"`
	Note          string        `json:"note,omitempty"`
}

// ListInvoicesParams for listing/searching invoices.
type ListInvoicesParams struct {
	Page       int    `json:"page,omitempty"`
	PageSize   int    `json:"pageSize,omitempty"`
	FromDate   string `json:"fromDate,omitempty"`   // yyyy-MM-dd
	ToDate     string `json:"toDate,omitempty"`    // yyyy-MM-dd
	Status     string `json:"status,omitempty"`
	InvNo      string `json:"invNo,omitempty"`
	TemplateCode string `json:"templateCode,omitempty"`
}

// ListInvoicesResult result of list invoices.
type ListInvoicesResult struct {
	Items      []Invoice `json:"items,omitempty"`
	Total      int       `json:"total,omitempty"`
	Page       int       `json:"page,omitempty"`
	PageSize   int       `json:"pageSize,omitempty"`
}

// TokenResponse from auth/token API.
type TokenResponse struct {
	AccessToken  string `json:"accessToken,omitempty"`
	TokenType    string `json:"tokenType,omitempty"`
	ExpiresIn    int    `json:"expiresIn,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
}

// APIError represents provider API error.
type APIError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
	Details string `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	if e.Details != "" {
		return e.Code + ": " + e.Message + " (" + e.Details + ")"
	}
	return e.Code + ": " + e.Message
}

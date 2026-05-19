package stripe

import "context"

// Client is the main Stripe client interface
type Client interface {
	Customers() CustomersClient
	PaymentIntents() PaymentIntentsClient
	Subscriptions() SubscriptionsClient
	Invoices() InvoicesClient
	Products() ProductsClient
	Prices() PricesClient
}

// CustomersClient provides operations for Stripe customers
type CustomersClient interface {
	Create(ctx context.Context, params *CustomerParams) (*Customer, error)
	Get(ctx context.Context, id string) (*Customer, error)
	Update(ctx context.Context, id string, params *CustomerParams) (*Customer, error)
	Delete(ctx context.Context, id string) (*DeletedCustomer, error)
	List(ctx context.Context, params *ListParams) (*CustomerList, error)
}

// PaymentIntentsClient provides operations for Stripe payment intents
type PaymentIntentsClient interface {
	Create(ctx context.Context, params *PaymentIntentParams) (*PaymentIntent, error)
	Get(ctx context.Context, id string) (*PaymentIntent, error)
	Update(ctx context.Context, id string, params *PaymentIntentParams) (*PaymentIntent, error)
	Confirm(ctx context.Context, id string, params *PaymentIntentConfirmParams) (*PaymentIntent, error)
	Cancel(ctx context.Context, id string) (*PaymentIntent, error)
	Capture(ctx context.Context, id string, params *PaymentIntentCaptureParams) (*PaymentIntent, error)
	List(ctx context.Context, params *ListParams) (*PaymentIntentList, error)
}

// SubscriptionsClient provides operations for Stripe subscriptions
type SubscriptionsClient interface {
	Create(ctx context.Context, params *SubscriptionParams) (*Subscription, error)
	Get(ctx context.Context, id string) (*Subscription, error)
	Update(ctx context.Context, id string, params *SubscriptionParams) (*Subscription, error)
	Cancel(ctx context.Context, id string, params *SubscriptionCancelParams) (*Subscription, error)
	List(ctx context.Context, params *SubscriptionListParams) (*SubscriptionList, error)
}

// InvoicesClient provides operations for Stripe invoices
type InvoicesClient interface {
	Create(ctx context.Context, params *InvoiceParams) (*Invoice, error)
	Get(ctx context.Context, id string) (*Invoice, error)
	Update(ctx context.Context, id string, params *InvoiceParams) (*Invoice, error)
	Pay(ctx context.Context, id string) (*Invoice, error)
	SendInvoice(ctx context.Context, id string) (*Invoice, error)
	VoidInvoice(ctx context.Context, id string) (*Invoice, error)
	List(ctx context.Context, params *InvoiceListParams) (*InvoiceList, error)
}

// ProductsClient provides operations for Stripe products
type ProductsClient interface {
	Create(ctx context.Context, params *ProductParams) (*Product, error)
	Get(ctx context.Context, id string) (*Product, error)
	Update(ctx context.Context, id string, params *ProductParams) (*Product, error)
	Delete(ctx context.Context, id string) (*DeletedProduct, error)
	List(ctx context.Context, params *ListParams) (*ProductList, error)
}

// PricesClient provides operations for Stripe prices
type PricesClient interface {
	Create(ctx context.Context, params *PriceParams) (*Price, error)
	Get(ctx context.Context, id string) (*Price, error)
	Update(ctx context.Context, id string, params *PriceParams) (*Price, error)
	List(ctx context.Context, params *PriceListParams) (*PriceList, error)
}

// Customer represents a Stripe customer
type Customer struct {
	ID             string            `json:"id"`
	Object         string            `json:"object"`
	Address        *Address          `json:"address,omitempty"`
	Balance        int64             `json:"balance"`
	Created        int64             `json:"created"`
	Currency       string            `json:"currency,omitempty"`
	DefaultSource  string            `json:"default_source,omitempty"`
	Deleted        bool              `json:"deleted,omitempty"`
	Delinquent     bool              `json:"delinquent"`
	Description    string            `json:"description,omitempty"`
	Email          string            `json:"email,omitempty"`
	InvoicePrefix  string            `json:"invoice_prefix,omitempty"`
	Livemode       bool              `json:"livemode"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Name           string            `json:"name,omitempty"`
	Phone          string            `json:"phone,omitempty"`
	Shipping       *Shipping         `json:"shipping,omitempty"`
}

// CustomerParams represents parameters for creating/updating a customer
type CustomerParams struct {
	Address       *AddressParams     `json:"address,omitempty"`
	Balance       *int64             `json:"balance,omitempty"`
	Description   string             `json:"description,omitempty"`
	Email         string             `json:"email,omitempty"`
	InvoicePrefix string             `json:"invoice_prefix,omitempty"`
	Metadata      map[string]string  `json:"metadata,omitempty"`
	Name          string             `json:"name,omitempty"`
	Phone         string             `json:"phone,omitempty"`
	Shipping      *ShippingParams    `json:"shipping,omitempty"`
	Source        string             `json:"source,omitempty"`
	PaymentMethod string             `json:"payment_method,omitempty"`
}

// CustomerList represents a list of customers
type CustomerList struct {
	Object  string     `json:"object"`
	URL     string     `json:"url"`
	HasMore bool       `json:"has_more"`
	Data    []Customer `json:"data"`
}

// DeletedCustomer represents a deleted customer
type DeletedCustomer struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}

// PaymentIntent represents a Stripe payment intent
type PaymentIntent struct {
	ID                     string            `json:"id"`
	Object                 string            `json:"object"`
	Amount                 int64             `json:"amount"`
	AmountCapturable       int64             `json:"amount_capturable"`
	AmountReceived         int64             `json:"amount_received"`
	CaptureMethod          string            `json:"capture_method"`
	ClientSecret           string            `json:"client_secret"`
	ConfirmationMethod     string            `json:"confirmation_method"`
	Created                int64             `json:"created"`
	Currency               string            `json:"currency"`
	Customer               string            `json:"customer,omitempty"`
	Description            string            `json:"description,omitempty"`
	Livemode               bool              `json:"livemode"`
	Metadata               map[string]string `json:"metadata,omitempty"`
	PaymentMethod          string            `json:"payment_method,omitempty"`
	PaymentMethodTypes     []string          `json:"payment_method_types"`
	ReceiptEmail           string            `json:"receipt_email,omitempty"`
	Status                 string            `json:"status"`
	StatementDescriptor    string            `json:"statement_descriptor,omitempty"`
}

// PaymentIntentParams represents parameters for creating/updating a payment intent
type PaymentIntentParams struct {
	Amount               int64             `json:"amount,omitempty"`
	Currency             string            `json:"currency,omitempty"`
	Customer             string            `json:"customer,omitempty"`
	Description          string            `json:"description,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	PaymentMethod        string            `json:"payment_method,omitempty"`
	PaymentMethodTypes   []string          `json:"payment_method_types,omitempty"`
	ReceiptEmail         string            `json:"receipt_email,omitempty"`
	StatementDescriptor  string            `json:"statement_descriptor,omitempty"`
	CaptureMethod        string            `json:"capture_method,omitempty"`
	ConfirmationMethod   string            `json:"confirmation_method,omitempty"`
	Confirm              bool              `json:"confirm,omitempty"`
	ReturnURL            string            `json:"return_url,omitempty"`
}

// PaymentIntentConfirmParams represents parameters for confirming a payment intent
type PaymentIntentConfirmParams struct {
	PaymentMethod string `json:"payment_method,omitempty"`
	ReturnURL     string `json:"return_url,omitempty"`
}

// PaymentIntentCaptureParams represents parameters for capturing a payment intent
type PaymentIntentCaptureParams struct {
	AmountToCapture int64 `json:"amount_to_capture,omitempty"`
}

// PaymentIntentList represents a list of payment intents
type PaymentIntentList struct {
	Object  string          `json:"object"`
	URL     string          `json:"url"`
	HasMore bool            `json:"has_more"`
	Data    []PaymentIntent `json:"data"`
}

// Subscription represents a Stripe subscription
type Subscription struct {
	ID                 string            `json:"id"`
	Object             string            `json:"object"`
	BillingCycleAnchor int64             `json:"billing_cycle_anchor"`
	CancelAt           int64             `json:"cancel_at,omitempty"`
	CancelAtPeriodEnd  bool              `json:"cancel_at_period_end"`
	CanceledAt         int64             `json:"canceled_at,omitempty"`
	CollectionMethod   string            `json:"collection_method"`
	Created            int64             `json:"created"`
	CurrentPeriodEnd   int64             `json:"current_period_end"`
	CurrentPeriodStart int64             `json:"current_period_start"`
	Customer           string            `json:"customer"`
	DaysUntilDue       int               `json:"days_until_due,omitempty"`
	DefaultPaymentMethod string          `json:"default_payment_method,omitempty"`
	EndedAt            int64             `json:"ended_at,omitempty"`
	Items              *SubscriptionItems `json:"items"`
	Livemode           bool              `json:"livemode"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	Status             string            `json:"status"`
	TrialEnd           int64             `json:"trial_end,omitempty"`
	TrialStart         int64             `json:"trial_start,omitempty"`
}

// SubscriptionItems represents subscription items
type SubscriptionItems struct {
	Object  string             `json:"object"`
	HasMore bool               `json:"has_more"`
	Data    []SubscriptionItem `json:"data"`
}

// SubscriptionItem represents a subscription item
type SubscriptionItem struct {
	ID           string            `json:"id"`
	Object       string            `json:"object"`
	Created      int64             `json:"created"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	Price        *Price            `json:"price"`
	Quantity     int64             `json:"quantity"`
	Subscription string            `json:"subscription"`
}

// SubscriptionParams represents parameters for creating/updating a subscription
type SubscriptionParams struct {
	Customer             string            `json:"customer,omitempty"`
	Items                []SubItemParams   `json:"items,omitempty"`
	Metadata             map[string]string `json:"metadata,omitempty"`
	DefaultPaymentMethod string            `json:"default_payment_method,omitempty"`
	CollectionMethod     string            `json:"collection_method,omitempty"`
	DaysUntilDue         int               `json:"days_until_due,omitempty"`
	TrialEnd             int64             `json:"trial_end,omitempty"`
	TrialPeriodDays      int               `json:"trial_period_days,omitempty"`
	CancelAtPeriodEnd    bool              `json:"cancel_at_period_end,omitempty"`
}

// SubItemParams represents subscription item parameters
type SubItemParams struct {
	Price    string `json:"price,omitempty"`
	Quantity int64  `json:"quantity,omitempty"`
}

// SubscriptionCancelParams represents parameters for canceling a subscription
type SubscriptionCancelParams struct {
	InvoiceNow bool `json:"invoice_now,omitempty"`
	Prorate    bool `json:"prorate,omitempty"`
}

// SubscriptionListParams represents parameters for listing subscriptions
type SubscriptionListParams struct {
	ListParams
	Customer string `json:"customer,omitempty"`
	Price    string `json:"price,omitempty"`
	Status   string `json:"status,omitempty"`
}

// SubscriptionList represents a list of subscriptions
type SubscriptionList struct {
	Object  string         `json:"object"`
	URL     string         `json:"url"`
	HasMore bool           `json:"has_more"`
	Data    []Subscription `json:"data"`
}

// Invoice represents a Stripe invoice
type Invoice struct {
	ID                 string            `json:"id"`
	Object             string            `json:"object"`
	AmountDue          int64             `json:"amount_due"`
	AmountPaid         int64             `json:"amount_paid"`
	AmountRemaining    int64             `json:"amount_remaining"`
	BillingReason      string            `json:"billing_reason,omitempty"`
	CollectionMethod   string            `json:"collection_method"`
	Created            int64             `json:"created"`
	Currency           string            `json:"currency"`
	Customer           string            `json:"customer"`
	CustomerEmail      string            `json:"customer_email,omitempty"`
	CustomerName       string            `json:"customer_name,omitempty"`
	Description        string            `json:"description,omitempty"`
	DueDate            int64             `json:"due_date,omitempty"`
	HostedInvoiceURL   string            `json:"hosted_invoice_url,omitempty"`
	InvoicePDF         string            `json:"invoice_pdf,omitempty"`
	Livemode           bool              `json:"livemode"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	Number             string            `json:"number,omitempty"`
	Paid               bool              `json:"paid"`
	PaidOutOfBand      bool              `json:"paid_out_of_band"`
	Status             string            `json:"status"`
	Subscription       string            `json:"subscription,omitempty"`
	Total              int64             `json:"total"`
}

// InvoiceParams represents parameters for creating/updating an invoice
type InvoiceParams struct {
	Customer           string            `json:"customer,omitempty"`
	Subscription       string            `json:"subscription,omitempty"`
	Description        string            `json:"description,omitempty"`
	Metadata           map[string]string `json:"metadata,omitempty"`
	CollectionMethod   string            `json:"collection_method,omitempty"`
	DaysUntilDue       int               `json:"days_until_due,omitempty"`
	DueDate            int64             `json:"due_date,omitempty"`
	AutoAdvance        bool              `json:"auto_advance,omitempty"`
}

// InvoiceListParams represents parameters for listing invoices
type InvoiceListParams struct {
	ListParams
	Customer     string `json:"customer,omitempty"`
	Subscription string `json:"subscription,omitempty"`
	Status       string `json:"status,omitempty"`
}

// InvoiceList represents a list of invoices
type InvoiceList struct {
	Object  string    `json:"object"`
	URL     string    `json:"url"`
	HasMore bool      `json:"has_more"`
	Data    []Invoice `json:"data"`
}

// Product represents a Stripe product
type Product struct {
	ID          string            `json:"id"`
	Object      string            `json:"object"`
	Active      bool              `json:"active"`
	Created     int64             `json:"created"`
	Description string            `json:"description,omitempty"`
	Images      []string          `json:"images,omitempty"`
	Livemode    bool              `json:"livemode"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Name        string            `json:"name"`
	Updated     int64             `json:"updated"`
}

// ProductParams represents parameters for creating/updating a product
type ProductParams struct {
	Active      *bool             `json:"active,omitempty"`
	Description string            `json:"description,omitempty"`
	Images      []string          `json:"images,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Name        string            `json:"name,omitempty"`
}

// ProductList represents a list of products
type ProductList struct {
	Object  string    `json:"object"`
	URL     string    `json:"url"`
	HasMore bool      `json:"has_more"`
	Data    []Product `json:"data"`
}

// DeletedProduct represents a deleted product
type DeletedProduct struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}

// Price represents a Stripe price
type Price struct {
	ID              string            `json:"id"`
	Object          string            `json:"object"`
	Active          bool              `json:"active"`
	BillingScheme   string            `json:"billing_scheme"`
	Created         int64             `json:"created"`
	Currency        string            `json:"currency"`
	Livemode        bool              `json:"livemode"`
	LookupKey       string            `json:"lookup_key,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	Nickname        string            `json:"nickname,omitempty"`
	Product         string            `json:"product"`
	Recurring       *Recurring        `json:"recurring,omitempty"`
	Type            string            `json:"type"`
	UnitAmount      int64             `json:"unit_amount,omitempty"`
	UnitAmountDecimal string          `json:"unit_amount_decimal,omitempty"`
}

// Recurring represents recurring price information
type Recurring struct {
	AggregateUsage string `json:"aggregate_usage,omitempty"`
	Interval       string `json:"interval"`
	IntervalCount  int    `json:"interval_count"`
	UsageType      string `json:"usage_type"`
}

// PriceParams represents parameters for creating/updating a price
type PriceParams struct {
	Active            *bool             `json:"active,omitempty"`
	Currency          string            `json:"currency,omitempty"`
	LookupKey         string            `json:"lookup_key,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	Nickname          string            `json:"nickname,omitempty"`
	Product           string            `json:"product,omitempty"`
	Recurring         *RecurringParams  `json:"recurring,omitempty"`
	UnitAmount        int64             `json:"unit_amount,omitempty"`
	UnitAmountDecimal string            `json:"unit_amount_decimal,omitempty"`
}

// RecurringParams represents recurring price parameters
type RecurringParams struct {
	Interval      string `json:"interval,omitempty"`
	IntervalCount int    `json:"interval_count,omitempty"`
	UsageType     string `json:"usage_type,omitempty"`
}

// PriceListParams represents parameters for listing prices
type PriceListParams struct {
	ListParams
	Active    *bool  `json:"active,omitempty"`
	Currency  string `json:"currency,omitempty"`
	Product   string `json:"product,omitempty"`
	Type      string `json:"type,omitempty"`
	LookupKey string `json:"lookup_key,omitempty"`
}

// PriceList represents a list of prices
type PriceList struct {
	Object  string  `json:"object"`
	URL     string  `json:"url"`
	HasMore bool    `json:"has_more"`
	Data    []Price `json:"data"`
}

// Address represents a billing/shipping address
type Address struct {
	City       string `json:"city,omitempty"`
	Country    string `json:"country,omitempty"`
	Line1      string `json:"line1,omitempty"`
	Line2      string `json:"line2,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	State      string `json:"state,omitempty"`
}

// AddressParams represents address parameters
type AddressParams struct {
	City       string `json:"city,omitempty"`
	Country    string `json:"country,omitempty"`
	Line1      string `json:"line1,omitempty"`
	Line2      string `json:"line2,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	State      string `json:"state,omitempty"`
}

// Shipping represents shipping information
type Shipping struct {
	Address *Address `json:"address,omitempty"`
	Name    string   `json:"name,omitempty"`
	Phone   string   `json:"phone,omitempty"`
}

// ShippingParams represents shipping parameters
type ShippingParams struct {
	Address *AddressParams `json:"address,omitempty"`
	Name    string         `json:"name,omitempty"`
	Phone   string         `json:"phone,omitempty"`
}

// ListParams represents common list parameters
type ListParams struct {
	Limit         int    `json:"limit,omitempty"`
	StartingAfter string `json:"starting_after,omitempty"`
	EndingBefore  string `json:"ending_before,omitempty"`
}

// APIError represents a Stripe API error
type APIError struct {
	Error struct {
		Type    string `json:"type"`
		Code    string `json:"code,omitempty"`
		Message string `json:"message"`
		Param   string `json:"param,omitempty"`
	} `json:"error"`
}

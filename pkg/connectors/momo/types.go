package momo

// PaymentType selects MoMo payment method: wallet-to-wallet (e-wallet) or credit card.
type PaymentType string

const (
	// PaymentTypeWallet is MoMo e-wallet (wallet-to-wallet); requestType "captureWallet".
	PaymentTypeWallet PaymentType = "wallet"
	// PaymentTypeCreditCard is credit/debit card; requestType "payWithCC".
	PaymentTypeCreditCard PaymentType = "credit_card"
)

// CreatePaymentRequest is the request body for POST /v2/gateway/api/create (one-time payment).
// See https://developers.momo.vn/v3/docs/payment/api/credit/onetime/ and wallet onetime.
type CreatePaymentRequest struct {
	PartnerCode  string   `json:"partnerCode"`
	PartnerName  string   `json:"partnerName,omitempty"`
	StoreId      string   `json:"storeId,omitempty"`
	RequestId    string   `json:"requestId"`
	Amount       int64    `json:"amount"`        // VND; limits depend on requestType
	OrderId      string   `json:"orderId"`
	OrderInfo    string   `json:"orderInfo"`
	RedirectUrl  string   `json:"redirectUrl"`
	IpnUrl       string   `json:"ipnUrl"`
	RequestType  string   `json:"requestType"`  // "payWithCC" or "captureWallet"
	ExtraData    string   `json:"extraData,omitempty"` // base64 JSON
	UserInfo     *UserInfo `json:"userInfo,omitempty"`
	AutoCapture  bool     `json:"autoCapture,omitempty"`
	Lang         string   `json:"lang,omitempty"` // vi | en
	Signature    string   `json:"signature"`
}

// UserInfo for create payment (email required for notification).
type UserInfo struct {
	Name        string `json:"name,omitempty"`
	PhoneNumber string `json:"phoneNumber,omitempty"`
	Email       string `json:"email"`
}

// CreatePaymentResponse is the response from create API.
// For wallet (captureWallet), MoMo may also return Deeplink, QrCodeUrl, DeeplinkMiniApp.
// See https://developers.momo.vn/v3/docs/payment/api/wallet/onetime
type CreatePaymentResponse struct {
	PartnerCode     string `json:"partnerCode"`
	RequestId       string `json:"requestId"`
	OrderId         string `json:"orderId"`
	Amount          int64  `json:"amount"`
	ResponseTime    int64  `json:"responseTime"`
	Message         string `json:"message"`
	ResultCode      int    `json:"resultCode"` // 0 = success
	PayUrl          string `json:"payUrl"`     // Redirect user to this URL (both wallet & credit)
	// Wallet-only (captureWallet): open MoMo app or show QR
	Deeplink        string `json:"deeplink,omitempty"`        // Open MoMo app directly
	QrCodeUrl       string `json:"qrCodeUrl,omitempty"`       // Data to generate QR (use external lib to render)
	DeeplinkMiniApp string `json:"deeplinkMiniApp,omitempty"` // When site is embedded in MoMo app
	Signature       string `json:"signature,omitempty"`
	UserFee         int64  `json:"userFee,omitempty"`
}

// CallbackPayload is the payload MoMo sends to redirectUrl (query params) and ipnUrl (body).
// Used to verify and confirm payment result.
type CallbackPayload struct {
	PartnerCode   string `json:"partnerCode"`
	RequestId     string `json:"requestId"`
	Amount        int64  `json:"amount"`
	OrderId       string `json:"orderId"`
	OrderType     string `json:"orderType"`   // momo_wallet
	OrderInfo     string `json:"orderInfo"`
	PartnerUserId string `json:"partnerUserId,omitempty"`
	PartnerClientId string `json:"partnerClientId,omitempty"`
	CallbackToken string `json:"callbackToken,omitempty"`
	TransId       int64  `json:"transId"`      // MoMo transaction ID
	ResultCode    int    `json:"resultCode"`  // 0 = success
	Message       string `json:"message"`
	PayType       string `json:"payType"`     // credit
	ResponseTime  int64  `json:"responseTime"`
	ExtraData     string `json:"extraData,omitempty"`
	Signature     string `json:"signature"`
}

// IpnResponse is what partner must return to MoMo when receiving IPN (HTTP 200 body).
type IpnResponse struct {
	PartnerCode  string `json:"partnerCode"`
	RequestId    string `json:"requestId"`
	OrderId      string `json:"orderId"`
	ResultCode   int    `json:"resultCode"`
	Message      string `json:"message"`
	ResponseTime int64  `json:"responseTime"`
	ExtraData    string `json:"extraData,omitempty"`
	Signature    string `json:"signature"`
}

// CreatePaymentParams is a convenience struct for creating a payment from app code.
type CreatePaymentParams struct {
	RequestId   string      // Unique per request (idempotency)
	OrderId     string      // Your order/transaction ID
	Amount      int64       // VND; limits: credit_card 1k–10M, wallet 1k–50M
	OrderInfo   string      // Description
	RedirectUrl string      // Where to redirect after payment (MoMo trả kết quả qua query params tại đây)
	IpnUrl      string      // Optional: server-to-server IPN URL; để "" nếu chỉ xử lý từ redirectUrl
	UserInfo    *UserInfo   // Optional; email recommended
	ExtraData   string      // Optional base64 JSON
	PaymentType PaymentType // Wallet (e-wallet) or CreditCard; default CreditCard
}

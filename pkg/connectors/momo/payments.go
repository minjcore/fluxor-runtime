package momo

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const (
	requestTypePayWithCC     = "payWithCC"     // credit/debit card
	requestTypeCaptureWallet = "captureWallet" // wallet-to-wallet (e-wallet)
)

// Amount limits per MoMo docs: credit 1k–10M VND, wallet 1k–50M VND.
const (
	amountMinVND           = 1000
	amountMaxCreditCardVND = 10_000_000
	amountMaxWalletVND     = 50_000_000
)

// Create creates a payment and returns payUrl for redirect.
// PaymentType defaults to CreditCard; set PaymentTypeWallet for e-wallet (wallet-to-wallet).
// See https://developers.momo.vn/v3/docs/payment/api/credit/onetime/ and wallet onetime.
func (p *paymentsClient) Create(ctx context.Context, params *CreatePaymentParams) (*CreatePaymentResponse, error) {
	if params == nil {
		return nil, fmt.Errorf("momo: params cannot be nil")
	}
	pt := params.PaymentType
	if pt == "" {
		pt = PaymentTypeCreditCard
	}
	if params.Amount < amountMinVND {
		return nil, fmt.Errorf("momo: amount must be at least %d VND", amountMinVND)
	}
	switch pt {
	case PaymentTypeWallet:
		if params.Amount > amountMaxWalletVND {
			return nil, fmt.Errorf("momo: wallet amount must be %d-%d VND", amountMinVND, amountMaxWalletVND)
		}
	case PaymentTypeCreditCard:
		fallthrough
	default:
		if params.Amount > amountMaxCreditCardVND {
			return nil, fmt.Errorf("momo: credit card amount must be %d-%d VND", amountMinVND, amountMaxCreditCardVND)
		}
	}
	if params.OrderId == "" || params.RequestId == "" || params.RedirectUrl == "" {
		return nil, fmt.Errorf("momo: orderId, requestId, redirectUrl are required")
	}
	ipnUrl := params.IpnUrl // optional: chỉ dùng redirectUrl thì để ""

	requestType := requestTypePayWithCC
	if pt == PaymentTypeWallet {
		requestType = requestTypeCaptureWallet
	}

	cfg := &p.client.config
	extraData := params.ExtraData
	if extraData == "" {
		extraData = ""
	}

	sig := SignCreatePayment(
		cfg.AccessKey,
		fmt.Sprintf("%d", params.Amount),
		extraData,
		ipnUrl,
		params.OrderId,
		params.OrderInfo,
		cfg.PartnerCode,
		params.RedirectUrl,
		params.RequestId,
		requestType,
		cfg.SecretKey,
	)

	req := &CreatePaymentRequest{
		PartnerCode:  cfg.PartnerCode,
		PartnerName:  cfg.PartnerName,
		StoreId:      cfg.StoreId,
		RequestId:    params.RequestId,
		Amount:       params.Amount,
		OrderId:      params.OrderId,
		OrderInfo:    params.OrderInfo,
		RedirectUrl:  params.RedirectUrl,
		IpnUrl:       ipnUrl,
		RequestType:  requestType,
		ExtraData:    extraData,
		UserInfo:     params.UserInfo,
		Lang:         cfg.Lang,
		Signature:    sig,
	}

	body, err := p.client.do(ctx, "POST", "/v2/gateway/api/create", req)
	if err != nil {
		return nil, err
	}

	var out CreatePaymentResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("momo: parse response: %w", err)
	}
	return &out, nil
}

// VerifyCallback verifies the signature of callback/IPN from MoMo (delegates to ValidateSignature).
func (p *paymentsClient) VerifyCallback(cb *CallbackPayload) bool {
	return ValidateSignature(cb, p.client.config.AccessKey, p.client.config.SecretKey)
}

// BuildIpnResponse builds the JSON body partner must return to MoMo when receiving IPN (HTTP 200).
// resultCode and message: partner's decision (e.g. 0 / "Success" if order is confirmed).
func (p *paymentsClient) BuildIpnResponse(cb *CallbackPayload, resultCode int, message string) *IpnResponse {
	if cb == nil {
		return nil
	}
	cfg := &p.client.config
	responseTime := time.Now().Unix()
	sig := SignIpnResponse(
		cfg.AccessKey,
		cb.ExtraData,
		message,
		cb.OrderId,
		cfg.PartnerCode,
		cb.RequestId,
		responseTime,
		resultCode,
		cfg.SecretKey,
	)
	return &IpnResponse{
		PartnerCode:  cfg.PartnerCode,
		RequestId:    cb.RequestId,
		OrderId:      cb.OrderId,
		ResultCode:   resultCode,
		Message:      message,
		ResponseTime: responseTime,
		ExtraData:    cb.ExtraData,
		Signature:    sig,
	}
}

// Package momo provides the MoMo Payment Gateway connector for Fluxor.
//
// MoMo (Vietnam e-wallet) integration: create one-time payments via wallet-to-wallet
// (PaymentTypeWallet) or credit card (PaymentTypeCreditCard), redirect user to payUrl,
// and verify IPN/callback with HMAC_SHA256 signature.
//
// Usage:
//
//	cfg := momo.DefaultConfig()
//	cfg.PartnerCode = "MOMO"
//	cfg.AccessKey = "..."
//	cfg.SecretKey = "..."
//	comp := momo.NewComponent(cfg)
//	_ = comp.Start(ctx)
//	payments, _ := comp.Payments()
//	res, _ := payments.Create(ctx, &momo.CreatePaymentParams{...})
//	// Redirect user to res.PayUrl
//
// Callback: receive POST at ipnUrl, parse body as momo.CallbackPayload,
// verify with payments.VerifyCallback(cb), then respond with payments.BuildIpnResponse(cb, 0, "Success").
//
// Docs: https://developers.momo.vn/v3/docs/payment/api/credit/onetime/
//
// Path: pkg/connectors/momo
package momo

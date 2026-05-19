# MoMo Payment Gateway Connector

Connector MoMo (ví điện tử Việt Nam) cho Fluxor: tạo thanh toán một lần **wallet-to-wallet** (e-wallet) hoặc **credit card**, redirect user tới `payUrl`, xác thực IPN/callback bằng chữ ký HMAC_SHA256.

**Payment types:**

- **Wallet (e-wallet)** — `momo.PaymentTypeWallet`: thanh toán qua ví MoMo (wallet-to-wallet). Giới hạn: 1.000–50.000.000 VND. Response có thể có thêm `Deeplink` (mở app MoMo), `QrCodeUrl` (tạo QR). [Docs](https://developers.momo.vn/v3/docs/payment/api/wallet/onetime)
- **Credit card** — `momo.PaymentTypeCreditCard` (mặc định): thanh toán thẻ tín dụng/ghi nợ. Giới hạn: 1.000–10.000.000 VND.

**Use case (all-in-one):** Chỉ cần connector này cho luồng thanh toán online — backend gọi Create → user mở `payUrl` (MoMo) → thanh toán xong → MoMo redirect về `redirectUrl` (nền tảng của bạn). User mở link trên platform của bạn, một luồng gọn.

## Cấu hình

Env hoặc config:

- `MOMO_PARTNER_CODE` — Mã đối tác (bắt buộc)
- `MOMO_ACCESS_KEY` — Access key từ M4B (bắt buộc)
- `MOMO_SECRET_KEY` — Secret key để ký HMAC (bắt buộc)
- `MOMO_BASE_URL` — Production: `https://payment.momo.vn`, Sandbox: `https://test-payment.momo.vn` (mặc định sandbox)
- `MOMO_TIMEOUT` — Timeout gọi API (mặc định 30s, MoMo khuyến nghị ≥ 30s)
- `MOMO_LANG` — `vi` hoặc `en`

## Sử dụng

```go
import "github.com/fluxorio/fluxor/pkg/connectors/momo"
import "github.com/fluxorio/fluxor/pkg/connectors"

cfg := momo.DefaultConfig()
cfg.PartnerCode = "MOMO"
cfg.AccessKey = "..."
cfg.SecretKey = "..."
comp := momo.NewComponent(cfg)
if err := comp.Start(ctx); err != nil {
    return err
}
defer comp.Stop(ctx)

payments, _ := comp.Payments()
res, err := payments.Create(ctx, &momo.CreatePaymentParams{
    RequestId:   "unique-request-id",
    OrderId:     "your-order-id",
    Amount:      50000, // VND
    OrderInfo:   "Thanh toan don hang",
    RedirectUrl: "https://yoursite.com/return",
    IpnUrl:      "https://yoursite.com/momo/ipn",
    UserInfo:    &momo.UserInfo{Email: "user@example.com"},
    PaymentType: momo.PaymentTypeCreditCard, // or momo.PaymentTypeWallet
})
if err != nil {
    return err
}
if res.ResultCode != 0 {
    return fmt.Errorf("momo: %s", res.Message)
}
// Redirect user to res.PayUrl
http.Redirect(w, r, res.PayUrl, http.StatusFound)
```

## redirectUrl nhận kết quả (không bắt buộc IPN)

MoMo **luôn trả kết quả qua redirectUrl**: khi redirect user về, MoMo thêm **query params** vào URL (cùng các field như `CallbackPayload`: `orderId`, `resultCode`, `amount`, `signature`, …). Server **không cần ipnUrl** vẫn có thể xử lý:

```go
// Create: ipnUrl có thể để "" nếu chỉ dùng redirect
res, _ := payments.Create(ctx, &momo.CreatePaymentParams{
    RequestId: "req-1", OrderId: "order-1", Amount: 50000,
    OrderInfo: "Thanh toan", RedirectUrl: "https://yoursite.com/pay/return",
    IpnUrl: "", // không dùng IPN
})

// Handler GET tại redirectUrl (https://yoursite.com/pay/return?orderId=...&resultCode=...&signature=...)
cb := momo.ParseRedirectQuery(r.URL.Query())
if cb == nil || !payments.VerifyCallback(cb) {
    http.Error(w, "Invalid", 400)
    return
}
if cb.ResultCode == 0 {
    // Thanh toán thành công — cập nhật đơn cb.OrderId, cb.TransId, cb.Amount
}
```

(IPN dùng khi cần nhận chắc chắn server-to-server, user đóng trình duyệt vẫn nhận.)

## IPN / Callback (tùy chọn)

Nhận POST tại `ipnUrl`, parse body JSON thành `momo.CallbackPayload`, xác thực chữ ký rồi trả lời MoMo.

```go
var cb momo.CallbackPayload
if err := json.NewDecoder(r.Body).Decode(&cb); err != nil {
    http.Error(w, "Bad Request", 400)
    return
}
payments, _ := momoComp.Payments()
if !payments.VerifyCallback(&cb) {
    http.Error(w, "Invalid signature", 400)
    return
}
if cb.ResultCode != 0 {
    // Giao dịch thất bại
}
// Cập nhật trạng thái đơn hàng (orderId, transId, amount)...

ipnResp := payments.BuildIpnResponse(&cb, 0, "Success")
w.Header().Set("Content-Type", "application/json; charset=UTF-8")
json.NewEncoder(w).Encode(ipnResp)
```

Lưu ý: Một số tài liệu MoMo yêu cầu trả HTTP 204 No Content; nếu dùng 204 thì không gửi body.

## Tài liệu MoMo

- [One-Time Payment (Credit Card)](https://developers.momo.vn/v3/docs/payment/api/credit/onetime/)
- [One-Time Payment (Wallet)](https://developers.momo.vn/v3/docs/payment/api/wallet/onetime/)
- [Payment Notification / IPN](https://developers.momo.vn/v3/docs/payment/api/result-handling/notification/)
- [Confirm / Query / Refund](https://developers.momo.vn/v3/docs/payment/api/payment-api/confirm/)

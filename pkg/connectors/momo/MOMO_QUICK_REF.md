# MoMo API — Quick reference (cho người tạo v1)

Một trang tóm tắt những gì cần nhớ từ docs MoMo, không cần đọc hết developers.momo.vn.

---

## 1. Create payment (lấy link thanh toán)

| | |
|---|---|
| **Method** | `POST` |
| **Sandbox** | `https://test-payment.momo.vn/v2/gateway/api/create` |
| **Production** | `https://payment.momo.vn/v2/gateway/api/create` |
| **Body** | JSON |

**Tham số bắt buộc:**

| Param | Kiểu | Ghi chú |
|-------|------|--------|
| `partnerCode` | string | Mã đối tác (M4B) |
| `requestId` | string | Unique mỗi request (idempotency) |
| `amount` | long | VND; credit: 1k–10M, wallet: 1k–50M |
| `orderId` | string | Mã đơn hàng của bạn |
| `orderInfo` | string | Mô tả đơn |
| `redirectUrl` | string | URL redirect sau khi user thanh toán xong |
| `ipnUrl` | string | URL nhận IPN (server-to-server) |
| `requestType` | string | `"payWithCC"` (credit card) hoặc `"captureWallet"` (wallet-to-wallet) |
| `signature` | string | HMAC_SHA256 (xem bên dưới) |

**Signature (create):**  
Chuỗi gõ theo thứ tự a-z, rồi HMAC_SHA256(chuỗi, secretKey):

```
accessKey=$accessKey&amount=$amount&extraData=$extraData&ipnUrl=$ipnUrl&orderId=$orderId&orderInfo=$orderInfo&partnerCode=$partnerCode&redirectUrl=$redirectUrl&requestId=$requestId&requestType=$requestType
```

**Response:**  
- `resultCode == 0` → thành công, dùng `payUrl` để redirect user.  
- `resultCode != 0` → xem `message`.

---

## 2. redirectUrl — MoMo trả kết quả qua query params

**Quan trọng:** MoMo **luôn** trả kết quả qua **redirectUrl**: khi redirect user về, MoMo **thêm query params** vào URL (GET). Các field giống `CallbackPayload`: `orderId`, `resultCode`, `amount`, `transId`, `signature`, `message`, …  

→ Server **không cần ipnUrl** vẫn xử lý được: handler GET tại `redirectUrl` → parse query → build `CallbackPayload` → `VerifyCallback(cb)`.

## 3. Callback / IPN (tùy chọn)

- **Redirect:** MoMo redirect user về `redirectUrl` (GET, params trên URL) — đủ để xử lý cho nhiều trường hợp.  
- **IPN:** MoMo gửi **POST** tới `ipnUrl`, body JSON — dùng khi cần chắc chắn server-to-server (user đóng browser vẫn nhận được).

**Body IPN gồm:**  
`partnerCode`, `requestId`, `orderId`, `amount`, `transId` (Momo transaction ID), `resultCode`, `message`, `signature`, … (đủ như struct `CallbackPayload` trong code).

**Verify:**  
Build lại chuỗi ký từ các field (key a-z): `accessKey`, `amount`, `callbackToken`, `extraData`, `message`, `orderId`, `orderInfo`, `orderType`, `partnerClientId`, `partnerCode`, `payType`, `requestId`, `responseTime`, `resultCode`, `transId` → HMAC_SHA256 so với `signature` MoMo gửi (dùng secretKey của bạn). Connector đã có `VerifyCallback(cb)`.

**Trả lời MoMo:**  
- Cách 1: HTTP 200, body JSON (partnerCode, requestId, orderId, resultCode, message, responseTime, extraData, signature). Connector có `BuildIpnResponse(...)`.  
- Cách 2: HTTP 204 No Content (một số doc MoMo ghi vậy).

---

## 4. Result codes (thường gặp)

| resultCode | Ý nghĩa |
|------------|--------|
| `0` | Giao dịch thành công |
| `9000` | Giao dịch được authorize (một số luồng) |
| Khác 0 | Lỗi — xem `message` (hoặc tra bảng error code đầy đủ trên MoMo). |

---

## 5. Credentials (M4B)

- **Partner Code** — mã đối tác  
- **Access Key** — dùng trong chuỗi ký  
- **Secret Key** — dùng để HMAC_SHA256, **không gửi lên MoMo**

---

## 6. Trong code connector này

- **Create:** `payments.Create(ctx, &CreatePaymentParams{...})` → trả `PayUrl`, `ResultCode`, `Message`.  
- **Verify callback:** `payments.VerifyCallback(&CallbackPayload)` → true/false.  
- **Trả lời IPN:** `payments.BuildIpnResponse(cb, resultCode, message)` → JSON body trả MoMo.

Chi tiết đầy đủ vẫn ở developers.momo.vn; file này chỉ giúp nhớ nhanh khi code.

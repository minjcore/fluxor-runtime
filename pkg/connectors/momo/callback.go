package momo

import (
	"crypto/subtle"
	"fmt"
	"net/url"
	"strconv"
)

// ParseRedirectQuery builds CallbackPayload from query params on redirectUrl.
// MoMo redirects GET to redirectUrl?orderId=...&resultCode=...&amount=...&signature=...
// Use after redirect; then validate with ValidateSignature(cb, accessKey, secretKey).
func ParseRedirectQuery(q url.Values) *CallbackPayload {
	if q == nil {
		return nil
	}
	amount, _ := strconv.ParseInt(q.Get("amount"), 10, 64)
	transId, _ := strconv.ParseInt(q.Get("transId"), 10, 64)
	resultCode, _ := strconv.Atoi(q.Get("resultCode"))
	responseTime, _ := strconv.ParseInt(q.Get("responseTime"), 10, 64)

	return &CallbackPayload{
		PartnerCode:     q.Get("partnerCode"),
		RequestId:       q.Get("requestId"),
		Amount:          amount,
		OrderId:         q.Get("orderId"),
		OrderType:       q.Get("orderType"),
		OrderInfo:       q.Get("orderInfo"),
		PartnerUserId:   q.Get("partnerUserId"),
		PartnerClientId: q.Get("partnerClientId"),
		CallbackToken:   q.Get("callbackToken"),
		TransId:         transId,
		ResultCode:      resultCode,
		Message:         q.Get("message"),
		PayType:         q.Get("payType"),
		ResponseTime:    responseTime,
		ExtraData:       q.Get("extraData"),
		Signature:       q.Get("signature"),
	}
}

// ValidateSignature verifies the HMAC_SHA256 signature of a callback (redirect or IPN).
// Use accessKey and secretKey from your MoMo config. Returns true only if the payload
// signature matches the recomputed one (constant-time comparison).
func ValidateSignature(cb *CallbackPayload, accessKey, secretKey string) bool {
	if cb == nil || secretKey == "" {
		return false
	}
	params := map[string]string{
		"accessKey":       accessKey,
		"amount":          fmt.Sprintf("%d", cb.Amount),
		"callbackToken":  cb.CallbackToken,
		"extraData":      cb.ExtraData,
		"message":        cb.Message,
		"orderId":        cb.OrderId,
		"orderInfo":      cb.OrderInfo,
		"orderType":      cb.OrderType,
		"partnerClientId": cb.PartnerClientId,
		"partnerCode":    cb.PartnerCode,
		"payType":        cb.PayType,
		"requestId":      cb.RequestId,
		"responseTime":   fmt.Sprintf("%d", cb.ResponseTime),
		"resultCode":     fmt.Sprintf("%d", cb.ResultCode),
		"transId":        fmt.Sprintf("%d", cb.TransId),
	}
	expected := SignCallback(params, secretKey)
	if len(expected) != len(cb.Signature) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(cb.Signature)) == 1
}

package momo

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// SignCreatePayment builds HMAC_SHA256 for create payment request.
// Format: accessKey=$accessKey&amount=$amount&extraData=$extraData&ipnUrl=$ipnUrl&orderId=$orderId&orderInfo=$orderInfo&partnerCode=$partnerCode&redirectUrl=$redirectUrl&requestId=$requestId&requestType=$requestType
// Keys sorted a-z. See https://developers.momo.vn/v3/docs/payment/api/credit/onetime/
func SignCreatePayment(accessKey, amount, extraData, ipnUrl, orderId, orderInfo, partnerCode, redirectUrl, requestId, requestType, secretKey string) string {
	raw := strings.Join([]string{
		"accessKey=" + accessKey,
		"amount=" + amount,
		"extraData=" + extraData,
		"ipnUrl=" + ipnUrl,
		"orderId=" + orderId,
		"orderInfo=" + orderInfo,
		"partnerCode=" + partnerCode,
		"redirectUrl=" + redirectUrl,
		"requestId=" + requestId,
		"requestType=" + requestType,
	}, "&")
	return hmacSHA256(raw, secretKey)
}

// SignCallback builds HMAC_SHA256 for callback/IPN verification (incoming from MoMo).
// Raw string: all keys a-z, key=value. See MoMo docs "Processing payment result".
func SignCallback(params map[string]string, secretKey string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	return hmacSHA256(strings.Join(parts, "&"), secretKey)
}

// SignIpnResponse builds HMAC_SHA256 for partner's IPN response to MoMo.
// Format: accessKey=$accessKey&extraData=$extraData&message=$message&orderId=$orderId&partnerCode=$partnerCode&requestId=$requestId&responseTime=$responseTime&resultCode=$resultCode
func SignIpnResponse(accessKey, extraData, message, orderId, partnerCode, requestId string, responseTime int64, resultCode int, secretKey string) string {
	raw := strings.Join([]string{
		"accessKey=" + accessKey,
		"extraData=" + extraData,
		"message=" + message,
		"orderId=" + orderId,
		"partnerCode=" + partnerCode,
		"requestId=" + requestId,
		"responseTime=" + formatInt64(responseTime),
		"resultCode=" + formatInt(resultCode),
	}, "&")
	return hmacSHA256(raw, secretKey)
}

func hmacSHA256(raw, secretKey string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(raw))
	return hex.EncodeToString(h.Sum(nil))
}

func formatInt64(n int64) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b) - 1
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		b[i] = byte('0' + n%10)
		n /= 10
		i--
	}
	if neg {
		b[i] = '-'
		i--
	}
	return string(b[i+1:])
}

func formatInt(n int) string {
	return formatInt64(int64(n))
}

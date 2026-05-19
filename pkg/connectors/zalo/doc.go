// Package zalo provides a Zalo connector for the Fluxor framework.
//
// It supports Zalo Official Account APIs including:
//
//   - OAuth: exchange code for token, refresh access token
//   - ZNS (Zalo Notification Service): gửi tin ZNS (templated notifications) tới số điện thoại,
//     xem thông tin template, xem quota. ZNS dùng template đã duyệt trên Zalo OA để gửi tin
//     nhắn (ví dụ: xác nhận đơn hàng, nhắc lịch, mã OTP). Tài liệu chính thức:
//     https://developers.zalo.me/docs/zalo-notification-service/gui-tin-zns/gui-zns
//
// Configuration uses environment variables: ZALO_ZNS_OA_ID, ZALO_ZNS_ACCESS_TOKEN,
// ZALO_ZNS_APP_ID, ZALO_ZNS_APP_SECRET, ZALO_ZNS_BASE_URL, ZALO_ZNS_TIMEOUT.
// Code is referenced from apps/fluxor-mail (ZNS client and zalo_zns utils).
//
// Token refresh uses a safe time window (refreshBuffer): the client refreshes when
// time until expiry is less than the buffer, so the token is never used after expiry.
// If you run an in-process cron to refresh or persist the token, run it at interval
// less than (token_ttl - refreshBuffer) so the valid window is never lost.
package zalo

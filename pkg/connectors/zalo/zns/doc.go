// Package zns provides a connector scaffold for Zalo Notification Service (ZNS).
//
// ZNS (Zalo Notification Service) cho phép gửi tin nhắn template tới người dùng qua Zalo OA:
// template do Zalo duyệt trước, gửi qua API với template_id và template_data. Dùng cho thông báo
// giao dịch, xác nhận đơn, nhắc lịch, OTP, v.v.
//
// Tài liệu gửi tin ZNS (Gửi ZNS):
// https://developers.zalo.me/docs/zalo-notification-service/gui-tin-zns/gui-zns
//
// This package contains a minimal Connector implementation; full HTTP client and send/template/quota
// APIs live in the parent package pkg/connectors/zalo (client.go, component.go).
//
// Path: pkg/connectors/zalo/zns
package zns


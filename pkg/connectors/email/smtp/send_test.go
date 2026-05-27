package smtp_test

import (
	"context"
	"testing"
	"time"

	"github.com/fluxorio/fluxor/pkg/connectors/email/smtp"
)

// TestSendReal gửi email thật qua Aliyun SMTPS (port 465).
// Chạy: go test ./pkg/connectors/email/smtp/ -run TestSendReal -v
func TestSendReal(t *testing.T) {
	cfg := smtp.Config{
		Host:     "smtpdm-ap-southeast-1.aliyuncs.com",
		Port:     465,
		User:     "otp@nivic.dev",
		Password: "EmailPassword10",
		From:     "otp@nivic.dev",
		FromName: "Nivic Dev",
		Timeout:  "30s",
	}

	client := smtp.NewClient(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := client.Send(ctx, smtp.SendInput{
		ToEmail: "caokhang91@gmail.com",
		Subject: "Test email từ Fluxor SMTP connector",
		Body:    `<h2>Hello from Fluxor!</h2><p>Email connector hoạt động tốt với Aliyun SMTPS port 465.</p>`,
	})

	if !result.Success {
		t.Fatalf("send failed: %s", result.Error)
	}
	t.Logf("OK: %s", result.Message)
}

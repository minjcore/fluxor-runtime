// Package smtp provides an SMTP (mail) connector for the Fluxor framework.
//
// It implements connectors.Connector with Send capability: send email via any
// SMTP server (Gmail, SendGrid, AWS SES SMTP, etc.). Configuration via env vars
// (SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD, SMTP_FROM, SMTP_FROM_NAME, SMTP_USE_TLS)
// or Config struct.
//
// Usage:
//
//	cfg := smtp.DefaultConfig()
//	cfg.Host = "smtp.example.com"
//	cfg.Port = 587
//	conn := smtp.NewMailConnector(cfg)
//	_ = conn.Start(ctx)
//	defer conn.Stop(ctx)
//	client, _ := conn.Client()
//	result := client.Send(ctx, smtp.SendInput{ToEmail: "user@example.com", Subject: "Hi", Body: "<p>Hello</p>"})
//
// Path: pkg/connectors/email/smtp
package smtp

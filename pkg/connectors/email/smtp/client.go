package smtp

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// Client is the SMTP mail client.
type Client struct {
	config Config
}

// NewClient creates a new SMTP client. Config should be validated before calling.
func NewClient(cfg Config) *Client {
	if cfg.Port <= 0 {
		cfg.Port = 587
	}
	if cfg.FromName == "" {
		cfg.FromName = "Fluxor Mail"
	}
	return &Client{config: cfg}
}

// Send sends an email. Returns SendResult with success or error details.
func (c *Client) Send(ctx context.Context, in SendInput) SendResult {
	if c.config.Host == "" {
		return SendResult{Success: false, Error: "smtp host not configured"}
	}
	if in.ToEmail == "" {
		return SendResult{Success: false, Error: "to_email is required"}
	}

	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	auth := smtp.PlainAuth("", c.config.User, c.config.Password, c.config.Host)

	from := c.config.From
	if from == "" {
		from = c.config.User
	}
	header := map[string]string{
		"From":         fmt.Sprintf("%s <%s>", c.config.FromName, from),
		"To":           in.ToEmail,
		"Subject":      in.Subject,
		"MIME-Version": "1.0",
		"Content-Type": "text/html; charset=UTF-8",
	}
	if in.RequestID != "" {
		header["X-Request-ID"] = in.RequestID
	}
	if in.ReplyTo != "" {
		header["Reply-To"] = in.ReplyTo
	}

	var sb strings.Builder
	for k, v := range header {
		sb.WriteString(k)
		sb.WriteString(": ")
		sb.WriteString(v)
		sb.WriteString("\r\n")
	}
	sb.WriteString("\r\n")
	sb.WriteString(in.Body)

	msg := []byte(sb.String())
	to := []string{in.ToEmail}
	if len(in.CC) > 0 {
		to = append(to, in.CC...)
	}
	if len(in.BCC) > 0 {
		to = append(to, in.BCC...)
	}

	timeout := c.config.GetTimeout()
	if deadline, ok := ctx.Deadline(); ok {
		if d := time.Until(deadline); d < timeout {
			timeout = d
		}
	}

	done := make(chan error, 1)
	go func() {
		done <- smtp.SendMail(addr, auth, from, to, msg)
	}()

	select {
	case <-ctx.Done():
		return SendResult{Success: false, Error: ctx.Err().Error()}
	case err := <-done:
		if err != nil {
			return SendResult{Success: false, Message: "send failed", Error: err.Error()}
		}
		return SendResult{Success: true, Message: "sent"}
	}
}

# SMTP (Mail) Connector

Connector for sending email via SMTP. Implements `connectors.Connector`.

## Config

| Field    | Env             | Description                |
|----------|-----------------|----------------------------|
| Host     | SMTP_HOST       | SMTP server host           |
| Port     | SMTP_PORT       | Port (default 587)         |
| User     | SMTP_USER       | Username                   |
| Password | SMTP_PASSWORD   | Password                   |
| From     | SMTP_FROM       | From address               |
| FromName | SMTP_FROM_NAME  | From display name          |
| UseTLS   | SMTP_USE_TLS     | Use TLS (default true)     |
| Timeout  | SMTP_TIMEOUT     | Send timeout (default 30s) |

## Usage

```go
cfg := smtp.DefaultConfig()
cfg.Host = "smtp.gmail.com"
cfg.Port = 587
cfg.User = "user"
cfg.Password = "app-password"
cfg.From = "noreply@example.com"

conn := smtp.NewMailConnector(cfg)
if err := conn.Start(ctx); err != nil {
    return err
}
defer conn.Stop(ctx)

client, _ := conn.Client()
result := client.Send(ctx, smtp.SendInput{
    ToEmail: "recipient@example.com",
    Subject: "Subject",
    Body:    "<p>HTML body</p>",
    RequestID: "req-123",
})
if !result.Success {
    return fmt.Errorf("send failed: %s", result.Error)
}
```

## Integration

Used by `apps/fluxor-mail` (publish-subscribe notification app). The app can optionally use this package instead of its local mail connector for consistency across the repo.

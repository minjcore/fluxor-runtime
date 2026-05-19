// Package gcp provides GCP (Google Cloud Platform) infrastructure integration.
//
// Implemented:
//   - Secret Manager: read/write secrets (cloud.google.com/go/secretmanager)
//   - Cloud Logging: structured logging to Google Cloud (cloud.google.com/go/logging)
//
// Example (Secret Manager):
//
//	cfg := gcp.DefaultSecretConfig("") // uses GCP_PROJECT env if empty
//	sm, err := gcp.NewSecretManager(ctx, cfg)
//	if err != nil { ... }
//	defer sm.Close()
//	val, err := sm.AccessSecret(ctx, "my-secret")
//
// Example (Cloud Logging):
//
//	cfg := gcp.DefaultLoggingConfig("") // uses GCP_PROJECT, GCP_LOG_NAME env
//	client, err := gcp.NewLoggingClient(ctx, cfg)
//	if err != nil { ... }
//	defer client.Close()
//	client.Write(ctx, []gcp.LogEntry{{Severity: "INFO", Message: "hello", Timestamp: time.Now()}})
//
// Example (log to Google when profile is enabled — e.g. gke-autopilot):
//
//	baseLogger := core.NewLogger(core.LoggerConfig{...})
//	if profile == "gke-autopilot" && os.Getenv("GCP_PROJECT") != "" {
//	    logger, closeFn, _ := gcp.NewLoggerWithCloud(ctx, baseLogger, gcp.DefaultLoggingConfig(""))
//	    defer closeFn()
//	    // use logger for app (logs go to stdout/file and to Cloud Logging)
//	} else {
//	    logger = baseLogger
//	}
package gcp

// Package downloader provides a common interface and implementations for downloading
// files from URLs (HTTP/HTTPS and extensible to other schemes).
//
// Use the Downloader interface for dependency injection and testing; use NewHTTP or
// Default() for a ready-to-use HTTP downloader.
//
// Example:
//
//	d := downloader.Default()
//	err := d.Download(ctx, "https://example.com/file.zip", "/tmp/file.zip")
//
// With options:
//
//	d := downloader.NewHTTP(downloader.HTTPOptions{Timeout: 30 * time.Second})
//	err := d.Download(ctx, url, dest)
package downloader

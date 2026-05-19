package downloader

import "context"

// Downloader downloads a resource from a URL to a destination path or returns bytes.
type Downloader interface {
	// Download fetches the resource at url and writes it to destPath.
	// The caller is responsible for creating the parent directory if needed.
	Download(ctx context.Context, url, destPath string) error
	// DownloadBytes fetches the resource at url and returns the body (caller closes not needed; in-memory).
	DownloadBytes(ctx context.Context, url string) ([]byte, error)
}

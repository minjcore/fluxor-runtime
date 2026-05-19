package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// HTTPOptions configures the HTTP downloader.
type HTTPOptions struct {
	Timeout    time.Duration     // Request timeout (0 = default 60s)
	Client     *http.Client      // Custom client (nil = use default with Timeout)
	Header     http.Header       // Optional request headers
}

// HTTP implements Downloader using HTTP/HTTPS.
type HTTP struct {
	client *http.Client
	header http.Header
}

// NewHTTP creates an HTTP downloader with the given options.
func NewHTTP(opts HTTPOptions) *HTTP {
	client := opts.Client
	if client == nil {
		timeout := opts.Timeout
		if timeout == 0 {
			timeout = 60 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}
	return &HTTP{
		client: client,
		header: opts.Header,
	}
}

// Default returns a default HTTP downloader (60s timeout).
func Default() Downloader {
	return NewHTTP(HTTPOptions{})
}

// Download implements Downloader.
func (h *HTTP) Download(ctx context.Context, url, destPath string) error {
	if url == "" || destPath == "" {
		return fmt.Errorf("downloader: url and destPath are required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("downloader: new request: %w", err)
	}
	for k, v := range h.header {
		req.Header[k] = append(req.Header[k], v...)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("downloader: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloader: HTTP %d", resp.StatusCode)
	}
	dir := filepath.Dir(destPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("downloader: mkdir: %w", err)
		}
	}
	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("downloader: create file: %w", err)
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return fmt.Errorf("downloader: write: %w", err)
	}
	return nil
}

// DownloadBytes implements Downloader.
func (h *HTTP) DownloadBytes(ctx context.Context, url string) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("downloader: url is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("downloader: new request: %w", err)
	}
	for k, v := range h.header {
		req.Header[k] = append(req.Header[k], v...)
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloader: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloader: HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

package io

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fluxorio/fluxor/pkg/core/failfast"
)

// DownloadFile downloads a file from a URL and saves it to the specified path.
// Fail-fast: Validates inputs before downloading
func DownloadFile(url, path string) error {
	return DownloadFileWithContext(context.Background(), url, path)
}

// DownloadFileWithContext downloads a file from a URL with context support.
// Fail-fast: Validates inputs before downloading
func DownloadFileWithContext(ctx context.Context, url, path string) error {
	failfast.If(url != "", "URL cannot be empty")
	failfast.If(path != "", "file path cannot be empty")

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if err := WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write downloaded file: %w", err)
	}

	return nil
}

// DownloadFileWithClient downloads a file using a custom HTTP client.
// Fail-fast: Validates inputs before downloading
func DownloadFileWithClient(client *http.Client, url, path string) error {
	return DownloadFileWithClientAndContext(context.Background(), client, url, path)
}

// DownloadFileWithClientAndContext downloads a file using a custom HTTP client with context.
// Fail-fast: Validates inputs before downloading
func DownloadFileWithClientAndContext(ctx context.Context, client *http.Client, url, path string) error {
	failfast.If(url != "", "URL cannot be empty")
	failfast.If(path != "", "file path cannot be empty")
	failfast.NotNil(client, "HTTP client")

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if err := WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write downloaded file: %w", err)
	}

	return nil
}

// ReadHTTPResponse reads the entire body of an HTTP response.
// Fail-fast: Validates response before reading
func ReadHTTPResponse(resp *http.Response) ([]byte, error) {
	failfast.NotNil(resp, "HTTP response")
	failfast.NotNil(resp.Body, "HTTP response body")

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read HTTP response body: %w", err)
	}

	return data, nil
}

// ReadHTTPResponseString reads the entire body of an HTTP response as a string.
// Fail-fast: Validates response before reading
func ReadHTTPResponseString(resp *http.Response) (string, error) {
	data, err := ReadHTTPResponse(resp)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteHTTPRequestBody writes data to an HTTP request body.
// Fail-fast: Validates inputs before writing
func WriteHTTPRequestBody(req *http.Request, data []byte) error {
	failfast.NotNil(req, "HTTP request")
	failfast.NotNil(data, "request body data")

	req.Body = io.NopCloser(bytes.NewReader(data))
	req.ContentLength = int64(len(data))

	return nil
}

// WriteHTTPRequestBodyString writes a string to an HTTP request body.
// Fail-fast: Validates inputs before writing
func WriteHTTPRequestBodyString(req *http.Request, body string) error {
	return WriteHTTPRequestBody(req, []byte(body))
}

// WriteHTTPRequestBodyReader writes data from a reader to an HTTP request body.
// Fail-fast: Validates inputs before writing
func WriteHTTPRequestBodyReader(req *http.Request, reader io.Reader) error {
	failfast.NotNil(req, "HTTP request")
	failfast.NotNil(reader, "reader")

	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read from reader: %w", err)
	}

	return WriteHTTPRequestBody(req, data)
}

// CopyHTTPResponse copies the HTTP response body to a writer.
// Fail-fast: Validates inputs before copying
func CopyHTTPResponse(dst io.Writer, resp *http.Response) (int64, error) {
	failfast.NotNil(dst, "destination writer")
	failfast.NotNil(resp, "HTTP response")
	failfast.NotNil(resp.Body, "HTTP response body")

	n, err := io.Copy(dst, resp.Body)
	if err != nil {
		return n, fmt.Errorf("failed to copy HTTP response body: %w", err)
	}

	return n, nil
}

// CopyHTTPResponseWithProgress copies the HTTP response body to a writer with progress tracking.
// Fail-fast: Validates inputs before copying
func CopyHTTPResponseWithProgress(dst io.Writer, resp *http.Response, progress func(bytes int64)) (int64, error) {
	failfast.NotNil(dst, "destination writer")
	failfast.NotNil(resp, "HTTP response")
	failfast.NotNil(resp.Body, "HTTP response body")

	// Use a progress writer if callback is provided
	var writer io.Writer = dst
	if progress != nil {
		writer = &progressWriter{
			writer:   dst,
			progress: progress,
		}
	}

	n, err := io.Copy(writer, resp.Body)
	if err != nil {
		return n, fmt.Errorf("failed to copy HTTP response body: %w", err)
	}

	return n, nil
}

// progressWriter wraps a writer and calls a progress callback
type progressWriter struct {
	writer   io.Writer
	progress func(bytes int64)
	written  int64
}

func (pw *progressWriter) Write(p []byte) (n int, err error) {
	n, err = pw.writer.Write(p)
	pw.written += int64(n)
	if pw.progress != nil {
		pw.progress(pw.written)
	}
	return n, err
}

// GetHTTP downloads content from a URL using GET method.
// Fail-fast: Validates URL before downloading
func GetHTTP(url string) ([]byte, error) {
	return GetHTTPWithContext(context.Background(), url)
}

// GetHTTPWithContext downloads content from a URL using GET method with context.
// Fail-fast: Validates URL before downloading
func GetHTTPWithContext(ctx context.Context, url string) ([]byte, error) {
	failfast.If(url != "", "URL cannot be empty")

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET request failed: HTTP %d", resp.StatusCode)
	}

	return ReadHTTPResponse(resp)
}

// PostHTTP sends a POST request with body and returns the response.
// Fail-fast: Validates inputs before posting
func PostHTTP(url string, body []byte, headers map[string]string) ([]byte, error) {
	return PostHTTPWithContext(context.Background(), url, body, headers)
}

// PostHTTPWithContext sends a POST request with body and context, returns the response.
// Fail-fast: Validates inputs before posting
func PostHTTPWithContext(ctx context.Context, url string, body []byte, headers map[string]string) ([]byte, error) {
	failfast.If(url != "", "URL cannot be empty")
	failfast.NotNil(body, "request body")

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to POST %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("POST request failed: HTTP %d", resp.StatusCode)
	}

	return ReadHTTPResponse(resp)
}

// PutHTTP sends a PUT request with body and returns the response.
// Fail-fast: Validates inputs before putting
func PutHTTP(url string, body []byte, headers map[string]string) ([]byte, error) {
	return PutHTTPWithContext(context.Background(), url, body, headers)
}

// PutHTTPWithContext sends a PUT request with body and context, returns the response.
// Fail-fast: Validates inputs before putting
func PutHTTPWithContext(ctx context.Context, url string, body []byte, headers map[string]string) ([]byte, error) {
	failfast.If(url != "", "URL cannot be empty")
	failfast.NotNil(body, "request body")

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to PUT %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("PUT request failed: HTTP %d", resp.StatusCode)
	}

	return ReadHTTPResponse(resp)
}

// DeleteHTTP sends a DELETE request and returns the response.
// Fail-fast: Validates URL before deleting
func DeleteHTTP(url string, headers map[string]string) ([]byte, error) {
	return DeleteHTTPWithContext(context.Background(), url, headers)
}

// DeleteHTTPWithContext sends a DELETE request with context and returns the response.
// Fail-fast: Validates URL before deleting
func DeleteHTTPWithContext(ctx context.Context, url string, headers map[string]string) ([]byte, error) {
	failfast.If(url != "", "URL cannot be empty")

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to DELETE %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("DELETE request failed: HTTP %d", resp.StatusCode)
	}

	return ReadHTTPResponse(resp)
}

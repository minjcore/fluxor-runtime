package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// FetchFileRaw retrieves a single file's raw content from GitHub using the Contents API.
// It supports private repos via PAT in Authorization header.
// Returns bytes and the detected content type from GitHub (if any).
func FetchFileRaw(cfg *AppConfig, owner, repo, path, ref string) ([]byte, string, error) {
	if owner == "" || repo == "" || path == "" {
		return nil, "", fmt.Errorf("fail-fast: owner/repo/path are required")
	}

	base := cfg.GitHub.BaseURL
	if base == "" {
		base = "https://api.github.com"
	}

	// Build URL: /repos/{owner}/{repo}/contents/{path}?ref=branch
	escapedPath := url.PathEscape(path)
	endpoint := fmt.Sprintf("%s/repos/%s/%s/contents/%s", base, owner, repo, escapedPath)
	if ref != "" {
		endpoint += "?ref=" + url.QueryEscape(ref)
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, "", fmt.Errorf("request build error: %w", err)
	}

	// Ask for raw content directly
	req.Header.Set("Accept", "application/vnd.github.v3.raw")
	ua := cfg.GitHub.UserAgent
	if ua == "" {
		ua = "playfluxor-gitshare"
	}
	req.Header.Set("User-Agent", ua)
	if cfg.GitHub.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.GitHub.Token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("github request error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, "", fmt.Errorf("file not found")
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, "", fmt.Errorf("unauthorized: ensure GITHUB_TOKEN has repo scope")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, "", fmt.Errorf("github error: status=%d body=%s", resp.StatusCode, string(b))
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read error: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "text/plain; charset=utf-8"
	}
	return data, ct, nil
}

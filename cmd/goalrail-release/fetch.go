package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

func fetchSource(ctx context.Context, rawURL string, maxBytes int64) ([]byte, error) {
	return fetchSourceWithClient(ctx, &http.Client{Timeout: 2 * time.Minute}, rawURL, maxBytes)
}

func fetchSourceWithClient(ctx context.Context, client *http.Client, rawURL string, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return nil, fmt.Errorf("source byte limit must be positive")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return nil, fmt.Errorf("source URL %q is not HTTPS", rawURL)
	}
	if client == nil {
		return nil, fmt.Errorf("source HTTP client is required")
	}
	origin := parsed.Host
	bounded := *client
	bounded.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) >= 3 || request.URL.Scheme != "https" || request.URL.Host != origin {
			return fmt.Errorf("refuse source redirect outside %s", origin)
		}
		return nil
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create source request: %w", err)
	}
	response, err := bounded.Do(request)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", rawURL, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: HTTP %d", rawURL, response.StatusCode)
	}
	if response.ContentLength > maxBytes {
		return nil, fmt.Errorf("fetch %s: declared size exceeds %d bytes", rawURL, maxBytes)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", rawURL, err)
	}
	if int64(len(raw)) > maxBytes {
		return nil, fmt.Errorf("fetch %s: payload exceeds %d bytes", rawURL, maxBytes)
	}
	return raw, nil
}

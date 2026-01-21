// Package api provides HTTP client and API interactions for Hytale services.
package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

const (
	userAgent  = "Go-http-client/2.0"
	maxRetries = 3
	retryDelay = time.Second
)

// Client is an HTTP client with retry logic and logging.
type Client struct {
	http   *http.Client
	logger zerolog.Logger
}

// NewClient creates a new API client.
func NewClient(logger zerolog.Logger) *Client {
	return &Client{
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// NewClientWithTimeout creates a new API client with custom timeout.
func NewClientWithTimeout(logger zerolog.Logger, timeout time.Duration) *Client {
	return &Client{
		http: &http.Client{
			Timeout: timeout,
		},
		logger: logger,
	}
}

// Do executes an HTTP request with retry logic.
func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", userAgent)

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Debug().
				Int("attempt", attempt+1).
				Msg("Retrying request")

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay * time.Duration(attempt)):
			}
		}

		c.logger.Debug().
			Str("method", req.Method).
			Str("url", req.URL.String()).
			Msg("Sending request")

		resp, err := c.http.Do(req.WithContext(ctx))
		if err != nil {
			lastErr = err
			continue
		}

		// Don't retry on client errors (4xx) except 429
		if resp.StatusCode >= 400 && resp.StatusCode < 500 && resp.StatusCode != 429 {
			return resp, nil
		}

		// Retry on server errors (5xx) and rate limiting (429)
		if resp.StatusCode >= 500 || resp.StatusCode == 429 {
			c.logger.Warn().
				Int("status", resp.StatusCode).
				Msg("Request failed, will retry")
			resp.Body.Close()
			lastErr = fmt.Errorf("request failed with status %d", resp.StatusCode)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(ctx, req)
}

// GetWithAuth performs an authenticated GET request.
func (c *Client) GetWithAuth(ctx context.Context, url, accessToken string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	return c.Do(ctx, req)
}

// GetJSON performs a GET request and reads the response body.
func (c *Client) GetJSON(ctx context.Context, url string) ([]byte, error) {
	resp, err := c.Get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// GetJSONWithAuth performs an authenticated GET request and reads the response body.
func (c *Client) GetJSONWithAuth(ctx context.Context, url, accessToken string) ([]byte, error) {
	resp, err := c.GetWithAuth(ctx, url, accessToken)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

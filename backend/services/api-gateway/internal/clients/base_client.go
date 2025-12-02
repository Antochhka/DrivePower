package clients

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"
)

// HTTPDoer defines http.Client interface subset.
type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}

// BaseClient provides simple GET/POST helpers.
type BaseClient struct {
	baseURL string
	client  HTTPDoer
}

// NewBaseClient builds client with base URL.
func NewBaseClient(baseURL string, client HTTPDoer) *BaseClient {
	return &BaseClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
	}
}

func (c *BaseClient) buildURL(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return c.baseURL + path
}

// Do executes HTTP request and returns status/body.
func (c *BaseClient) Do(ctx context.Context, method, path string, body []byte, headers map[string]string) (int, []byte, error) {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.buildURL(path), reader)
	if err != nil {
		return 0, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, respBody, nil
}

// NewDefaultHTTPClient returns *http.Client with timeout.
func NewDefaultHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout}
}

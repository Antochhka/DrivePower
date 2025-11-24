package clients

import (
	"context"
	"net/http"
)

// AuthClient proxies auth-service endpoints.
type AuthClient struct {
	base *BaseClient
}

// NewAuthClient returns client.
func NewAuthClient(baseURL string, httpClient HTTPDoer) *AuthClient {
	return &AuthClient{base: NewBaseClient(baseURL, httpClient)}
}

// Signup forwards signup payload.
func (c *AuthClient) Signup(ctx context.Context, body []byte) (int, []byte, error) {
	return c.base.Do(ctx, http.MethodPost, "/auth/signup", body, nil)
}

// Login forwards login payload.
func (c *AuthClient) Login(ctx context.Context, body []byte) (int, []byte, error) {
	return c.base.Do(ctx, http.MethodPost, "/auth/login", body, nil)
}

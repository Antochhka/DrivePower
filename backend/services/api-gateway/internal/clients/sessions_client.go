package clients

import (
	"context"
	"net/http"
	"strconv"
)

// SessionsClient proxies calls to sessions-service.
type SessionsClient struct {
	base *BaseClient
}

// NewSessionsClient returns client.
func NewSessionsClient(baseURL string, httpClient HTTPDoer) *SessionsClient {
	return &SessionsClient{base: NewBaseClient(baseURL, httpClient)}
}

// GetSessionsForUser fetches history for given user.
func (c *SessionsClient) GetSessionsForUser(ctx context.Context, userID int64) (int, []byte, error) {
	headers := map[string]string{
		"X-User-ID": strconv.FormatInt(userID, 10),
	}
	return c.base.Do(ctx, http.MethodGet, "/sessions/me", nil, headers)
}

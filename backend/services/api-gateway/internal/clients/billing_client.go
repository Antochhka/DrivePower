package clients

import (
	"context"
	"net/http"
	"strconv"
)

// BillingClient proxies requests to billing-service.
type BillingClient struct {
	base *BaseClient
}

// NewBillingClient returns client instance.
func NewBillingClient(baseURL string, httpClient HTTPDoer) *BillingClient {
	return &BillingClient{base: NewBaseClient(baseURL, httpClient)}
}

// GetTransactionsForUser fetches billing history.
func (c *BillingClient) GetTransactionsForUser(ctx context.Context, userID int64) (int, []byte, error) {
	headers := map[string]string{
		"X-User-ID": strconv.FormatInt(userID, 10),
	}
	return c.base.Do(ctx, http.MethodGet, "/billing/me/transactions", nil, headers)
}

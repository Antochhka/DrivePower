package clients

import (
	"context"
	"net/http"
)

// StationsClient fetches station information from upstream (sessions/ocpp).
type StationsClient struct {
	base *BaseClient
}

// NewStationsClient returns client.
func NewStationsClient(baseURL string, httpClient HTTPDoer) *StationsClient {
	return &StationsClient{base: NewBaseClient(baseURL, httpClient)}
}

// ListStations fetches upstream data.
func (c *StationsClient) ListStations(ctx context.Context) (int, []byte, error) {
	return c.base.Do(ctx, http.MethodGet, "/stations", nil, nil)
}


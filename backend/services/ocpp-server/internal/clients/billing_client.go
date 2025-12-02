package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// BillingClient notifies billing-service about session lifecycle.
type BillingClient struct {
	baseURL string
	client  *http.Client
	logger  *zap.Logger
}

// BillingStopRequest payload for stop event (uses billing-service OCPP handler).
type BillingStopRequest struct {
	SessionID int64   `json:"session_id"`
	UserID    int64   `json:"user_id"`
	EnergyKWh float64 `json:"energy_kwh"`
}

// NewBillingClient returns HTTP client wrapper.
func NewBillingClient(baseURL string, logger *zap.Logger) *BillingClient {
	return &BillingClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
	}
}

// NotifySessionStop best-effort call.
func (c *BillingClient) NotifySessionStop(ctx context.Context, req BillingStopRequest) error {
	if c.baseURL == "" {
		c.logger.Debug("billing client disabled, skip stop notification")
		return nil
	}
	return c.post(ctx, "/internal/ocpp/session-stopped", req)
}

func (c *BillingClient) post(ctx context.Context, path string, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s%s", c.baseURL, path), bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Warn("billing client request failed", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		c.logger.Warn("billing client returned non-success", zap.Int("status", resp.StatusCode))
	}
	return nil
}

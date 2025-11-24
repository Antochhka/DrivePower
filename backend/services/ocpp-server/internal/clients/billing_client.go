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

// BillingStartRequest request payload.
type BillingStartRequest struct {
	StationID    string `json:"station_id"`
	ConnectorID  int    `json:"connector_id"`
	TransactionID string `json:"transaction_id"`
	MeterStart   int64  `json:"meter_start"`
}

// BillingStopRequest payload for stop event.
type BillingStopRequest struct {
	TransactionID string `json:"transaction_id"`
	MeterStop     int64  `json:"meter_stop"`
	Reason        string `json:"reason"`
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

// NotifySessionStart best-effort call to billing service.
func (c *BillingClient) NotifySessionStart(ctx context.Context, req BillingStartRequest) error {
	if c.baseURL == "" {
		c.logger.Debug("billing client disabled, skip start notification")
		return nil
	}
	return c.post(ctx, "/billing/ocpp/session/start", req)
}

// NotifySessionStop best-effort call.
func (c *BillingClient) NotifySessionStop(ctx context.Context, req BillingStopRequest) error {
	if c.baseURL == "" {
		c.logger.Debug("billing client disabled, skip stop notification")
		return nil
	}
	return c.post(ctx, "/billing/ocpp/session/stop", req)
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


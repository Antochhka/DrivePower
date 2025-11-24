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

// SessionsClient notifies sessions-service about lifecycle events.
type SessionsClient struct {
	baseURL string
	client  *http.Client
	logger  *zap.Logger
}

// StartSessionRequest minimal payload for start notification.
type StartSessionRequest struct {
	StationID    string `json:"station_id"`
	ConnectorID  int    `json:"connector_id"`
	TransactionID string `json:"transaction_id"`
	MeterStart   int64  `json:"meter_start"`
}

// StopSessionRequest minimal payload when transaction ends.
type StopSessionRequest struct {
	TransactionID string `json:"transaction_id"`
	MeterStop     int64  `json:"meter_stop"`
	Reason        string `json:"reason"`
}

// NewSessionsClient builds HTTP client wrapper.
func NewSessionsClient(baseURL string, logger *zap.Logger) *SessionsClient {
	return &SessionsClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
	}
}

// CreateFromOCPP notifies about session start (best-effort).
func (c *SessionsClient) CreateFromOCPP(ctx context.Context, req StartSessionRequest) error {
	if c.baseURL == "" {
		c.logger.Debug("sessions client disabled, skipping start notification")
		return nil
	}
	return c.post(ctx, "/sessions/ocpp/start", req)
}

// CompleteFromOCPP notifies about session completion.
func (c *SessionsClient) CompleteFromOCPP(ctx context.Context, req StopSessionRequest) error {
	if c.baseURL == "" {
		c.logger.Debug("sessions client disabled, skipping stop notification")
		return nil
	}
	return c.post(ctx, "/sessions/ocpp/stop", req)
}

func (c *SessionsClient) post(ctx context.Context, path string, body interface{}) error {
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
		c.logger.Warn("sessions client request failed", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		c.logger.Warn("sessions client returned non-success", zap.Int("status", resp.StatusCode))
	}
	return nil
}


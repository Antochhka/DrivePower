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

// TelemetryClient notifies telemetry-service about meter values.
type TelemetryClient struct {
	baseURL string
	client  *http.Client
	logger  *zap.Logger
}

// MeterValueRequest payload for telemetry-service.
type MeterValueRequest struct {
	SessionID   int64     `json:"session_id"`
	StationID   string    `json:"station_id"`
	ConnectorID int       `json:"connector_id"`
	MeterValue  float64   `json:"meter_value"`
	Unit        string    `json:"unit"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewTelemetryClient returns client wrapper.
func NewTelemetryClient(baseURL string, logger *zap.Logger) *TelemetryClient {
	return &TelemetryClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
	}
}

// NotifyMeterValue sends meter value to telemetry-service.
func (c *TelemetryClient) NotifyMeterValue(ctx context.Context, req MeterValueRequest) error {
	if c.baseURL == "" {
		c.logger.Debug("telemetry client disabled, skipping meter value")
		return nil
	}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/internal/ocpp/meter-values", c.baseURL), bytes.NewReader(data))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		c.logger.Warn("telemetry client request failed", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		c.logger.Warn("telemetry client returned non-success", zap.Int("status", resp.StatusCode))
	}
	return nil
}

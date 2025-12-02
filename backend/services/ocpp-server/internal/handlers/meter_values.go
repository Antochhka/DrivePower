package handlers

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"

	"drivepower/backend/services/ocpp-server/internal/clients"
	"drivepower/backend/services/ocpp-server/internal/ocpp"
	"drivepower/backend/services/ocpp-server/internal/ocpp/protocol"
	"drivepower/backend/services/ocpp-server/internal/service"
)

// NewMeterValuesHandler routes meter values to telemetry-service.
func NewMeterValuesHandler(telemetry *clients.TelemetryClient, txStore *service.TransactionStore, logger *zap.Logger) ocpp.HandlerFunc {
	return func(ctx context.Context, stationID string, payload json.RawMessage) (interface{}, error) {
		req, err := ocpp.Decode[protocol.MeterValuesRequest](payload)
		if err != nil {
			return nil, err
		}

		// find session id from transaction context
		txCtx, ok := txStore.Get(req.TransactionID)
		if !ok || txCtx.SessionID == 0 {
			logger.Warn("meter values without session context", zap.String("transaction_id", req.TransactionID))
			return nil, nil
		}

		if telemetry != nil {
			_ = telemetry.NotifyMeterValue(ctx, clients.MeterValueRequest{
				SessionID:   txCtx.SessionID,
				StationID:   req.StationID,
				ConnectorID: req.ConnectorID,
				MeterValue:  req.MeterValue,
				Unit:        "kWh",
				Timestamp:   req.Timestamp.UTC(),
			})
		}

		// update station state heartbeat-like timestamp if needed
		if req.Timestamp.IsZero() {
			req.Timestamp = time.Now().UTC()
		}

		return nil, nil
	}
}

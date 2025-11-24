package handlers

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"drivepower/backend/services/ocpp-server/internal/clients"
	"drivepower/backend/services/ocpp-server/internal/ocpp"
	"drivepower/backend/services/ocpp-server/internal/ocpp/protocol"
	"drivepower/backend/services/ocpp-server/internal/service"
)

// NewStartTransactionHandler notifies dependent services about start event.
func NewStartTransactionHandler(
	sessions *clients.SessionsClient,
	billing *clients.BillingClient,
	state *service.StationState,
	logger *zap.Logger,
) ocpp.HandlerFunc {
	return func(ctx context.Context, stationID string, payload []byte) (interface{}, error) {
		req, err := ocpp.Decode[protocol.StartTransactionRequest](payload)
		if err != nil {
			return nil, err
		}

		transactionID := req.TransactionID
		if transactionID == "" {
			transactionID = fmt.Sprintf("%s-%d", stationID, time.Now().UnixNano())
		}

		if sessions != nil {
			if err := sessions.CreateFromOCPP(ctx, clients.StartSessionRequest{
				StationID:     stationID,
				ConnectorID:   req.ConnectorID,
				TransactionID: transactionID,
				MeterStart:    req.MeterStart,
			}); err != nil {
				logger.Warn("sessions start notification failed", zap.String("station_id", stationID), zap.Error(err))
			}
		}

		if billing != nil {
			if err := billing.NotifySessionStart(ctx, clients.BillingStartRequest{
				StationID:     stationID,
				ConnectorID:   req.ConnectorID,
				TransactionID: transactionID,
				MeterStart:    req.MeterStart,
			}); err != nil {
				logger.Warn("billing start notification failed", zap.String("station_id", stationID), zap.Error(err))
			}
		}

		if req.ConnectorID > 0 {
			state.UpdateConnector(stationID, req.ConnectorID, protocol.ConnectorCharging)
		}

		resp := protocol.StartTransactionResponse{
			TransactionID: transactionID,
		}
		resp.IdTagInfo.Status = "Accepted"
		return resp, nil
	}
}

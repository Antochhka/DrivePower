package handlers

import (
	"context"

	"go.uber.org/zap"

	"drivepower/backend/services/ocpp-server/internal/clients"
	"drivepower/backend/services/ocpp-server/internal/ocpp"
	"drivepower/backend/services/ocpp-server/internal/ocpp/protocol"
	"drivepower/backend/services/ocpp-server/internal/service"
)

// NewStopTransactionHandler notifies dependent services about stop event.
func NewStopTransactionHandler(
	sessions *clients.SessionsClient,
	billing *clients.BillingClient,
	state *service.StationState,
	logger *zap.Logger,
) ocpp.HandlerFunc {
	return func(ctx context.Context, stationID string, payload []byte) (interface{}, error) {
		req, err := ocpp.Decode[protocol.StopTransactionRequest](payload)
		if err != nil {
			return nil, err
		}

		if sessions != nil {
			if err := sessions.CompleteFromOCPP(ctx, clients.StopSessionRequest{
				TransactionID: req.TransactionID,
				MeterStop:     req.MeterStop,
				Reason:        req.Reason,
			}); err != nil {
				logger.Warn("sessions stop notification failed", zap.String("station_id", stationID), zap.Error(err))
			}
		}

		if billing != nil {
			if err := billing.NotifySessionStop(ctx, clients.BillingStopRequest{
				TransactionID: req.TransactionID,
				MeterStop:     req.MeterStop,
				Reason:        req.Reason,
			}); err != nil {
				logger.Warn("billing stop notification failed", zap.String("station_id", stationID), zap.Error(err))
			}
		}

		state.UpdateStation(stationID, protocol.ConnectorAvailable)

		return protocol.StopTransactionResponse{}, nil
	}
}

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

// NewStopTransactionHandler notifies dependent services about stop event.
func NewStopTransactionHandler(
	sessions *clients.SessionsClient,
	billing *clients.BillingClient,
	state *service.StationState,
	txStore *service.TransactionStore,
	logger *zap.Logger,
) ocpp.HandlerFunc {
	return func(ctx context.Context, stationID string, payload json.RawMessage) (interface{}, error) {
		req, err := ocpp.Decode[protocol.StopTransactionRequest](payload)
		if err != nil {
			return nil, err
		}

		var energyKWh float64
		var sessionID int64
		if ctxInfo, ok := txStore.Get(req.TransactionID); ok {
			sessionID = ctxInfo.SessionID
			if req.MeterStop > ctxInfo.MeterStart {
				energyKWh = float64(req.MeterStop-ctxInfo.MeterStart) / 1000.0
			}
			txStore.Delete(req.TransactionID)
		}

		if sessions != nil {
			if err := sessions.CompleteFromOCPP(ctx, clients.StopSessionRequest{
				TransactionID: req.TransactionID,
				MeterStop:     req.MeterStop,
				Reason:        req.Reason,
				EnergyKWh:     energyKWh,
				EndTime:       time.Now().UTC(),
			}); err != nil {
				logger.Warn("sessions stop notification failed", zap.String("station_id", stationID), zap.Error(err))
			}
		}

		if billing != nil {
			if sessionID > 0 {
				if err := billing.NotifySessionStop(ctx, clients.BillingStopRequest{
					SessionID: sessionID,
					UserID:    0,
					EnergyKWh: energyKWh,
				}); err != nil {
					logger.Warn("billing stop notification failed", zap.String("station_id", stationID), zap.Error(err))
				}
			}
		}

		state.UpdateStation(stationID, protocol.ConnectorAvailable)

		return protocol.StopTransactionResponse{}, nil
	}
}

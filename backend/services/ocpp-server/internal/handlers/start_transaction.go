package handlers

import (
	"context"
	"encoding/json"
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
	txStore *service.TransactionStore,
	logger *zap.Logger,
) ocpp.HandlerFunc {
	return func(ctx context.Context, stationID string, payload json.RawMessage) (interface{}, error) {
		req, err := ocpp.Decode[protocol.StartTransactionRequest](payload)
		if err != nil {
			return nil, err
		}

		transactionID := req.TransactionID
		if transactionID == "" {
			transactionID = fmt.Sprintf("%s-%d", stationID, time.Now().UnixNano())
		}

		var sessionID int64
		if sessions != nil {
			sessionID, err = sessions.CreateFromOCPP(ctx, clients.StartSessionRequest{
				StationID:     stationID,
				ConnectorID:   req.ConnectorID,
				TransactionID: transactionID,
				MeterStart:    req.MeterStart,
			})
			if err != nil {
				logger.Warn("sessions start notification failed", zap.String("station_id", stationID), zap.Error(err))
			}
		}

		if req.ConnectorID > 0 {
			state.UpdateConnector(stationID, req.ConnectorID, protocol.ConnectorCharging)
		}

		txStore.Set(transactionID, service.TransactionContext{
			SessionID: sessionID,
			MeterStart: req.MeterStart,
		})

		resp := protocol.StartTransactionResponse{
			TransactionID: transactionID,
		}
		resp.IdTagInfo.Status = "Accepted"
		return resp, nil
	}
}

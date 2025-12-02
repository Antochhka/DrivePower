package handlers

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	"drivepower/backend/services/ocpp-server/internal/ocpp"
	"drivepower/backend/services/ocpp-server/internal/ocpp/protocol"
	"drivepower/backend/services/ocpp-server/internal/repository"
	"drivepower/backend/services/ocpp-server/internal/service"
)

// NewStatusNotificationHandler updates station/connector status.
func NewStatusNotificationHandler(repo *repository.StationRepository, state *service.StationState, logger *zap.Logger) ocpp.HandlerFunc {
	return func(ctx context.Context, stationID string, payload json.RawMessage) (interface{}, error) {
		req, err := ocpp.Decode[protocol.StatusNotificationRequest](payload)
		if err != nil {
			return nil, err
		}

		if req.ConnectorStatus == "" {
			req.ConnectorStatus = protocol.ConnectorAvailable
		}

		if err := repo.UpdateStatus(ctx, stationID, req.ConnectorStatus); err != nil {
			logger.Warn("failed to update station status", zap.String("station_id", stationID), zap.Error(err))
		}
		state.UpdateStation(stationID, req.ConnectorStatus)
		if req.ConnectorID > 0 {
			state.UpdateConnector(stationID, req.ConnectorID, req.ConnectorStatus)
		}

		return protocol.StatusNotificationResponse{}, nil
	}
}

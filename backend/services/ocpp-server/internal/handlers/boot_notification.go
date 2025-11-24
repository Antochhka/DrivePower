package handlers

import (
	"context"
	"time"

	"go.uber.org/zap"

	"drivepower/backend/services/ocpp-server/internal/models"
	"drivepower/backend/services/ocpp-server/internal/ocpp"
	"drivepower/backend/services/ocpp-server/internal/ocpp/protocol"
	"drivepower/backend/services/ocpp-server/internal/repository"
	"drivepower/backend/services/ocpp-server/internal/service"
)

// NewBootNotificationHandler registers handler.
func NewBootNotificationHandler(repo *repository.StationRepository, state *service.StationState, logger *zap.Logger) ocpp.HandlerFunc {
	return func(ctx context.Context, stationID string, payload []byte) (interface{}, error) {
		req, err := ocpp.Decode[protocol.BootNotificationRequest](payload)
		if err != nil {
			return nil, err
		}

		station := &models.Station{
			ID:              stationID,
			Vendor:          req.ChargePointVendor,
			Model:           req.ChargePointModel,
			FirmwareVersion: req.FirmwareVersion,
			Status:          protocol.ConnectorAvailable,
			LastHeartbeat:   time.Now().UTC(),
		}

		if err := repo.Upsert(ctx, station); err != nil {
			logger.Error("failed to upsert station", zap.String("station_id", stationID), zap.Error(err))
			return nil, err
		}

		state.UpdateStation(stationID, protocol.ConnectorAvailable)

		resp := protocol.BootNotificationResponse{
			CurrentTime: time.Now().UTC(),
			Interval:    30,
			Status:      protocol.RegistrationAccepted,
		}
		return resp, nil
	}
}


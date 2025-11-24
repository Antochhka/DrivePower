package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"drivepower/backend/services/telemetry-service/internal/models"
	"drivepower/backend/services/telemetry-service/internal/repository"
)

// TelemetryService handles persistence and aggregation.
type TelemetryService struct {
	repo      *repository.TelemetryRepository
	view      *repository.SessionEnergyView
	logger    *zap.Logger
}

// MeterValueInput represents payload from OCPP server.
type MeterValueInput struct {
	SessionID   int64     `json:"session_id"`
	StationID   string    `json:"station_id"`
	ConnectorID int       `json:"connector_id"`
	MeterValue  float64   `json:"meter_value"`
	Unit        string    `json:"unit"`
	Timestamp   time.Time `json:"timestamp"`
}

// NewTelemetryService returns service instance.
func NewTelemetryService(repo *repository.TelemetryRepository, view *repository.SessionEnergyView, logger *zap.Logger) *TelemetryService {
	return &TelemetryService{
		repo:   repo,
		view:   view,
		logger: logger,
	}
}

// StoreMeterValue persists incoming meter data.
func (s *TelemetryService) StoreMeterValue(ctx context.Context, input MeterValueInput) error {
	if input.Timestamp.IsZero() {
		input.Timestamp = time.Now().UTC()
	}
	data := &models.TelemetryData{
		SessionID:   input.SessionID,
		StationID:   input.StationID,
		ConnectorID: input.ConnectorID,
		MeterValue:  input.MeterValue,
		Unit:        input.Unit,
		RecordedAt:  input.Timestamp.UTC(),
	}
	if err := s.repo.Insert(ctx, data); err != nil {
		return err
	}
	return nil
}

// TotalEnergy returns total energy consumption for session.
func (s *TelemetryService) TotalEnergy(ctx context.Context, sessionID int64) (float64, error) {
	if s.view != nil {
		if total, err := s.view.GetTotalEnergy(ctx, sessionID); err == nil {
			return total, nil
		}
	}
	return s.repo.SumEnergyBySession(ctx, sessionID)
}


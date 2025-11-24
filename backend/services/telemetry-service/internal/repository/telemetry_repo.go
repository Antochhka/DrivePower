package repository

import (
	"context"
	"database/sql"
	"time"

	"drivepower/backend/services/telemetry-service/internal/models"
)

// TelemetryRepository persists meter readings.
type TelemetryRepository struct {
	db *sql.DB
}

// NewTelemetryRepository returns repository.
func NewTelemetryRepository(db *sql.DB) *TelemetryRepository {
	return &TelemetryRepository{db: db}
}

// Insert stores new telemetry entry.
func (r *TelemetryRepository) Insert(ctx context.Context, data *models.TelemetryData) error {
	const query = `
		INSERT INTO telemetry_data (session_id, station_id, connector_id, meter_value, unit, recorded_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		RETURNING id, created_at
	`
	return r.db.QueryRowContext(ctx, query,
		data.SessionID,
		data.StationID,
		data.ConnectorID,
		data.MeterValue,
		data.Unit,
		data.RecordedAt,
	).Scan(&data.ID, &data.CreatedAt)
}

// SumEnergyBySession returns accumulated meter difference (assuming incremental meter values).
func (r *TelemetryRepository) SumEnergyBySession(ctx context.Context, sessionID int64) (float64, error) {
	const query = `
		SELECT COALESCE(MAX(meter_value) - MIN(meter_value), 0)
		FROM telemetry_data
		WHERE session_id = $1
	`
	var total float64
	if err := r.db.QueryRowContext(ctx, query, sessionID).Scan(&total); err != nil {
		return 0, err
	}
	if total < 0 {
		total = 0
	}
	return total, nil
}

// LastMeterValue returns the latest meter reading for delta calculations.
func (r *TelemetryRepository) LastMeterValue(ctx context.Context, sessionID int64) (float64, time.Time, error) {
	const query = `
		SELECT meter_value, recorded_at
		FROM telemetry_data
		WHERE session_id = $1
		ORDER BY recorded_at DESC
		LIMIT 1
	`
	var (
		value float64
		ts    time.Time
	)
	err := r.db.QueryRowContext(ctx, query, sessionID).Scan(&value, &ts)
	if err != nil {
		return 0, time.Time{}, err
	}
	return value, ts, nil
}


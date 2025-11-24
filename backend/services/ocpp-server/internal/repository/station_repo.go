package repository

import (
	"context"
	"database/sql"
	"time"

	"drivepower/backend/services/ocpp-server/internal/models"
)

// StationRepository manages charging station persistence.
type StationRepository struct {
	db *sql.DB
}

// NewStationRepository returns repository.
func NewStationRepository(db *sql.DB) *StationRepository {
	return &StationRepository{db: db}
}

// Upsert stores or updates station metadata.
func (r *StationRepository) Upsert(ctx context.Context, station *models.Station) error {
	const query = `
		INSERT INTO charging_stations (id, vendor, model, firmware_version, status, last_heartbeat, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			vendor = EXCLUDED.vendor,
			model = EXCLUDED.model,
			firmware_version = EXCLUDED.firmware_version,
			status = EXCLUDED.status,
			last_heartbeat = EXCLUDED.last_heartbeat,
			updated_at = NOW()
	`
	if station.LastHeartbeat.IsZero() {
		station.LastHeartbeat = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, query,
		station.ID,
		station.Vendor,
		station.Model,
		station.FirmwareVersion,
		station.Status,
		station.LastHeartbeat,
	)
	return err
}

// UpdateStatus changes station status and heartbeat.
func (r *StationRepository) UpdateStatus(ctx context.Context, stationID, status string) error {
	const query = `
		UPDATE charging_stations
		SET status = $2,
		    last_heartbeat = NOW(),
		    updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, stationID, status)
	return err
}


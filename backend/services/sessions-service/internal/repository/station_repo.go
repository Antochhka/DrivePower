package repository

import (
	"context"
	"database/sql"

	"drivepower/backend/services/sessions-service/internal/models"
)

// StationRepository stores metadata about stations (optional for joins / caching).
type StationRepository struct {
	db *sql.DB
}

// NewStationRepository returns repository.
func NewStationRepository(db *sql.DB) *StationRepository {
	return &StationRepository{db: db}
}

// Upsert persists station info if provided.
func (r *StationRepository) Upsert(ctx context.Context, station *models.Station) error {
	const query = `
		INSERT INTO stations (id, name, location, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			location = EXCLUDED.location,
			updated_at = NOW()
	`
	_, err := r.db.ExecContext(ctx, query, station.ID, station.Name, station.Location)
	return err
}


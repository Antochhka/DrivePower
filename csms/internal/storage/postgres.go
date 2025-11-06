package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStationRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	return pgxpool.New(ctx, dsn)
}

func NewPostgresStationRepository(pool *pgxpool.Pool) *PostgresStationRepository {
	return &PostgresStationRepository{pool: pool}
}

func (r *PostgresStationRepository) UpsertBoot(ctx context.Context, info StationBootInfo) error {
	if info.StationID == "" {
		return fmt.Errorf("station id is required")
	}

	_, err := r.pool.Exec(ctx, `
        INSERT INTO stations (station_id, vendor, model, boot_reason, last_seen_at)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (station_id)
        DO UPDATE SET vendor = EXCLUDED.vendor,
                      model = EXCLUDED.model,
                      boot_reason = EXCLUDED.boot_reason,
                      last_seen_at = EXCLUDED.last_seen_at
    `, info.StationID, info.Vendor, info.Model, info.Reason, info.Time)
	return err
}

func (r *PostgresStationRepository) UpdateLastSeen(ctx context.Context, stationID string, ts time.Time) error {
	if stationID == "" {
		return fmt.Errorf("station id is required")
	}

	_, err := r.pool.Exec(ctx, `
        INSERT INTO stations (station_id, last_seen_at)
        VALUES ($1, $2)
        ON CONFLICT (station_id)
        DO UPDATE SET last_seen_at = EXCLUDED.last_seen_at
    `, stationID, ts)
	return err
}

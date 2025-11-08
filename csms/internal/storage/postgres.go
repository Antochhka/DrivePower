package storage

import (
	"context"
	"fmt"
	"log"
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

	log.Printf("Persisting BootNotification: station=%s vendor=%s model=%s reason=%s at=%s\n",
		info.StationID,
		info.Vendor,
		info.Model,
		info.Reason,
		info.Time.Format(time.RFC3339))

	_, err := r.pool.Exec(ctx, `
        INSERT INTO stations (station_id, vendor, model, boot_reason, last_seen_at)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (station_id)
        DO UPDATE SET vendor = EXCLUDED.vendor,
                      model = EXCLUDED.model,
                      boot_reason = EXCLUDED.boot_reason,
                      last_seen_at = EXCLUDED.last_seen_at
    `, info.StationID, info.Vendor, info.Model, info.Reason, info.Time)
	if err != nil {
		log.Printf("BootNotification persist failed: station=%s err=%v\n", info.StationID, err)
	} else {
		log.Printf("BootNotification persisted: station=%s\n", info.StationID)
	}
	return err
}

func (r *PostgresStationRepository) UpdateLastSeen(ctx context.Context, stationID string, ts time.Time) error {
	if stationID == "" {
		return fmt.Errorf("station id is required")
	}

	log.Printf("Updating last_seen_at: station=%s timestamp=%s\n", stationID, ts.Format(time.RFC3339))

	_, err := r.pool.Exec(ctx, `
        INSERT INTO stations (station_id, last_seen_at)
        VALUES ($1, $2)
        ON CONFLICT (station_id)
        DO UPDATE SET last_seen_at = EXCLUDED.last_seen_at
    `, stationID, ts)
	if err != nil {
		log.Printf("last_seen_at persist failed: station=%s err=%v\n", stationID, err)
	} else {
		log.Printf("last_seen_at persisted: station=%s\n", stationID)
	}
	return err
}

func (r *PostgresStationRepository) UpsertConnectorStatus(ctx context.Context, status ConnectorStatusRecord) error {
	if status.StationID == "" {
		return fmt.Errorf("station id is required")
	}
	if status.EVSEID <= 0 {
		return fmt.Errorf("evse id must be positive")
	}
	if status.ConnectorID <= 0 {
		return fmt.Errorf("connector id must be positive")
	}
	if status.ConnectorStatus == "" {
		return fmt.Errorf("connector status is required")
	}

	log.Printf("Persisting connector status: station=%s evse=%d connector=%d status=%s timestamp=%s recorded=%s\n",
		status.StationID,
		status.EVSEID,
		status.ConnectorID,
		status.ConnectorStatus,
		status.StatusTimestamp.Format(time.RFC3339),
		status.RecordedAt.Format(time.RFC3339))

	_, err := r.pool.Exec(ctx, `
        INSERT INTO station_connector_statuses (
            station_id,
            evse_id,
            connector_id,
            connector_status,
            evse_status,
            connector_type,
            reason_code,
            vendor_id,
            vendor_description,
            status_timestamp,
            recorded_at
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
        ON CONFLICT (station_id, evse_id, connector_id)
        DO UPDATE SET connector_status = EXCLUDED.connector_status,
                      evse_status = EXCLUDED.evse_status,
                      connector_type = EXCLUDED.connector_type,
                      reason_code = EXCLUDED.reason_code,
                      vendor_id = EXCLUDED.vendor_id,
                      vendor_description = EXCLUDED.vendor_description,
                      status_timestamp = EXCLUDED.status_timestamp,
                      recorded_at = EXCLUDED.recorded_at
    `,
		status.StationID,
		status.EVSEID,
		status.ConnectorID,
		status.ConnectorStatus,
		nullIfEmpty(status.EVSEStatus),
		nullIfEmpty(status.ConnectorType),
		nullIfEmpty(status.ReasonCode),
		nullIfEmpty(status.VendorID),
		nullIfEmpty(status.VendorDescription),
		status.StatusTimestamp,
		status.RecordedAt,
	)
	if err != nil {
		log.Printf("Connector status persist failed: station=%s evse=%d connector=%d err=%v\n",
			status.StationID,
			status.EVSEID,
			status.ConnectorID,
			err)
	} else {
		log.Printf("Connector status persisted: station=%s evse=%d connector=%d\n",
			status.StationID,
			status.EVSEID,
			status.ConnectorID)
	}
	return err
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

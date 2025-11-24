package repository

import (
	"context"
	"database/sql"
)

// OCPPLogRepository stores raw OCPP messages.
type OCPPLogRepository struct {
	db *sql.DB
}

// NewOCPPLogRepository ctor.
func NewOCPPLogRepository(db *sql.DB) *OCPPLogRepository {
	return &OCPPLogRepository{db: db}
}

// Save stores log entry.
func (r *OCPPLogRepository) Save(ctx context.Context, stationID, direction, messageType string, payload []byte) error {
	const query = `
		INSERT INTO ocpp_messages (station_id, direction, message_type, payload)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.db.ExecContext(ctx, query, stationID, direction, messageType, payload)
	return err
}


package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"drivepower/backend/services/sessions-service/internal/models"
)

// SessionRepository handles persistence of charging sessions.
type SessionRepository struct {
	db *sql.DB
}

// NewSessionRepository returns repository.
func NewSessionRepository(db *sql.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// StartSession either creates a new session or updates existing by transaction id.
func (r *SessionRepository) StartSession(ctx context.Context, session *models.Session) (*models.Session, error) {
	const query = `
		INSERT INTO charging_sessions (user_id, station_id, connector_id, status, start_time, transaction_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		ON CONFLICT (transaction_id) DO UPDATE SET
			user_id = EXCLUDED.user_id,
			station_id = EXCLUDED.station_id,
			connector_id = EXCLUDED.connector_id,
			status = EXCLUDED.status,
			start_time = EXCLUDED.start_time,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		session.UserID,
		session.StationID,
		session.ConnectorID,
		session.Status,
		session.StartTime,
		session.Transaction,
	).Scan(&session.ID, &session.CreatedAt, &session.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return session, nil
}

// CompleteSession finalizes session by transaction id.
func (r *SessionRepository) CompleteSession(ctx context.Context, transactionID string, endTime time.Time, energy float64, status string) error {
	const query = `
		UPDATE charging_sessions
		SET end_time = $2,
		    energy_kwh = $3,
		    status = $4,
		    updated_at = NOW()
		WHERE transaction_id = $1
	`
	result, err := r.db.ExecContext(ctx, query, transactionID, endTime, energy, status)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// GetSessionsByUser returns last N sessions for user.
func (r *SessionRepository) GetSessionsByUser(ctx context.Context, userID int64, limit int) ([]models.Session, error) {
	if limit <= 0 {
		limit = 50
	}
	const query = `
		SELECT id, user_id, station_id, connector_id, status, start_time, end_time, energy_kwh, transaction_id, created_at, updated_at
		FROM charging_sessions
		WHERE user_id = $1
		ORDER BY start_time DESC
		LIMIT $2
	`
	rows, err := r.db.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var s models.Session
		if err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.StationID,
			&s.ConnectorID,
			&s.Status,
			&s.StartTime,
			&s.EndTime,
			&s.EnergyKWh,
			&s.Transaction,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}

// ErrSessionNotFound indicates missing transaction.
var ErrSessionNotFound = errors.New("session not found")

// GetActiveSessions returns currently active sessions.
func (r *SessionRepository) GetActiveSessions(ctx context.Context, limit int) ([]models.Session, error) {
	if limit <= 0 {
		limit = 50
	}
	const query = `
		SELECT id, user_id, station_id, connector_id, status, start_time, end_time, energy_kwh, transaction_id, created_at, updated_at
		FROM charging_sessions
		WHERE status = 'active'
		ORDER BY start_time DESC
		LIMIT $1
	`
	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []models.Session
	for rows.Next() {
		var s models.Session
		if err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.StationID,
			&s.ConnectorID,
			&s.Status,
			&s.StartTime,
			&s.EndTime,
			&s.EnergyKWh,
			&s.Transaction,
			&s.CreatedAt,
			&s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}

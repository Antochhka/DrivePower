package models

import "time"

// Session represents a charging session.
type Session struct {
	ID          int64     `db:"id" json:"id"`
	UserID      int64     `db:"user_id" json:"user_id"`
	StationID   string    `db:"station_id" json:"station_id"`
	ConnectorID int       `db:"connector_id" json:"connector_id"`
	Status      string    `db:"status" json:"status"`
	StartTime   time.Time `db:"start_time" json:"start_time"`
	EndTime     time.Time `db:"end_time" json:"end_time"`
	EnergyKWh   float64   `db:"energy_kwh" json:"energy_kwh"`
	Transaction string    `db:"transaction_id" json:"transaction_id"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}


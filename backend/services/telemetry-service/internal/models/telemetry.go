package models

import "time"

// TelemetryData represents a single meter reading.
type TelemetryData struct {
	ID          int64     `db:"id" json:"id"`
	SessionID   int64     `db:"session_id" json:"session_id"`
	StationID   string    `db:"station_id" json:"station_id"`
	ConnectorID int       `db:"connector_id" json:"connector_id"`
	MeterValue  float64   `db:"meter_value" json:"meter_value"`
	Unit        string    `db:"unit" json:"unit"`
	RecordedAt  time.Time `db:"recorded_at" json:"recorded_at"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}


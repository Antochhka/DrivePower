package models

import "time"

// SessionDTO mirrors sessions-service payload.
type SessionDTO struct {
	ID          int64     `json:"id"`
	StationID   string    `json:"station_id"`
	ConnectorID int       `json:"connector_id"`
	Status      string    `json:"status"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	EnergyKWh   float64   `json:"energy_kwh"`
}


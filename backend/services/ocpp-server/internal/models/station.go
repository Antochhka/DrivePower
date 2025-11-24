package models

import "time"

// Station represents a charging station.
type Station struct {
	ID              string    `db:"id" json:"id"`
	Vendor          string    `db:"vendor" json:"vendor"`
	Model           string    `db:"model" json:"model"`
	FirmwareVersion string    `db:"firmware_version" json:"firmwareVersion"`
	LastHeartbeat   time.Time `db:"last_heartbeat" json:"lastHeartbeat"`
	Status          string    `db:"status" json:"status"`
	CreatedAt       time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time `db:"updated_at" json:"updatedAt"`
}


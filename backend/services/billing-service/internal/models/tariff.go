package models

import "time"

// Tariff describes price per kWh.
type Tariff struct {
	ID         int64     `db:"id" json:"id"`
	Name       string    `db:"name" json:"name"`
	PricePerKWh float64  `db:"price_per_kwh" json:"price_per_kwh"`
	IsActive   bool      `db:"is_active" json:"is_active"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
	UpdatedAt  time.Time `db:"updated_at" json:"updated_at"`
}


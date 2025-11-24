package models

import "time"

// Transaction represents billing entry for completed session.
type Transaction struct {
	ID          int64     `db:"id" json:"id"`
	SessionID   int64     `db:"session_id" json:"session_id"`
	UserID      int64     `db:"user_id" json:"user_id"`
	EnergyKWh   float64   `db:"energy_kwh" json:"energy_kwh"`
	PricePerKWh float64   `db:"price_per_kwh" json:"price_per_kwh"`
	Amount      float64   `db:"amount" json:"amount"`
	Status      string    `db:"status" json:"status"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}


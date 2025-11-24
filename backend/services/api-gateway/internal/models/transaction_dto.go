package models

import "time"

// TransactionDTO mirrors billing-service response.
type TransactionDTO struct {
	ID          int64     `json:"id"`
	SessionID   int64     `json:"session_id"`
	EnergyKWh   float64   `json:"energy_kwh"`
	PricePerKWh float64   `json:"price_per_kwh"`
	Amount      float64   `json:"amount"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}


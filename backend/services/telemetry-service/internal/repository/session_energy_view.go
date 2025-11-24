package repository

import (
	"context"
	"database/sql"
)

// SessionEnergyView provides aggregated energy queries.
type SessionEnergyView struct {
	db *sql.DB
}

// NewSessionEnergyView returns view accessor.
func NewSessionEnergyView(db *sql.DB) *SessionEnergyView {
	return &SessionEnergyView{db: db}
}

// GetTotalEnergy returns precomputed aggregated energy if available.
func (v *SessionEnergyView) GetTotalEnergy(ctx context.Context, sessionID int64) (float64, error) {
	const query = `
		SELECT total_energy_kwh
		FROM session_energy_view
		WHERE session_id = $1
	`
	var energy float64
	if err := v.db.QueryRowContext(ctx, query, sessionID).Scan(&energy); err != nil {
		return 0, err
	}
	return energy, nil
}


package repository

import (
	"context"
	"database/sql"

	"drivepower/backend/services/billing-service/internal/models"
)

// TariffRepository handles tariff lookups.
type TariffRepository struct {
	db *sql.DB
}

// NewTariffRepository returns repository.
func NewTariffRepository(db *sql.DB) *TariffRepository {
	return &TariffRepository{db: db}
}

// GetActive returns currently active tariff (first active row).
func (r *TariffRepository) GetActive(ctx context.Context) (*models.Tariff, error) {
	const query = `
		SELECT id, name, price_per_kwh, is_active, created_at, updated_at
		FROM tariffs
		WHERE is_active = true
		ORDER BY updated_at DESC
		LIMIT 1
	`
	var t models.Tariff
	if err := r.db.QueryRowContext(ctx, query).Scan(
		&t.ID,
		&t.Name,
		&t.PricePerKWh,
		&t.IsActive,
		&t.CreatedAt,
		&t.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &t, nil
}


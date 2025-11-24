package service

import (
	"context"
	"errors"

	"drivepower/backend/services/billing-service/internal/models"
	"drivepower/backend/services/billing-service/internal/repository"
)

// TariffService provides tariff lookups with fallback.
type TariffService struct {
	repo          *repository.TariffRepository
	defaultTariff models.Tariff
}

// NewTariffService returns service instance.
func NewTariffService(repo *repository.TariffRepository, defaultPrice float64) *TariffService {
	return &TariffService{
		repo: repo,
		defaultTariff: models.Tariff{
			Name:       "Default",
			PricePerKWh: defaultPrice,
			IsActive:   true,
		},
	}
}

// ActiveTariff returns currently active tariff or default fallback.
func (s *TariffService) ActiveTariff(ctx context.Context) (*models.Tariff, error) {
	if s.repo == nil {
		if s.defaultTariff.PricePerKWh <= 0 {
			return nil, errors.New("tariff: no tariff configured")
		}
		return &s.defaultTariff, nil
	}

	tariff, err := s.repo.GetActive(ctx)
	if err != nil {
		if s.defaultTariff.PricePerKWh <= 0 {
			return nil, err
		}
		return &s.defaultTariff, nil
	}
	return tariff, nil
}


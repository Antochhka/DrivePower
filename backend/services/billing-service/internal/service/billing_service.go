package service

import (
	"context"
	"errors"

	"go.uber.org/zap"

	"drivepower/backend/services/billing-service/internal/models"
	"drivepower/backend/services/billing-service/internal/repository"
)

// BillingService handles transaction creation.
type BillingService struct {
	txRepo        *repository.TransactionRepository
	tariffService *TariffService
	logger        *zap.Logger
}

// NewBillingService builds service.
func NewBillingService(txRepo *repository.TransactionRepository, tariffSvc *TariffService, logger *zap.Logger) *BillingService {
	return &BillingService{
		txRepo:        txRepo,
		tariffService: tariffSvc,
		logger:        logger,
	}
}

// CreateTransactionInput represents callback payload.
type CreateTransactionInput struct {
	SessionID int64
	UserID    int64
	EnergyKWh float64
}

// CalculateAndCreateTransaction calculates amount and stores transaction.
func (s *BillingService) CalculateAndCreateTransaction(ctx context.Context, input CreateTransactionInput) (*models.Transaction, error) {
	if input.SessionID == 0 {
		return nil, errors.New("billing: session id required")
	}

	tariff, err := s.tariffService.ActiveTariff(ctx)
	if err != nil {
		return nil, err
	}
	pricePerKWh := tariff.PricePerKWh
	if pricePerKWh <= 0 {
		pricePerKWh = 1
	}
	amount := input.EnergyKWh * pricePerKWh

	tx := &models.Transaction{
		SessionID:   input.SessionID,
		UserID:      input.UserID,
		EnergyKWh:   input.EnergyKWh,
		PricePerKWh: pricePerKWh,
		Amount:      amount,
		Status:      "completed",
	}

	if err := s.txRepo.Create(ctx, tx); err != nil {
		return nil, err
	}

	s.logger.Info("billing transaction created",
		zap.Int64("session_id", input.SessionID),
		zap.Float64("energy_kwh", input.EnergyKWh),
		zap.Float64("amount", amount),
	)
	return tx, nil
}

// TransactionsForUser returns history for given user.
func (s *BillingService) TransactionsForUser(ctx context.Context, userID int64, limit int) ([]models.Transaction, error) {
	return s.txRepo.ListByUser(ctx, userID, limit)
}


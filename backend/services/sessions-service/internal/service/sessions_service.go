package service

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"drivepower/backend/services/sessions-service/internal/models"
	"drivepower/backend/services/sessions-service/internal/redis"
	"drivepower/backend/services/sessions-service/internal/repository"
)

// Status constants.
const (
	SessionStatusActive    = "active"
	SessionStatusCompleted = "completed"
	SessionStatusUnknown   = "unknown"
)

// SessionsService ties repository and cache.
type SessionsService struct {
	repo        *repository.SessionRepository
	activeStore *redisstore.Store
	logger      *zap.Logger
}

// StartSessionInput data from OCPP start notification.
type StartSessionInput struct {
	UserID        int64
	StationID     string
	ConnectorID   int
	TransactionID string
	StartTime     time.Time
}

// StopSessionInput data from OCPP stop notification.
type StopSessionInput struct {
	TransactionID string
	EndTime       time.Time
	EnergyKWh     float64
}

// NewSessionsService builds service.
func NewSessionsService(
	repo *repository.SessionRepository,
	activeStore *redisstore.Store,
	logger *zap.Logger,
) *SessionsService {
	return &SessionsService{
		repo:        repo,
		activeStore: activeStore,
		logger:      logger,
	}
}

// StartSessionFromOCPP handles start event.
func (s *SessionsService) StartSessionFromOCPP(ctx context.Context, input StartSessionInput) (*models.Session, error) {
	if input.StartTime.IsZero() {
		input.StartTime = time.Now().UTC()
	}
	session := &models.Session{
		UserID:      input.UserID,
		StationID:   input.StationID,
		ConnectorID: input.ConnectorID,
		Status:      SessionStatusActive,
		StartTime:   input.StartTime.UTC(),
		Transaction: input.TransactionID,
	}

	session, err := s.repo.StartSession(ctx, session)
	if err != nil {
		return nil, err
	}

	if s.activeStore != nil {
		cacheErr := s.activeStore.Save(ctx, redisstore.ActiveSession{
			SessionID:     session.ID,
			TransactionID: input.TransactionID,
			StationID:     input.StationID,
			ConnectorID:   input.ConnectorID,
			UserID:        input.UserID,
		})
		if cacheErr != nil && cacheErr != redis.Nil {
			s.logger.Warn("failed to cache active session", zap.Error(cacheErr))
		}
	}

	return session, nil
}

// StopSessionFromOCPP handles completion event.
func (s *SessionsService) StopSessionFromOCPP(ctx context.Context, input StopSessionInput) error {
	if input.EndTime.IsZero() {
		input.EndTime = time.Now().UTC()
	}
	if err := s.repo.CompleteSession(ctx, input.TransactionID, input.EndTime, input.EnergyKWh, SessionStatusCompleted); err != nil {
		return err
	}

	if s.activeStore != nil {
		if err := s.activeStore.Delete(ctx, input.TransactionID); err != nil && err != redis.Nil {
			s.logger.Warn("failed to delete active session cache", zap.Error(err))
		}
	}
	return nil
}

// GetSessionsByUser returns user's session history.
func (s *SessionsService) GetSessionsByUser(ctx context.Context, userID int64, limit int) ([]models.Session, error) {
	return s.repo.GetSessionsByUser(ctx, userID, limit)
}

// GetActiveSessions returns currently running sessions.
func (s *SessionsService) GetActiveSessions(ctx context.Context, limit int) ([]models.Session, error) {
	return s.repo.GetActiveSessions(ctx, limit)
}

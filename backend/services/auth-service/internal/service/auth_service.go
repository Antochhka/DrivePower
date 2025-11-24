package service

import (
	"context"
	"errors"
	"strings"

	"go.uber.org/zap"

	"drivepower/backend/services/auth-service/internal/models"
	"drivepower/backend/services/auth-service/internal/password"
	"drivepower/backend/services/auth-service/internal/repository"
)

var (
	// ErrEmailInUse is returned when attempting to register duplicate email.
	ErrEmailInUse = errors.New("auth: email already registered")
	// ErrInvalidCredentials represents login failure.
	ErrInvalidCredentials = errors.New("auth: invalid credentials")
)

// UserRepository defines storage contract used by the service.
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByEmail(ctx context.Context, email string) (*models.User, error)
}

// AuthService contains registration/login logic.
type AuthService struct {
	repo      UserRepository
	hasher    password.Hasher
	tokenizer *TokenService
	logger    *zap.Logger
}

// NewAuthService builds AuthService.
func NewAuthService(repo UserRepository, hasher password.Hasher, tokenizer *TokenService, logger *zap.Logger) *AuthService {
	return &AuthService{
		repo:      repo,
		hasher:    hasher,
		tokenizer: tokenizer,
		logger:    logger,
	}
}

// Signup registers a new user.
func (s *AuthService) Signup(ctx context.Context, email, password string, role string) (*models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, errors.New("auth: email required")
	}
	if password == "" {
		return nil, errors.New("auth: password required")
	}
	if role == "" {
		role = "user"
	}

	if _, err := s.repo.GetByEmail(ctx, email); err == nil {
		return nil, ErrEmailInUse
	} else if !errors.Is(err, repository.ErrUserNotFound) {
		return nil, err
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Email:        email,
		PasswordHash: hash,
		Role:         role,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	s.logger.Info("user signed up", zap.Int64("user_id", user.ID), zap.String("email", user.Email))
	return user, nil
}

// Login authenticates a user and produces a JWT.
func (s *AuthService) Login(ctx context.Context, email, password string) (string, *models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return "", nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return "", nil, ErrInvalidCredentials
		}
		return "", nil, err
	}

	if err := s.hasher.Compare(user.PasswordHash, password); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	token, err := s.tokenizer.GenerateToken(user.ID, user.Role)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}


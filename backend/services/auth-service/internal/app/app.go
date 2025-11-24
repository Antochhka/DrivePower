package app

import (
	"context"
	"database/sql"

	"go.uber.org/zap"

	appconfig "drivepower/backend/services/auth-service/internal/config"
	"drivepower/backend/services/auth-service/internal/db"
	"drivepower/backend/services/auth-service/internal/http"
	"drivepower/backend/services/auth-service/internal/http/handlers"
	"drivepower/backend/services/auth-service/internal/password"
	"drivepower/backend/services/auth-service/internal/repository"
	"drivepower/backend/services/auth-service/internal/service"
)

// App wires dependencies for the auth service.
type App struct {
	server *httpserver.Server
	db     *sql.DB
	logger *zap.Logger
}

// New builds application graph.
func New(cfg *appconfig.Config, logger *zap.Logger) (*App, error) {
	sqlDB, err := db.NewPostgres(cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	userRepo := repository.NewUserRepository(sqlDB)
	hasher := password.NewBcryptHasher(0)
	tokenSvc := service.NewTokenService(cfg.JWT.Secret, cfg.JWTExpiration())
	authSvc := service.NewAuthService(userRepo, hasher, tokenSvc, logger)

	routes := httpserver.Routes{
		Signup: handlers.NewSignupHandler(authSvc),
		Login:  handlers.NewLoginHandler(authSvc),
		Health: handlers.NewHealthHandler(),
	}

	router := httpserver.NewRouter(routes)
	server := httpserver.NewServer(cfg.HTTPAddress(), router, logger)

	return &App{
		server: server,
		db:     sqlDB,
		logger: logger,
	}, nil
}

// Run starts serving HTTP traffic until context cancellation.
func (a *App) Run(ctx context.Context) error {
	return a.server.Run(ctx)
}

// Close releases acquired resources.
func (a *App) Close() {
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Warn("failed to close db", zap.Error(err))
		}
	}
}


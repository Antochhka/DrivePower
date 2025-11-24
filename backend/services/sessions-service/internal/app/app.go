package app

import (
	"context"
	"database/sql"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	libredis "drivepower/backend/libs/redis"
	"drivepower/backend/services/sessions-service/internal/config"
	"drivepower/backend/services/sessions-service/internal/db"
	httpserver "drivepower/backend/services/sessions-service/internal/http"
	"drivepower/backend/services/sessions-service/internal/http/handlers"
	redisstore "drivepower/backend/services/sessions-service/internal/redis"
	"drivepower/backend/services/sessions-service/internal/repository"
	"drivepower/backend/services/sessions-service/internal/service"
)

// App wires sessions-service dependencies.
type App struct {
	server      *httpserver.Server
	db          *sql.DB
	redisClient *redis.Client
	logger      *zap.Logger
}

// New constructs the application graph.
func New(cfg *config.Config, logger *zap.Logger) (*App, error) {
	sqlDB, err := db.NewPostgres(cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	redisClient, err := libredis.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password)
	if err != nil {
		sqlDB.Close()
		return nil, err
	}

	sessionRepo := repository.NewSessionRepository(sqlDB)
	stationRepo := repository.NewStationRepository(sqlDB)
	_ = stationRepo // placeholder for future use

	activeStore := redisstore.NewStore(redisClient, cfg.ActiveSessionTTL())
	sessionsService := service.NewSessionsService(sessionRepo, activeStore, logger)

	ocppHandler := handlers.NewOCPPCallbacksHandler(sessionsService, logger)

	routes := httpserver.Routes{
		SessionsMe:     handlers.NewSessionsMeHandler(sessionsService),
		ActiveSessions: handlers.NewActiveSessionsHandler(sessionsService),
		SessionStart:   ocppHandler.HandleSessionStart,
		SessionStop:    ocppHandler.HandleSessionStop,
		Health:         handlers.NewHealthHandler(),
	}

	router := httpserver.NewRouter(routes)
	server := httpserver.NewServer(cfg.HTTPAddress(), router, logger)

	return &App{
		server:      server,
		db:          sqlDB,
		redisClient: redisClient,
		logger:      logger,
	}, nil
}

// Run starts HTTP server.
func (a *App) Run(ctx context.Context) error {
	return a.server.Run(ctx)
}

// Close releases resources.
func (a *App) Close() {
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Warn("failed to close db", zap.Error(err))
		}
	}
	if a.redisClient != nil {
		if err := a.redisClient.Close(); err != nil {
			a.logger.Warn("failed to close redis", zap.Error(err))
		}
	}
}

package app

import (
	"context"
	"database/sql"

	"go.uber.org/zap"

	"drivepower/backend/services/telemetry-service/internal/config"
	"drivepower/backend/services/telemetry-service/internal/db"
	httpserver "drivepower/backend/services/telemetry-service/internal/http"
	"drivepower/backend/services/telemetry-service/internal/http/handlers"
	"drivepower/backend/services/telemetry-service/internal/repository"
	"drivepower/backend/services/telemetry-service/internal/service"
)

// App wires telemetry service dependencies.
type App struct {
	server *httpserver.Server
	db     *sql.DB
	logger *zap.Logger
}

// New constructs application components.
func New(cfg *config.Config, logger *zap.Logger) (*App, error) {
	sqlDB, err := db.NewPostgres(cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	telemetryRepo := repository.NewTelemetryRepository(sqlDB)
	energyView := repository.NewSessionEnergyView(sqlDB)
	telemetryService := service.NewTelemetryService(telemetryRepo, energyView, logger)

	routes := httpserver.Routes{
		MeterValues: handlers.NewMeterHandler(telemetryService, logger),
		Health:      handlers.NewHealthHandler(),
	}

	router := httpserver.NewRouter(routes)
	server := httpserver.NewServer(cfg.HTTPAddress(), router, logger)

	return &App{
		server: server,
		db:     sqlDB,
		logger: logger,
	}, nil
}

// Run starts serving HTTP requests.
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
}


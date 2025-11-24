package app

import (
	"context"
	"database/sql"

	"go.uber.org/zap"

	"drivepower/backend/services/billing-service/internal/config"
	"drivepower/backend/services/billing-service/internal/db"
	httpserver "drivepower/backend/services/billing-service/internal/http"
	"drivepower/backend/services/billing-service/internal/http/handlers"
	"drivepower/backend/services/billing-service/internal/repository"
	"drivepower/backend/services/billing-service/internal/service"
)

// App wires billing service dependencies.
type App struct {
	server *httpserver.Server
	db     *sql.DB
	logger *zap.Logger
}

// New constructs application graph.
func New(cfg *config.Config, logger *zap.Logger) (*App, error) {
	sqlDB, err := db.NewPostgres(cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	txRepo := repository.NewTransactionRepository(sqlDB)
	tariffRepo := repository.NewTariffRepository(sqlDB)
	tariffService := service.NewTariffService(tariffRepo, 7.0) // default price
	billingService := service.NewBillingService(txRepo, tariffService, logger)

	sessionStoppedHandler := handlers.NewOCPPStopHandler(billingService, logger)

	routes := httpserver.Routes{
		SessionStopped: sessionStoppedHandler,
		TransactionsMe: handlers.NewTransactionsMeHandler(billingService),
		Health:         handlers.NewHealthHandler(),
	}

	router := httpserver.NewRouter(routes)
	server := httpserver.NewServer(cfg.HTTPAddress(), router, logger)

	return &App{
		server: server,
		db:     sqlDB,
		logger: logger,
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
}


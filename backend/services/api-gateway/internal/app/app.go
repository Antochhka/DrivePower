package app

import (
	"context"
	"net/http"

	"go.uber.org/zap"

	"drivepower/backend/services/api-gateway/internal/clients"
	"drivepower/backend/services/api-gateway/internal/config"
	httpserver "drivepower/backend/services/api-gateway/internal/http"
	"drivepower/backend/services/api-gateway/internal/http/handlers"
	"drivepower/backend/services/api-gateway/internal/http/middleware"
)

// App wires API gateway dependencies.
type App struct {
	server *httpserver.Server
	logger *zap.Logger
}

// New constructs application graph.
func New(cfg *config.Config, logger *zap.Logger) (*App, error) {
	httpClient := clients.NewDefaultHTTPClient(cfg.HTTPTimeout())

	authClient := clients.NewAuthClient(cfg.Services.AuthURL, httpClient)
	sessionsClient := clients.NewSessionsClient(cfg.Services.SessionsURL, httpClient)
	billingClient := clients.NewBillingClient(cfg.Services.BillingURL, httpClient)
	stationsClient := clients.NewStationsClient(cfg.Services.StationsURL, httpClient)

	authHandlers := handlers.NewAuthHandlers(authClient, logger)
	sessionsHandlers := handlers.NewSessionsHandlers(sessionsClient, logger)
	billingHandlers := handlers.NewBillingHandlers(billingClient, logger)
	stationsHandlers := handlers.NewStationsHandlers(stationsClient, logger)

	router := httpserver.NewRouter(httpserver.RouterDeps{
		AuthHandlers:     authHandlers,
		StationsHandlers: stationsHandlers,
		SessionsHandlers: sessionsHandlers,
		BillingHandlers:  billingHandlers,
		HealthHandler:    handlers.NewHealthHandler(),
	}, middleware.AuthMiddleware(cfg.JWT.Secret))

	server := httpserver.NewServer(
		cfg.HTTPAddress(),
		router,
		logger,
		middleware.RecoveryMiddleware(logger),
		middleware.LoggingMiddleware(logger),
	)

	return &App{
		server: server,
		logger: logger,
	}, nil
}

// Run starts serving HTTP traffic.
func (a *App) Run(ctx context.Context) error {
	return a.server.Run(ctx)
}

// Close releases resources (none yet).
func (a *App) Close() {}


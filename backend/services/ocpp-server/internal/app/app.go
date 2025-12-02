package app

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"go.uber.org/zap"

	"drivepower/backend/services/ocpp-server/internal/clients"
	"drivepower/backend/services/ocpp-server/internal/config"
	"drivepower/backend/services/ocpp-server/internal/db"
	"drivepower/backend/services/ocpp-server/internal/handlers"
	"drivepower/backend/services/ocpp-server/internal/ocpp"
	"drivepower/backend/services/ocpp-server/internal/ocpp/protocol"
	"drivepower/backend/services/ocpp-server/internal/repository"
	"drivepower/backend/services/ocpp-server/internal/service"
	"drivepower/backend/services/ocpp-server/internal/ws"
)

// App wires all dependencies for the OCPP server.
type App struct {
	httpServer *http.Server
	db         *sql.DB
	manager    *ws.Manager
	logger     *zap.Logger
}

// New builds the application graph.
func New(cfg *config.Config, logger *zap.Logger) (*App, error) {
	sqlDB, err := db.NewPostgres(cfg.Database.DSN)
	if err != nil {
		return nil, err
	}

	stationRepo := repository.NewStationRepository(sqlDB)
	logRepo := repository.NewOCPPLogRepository(sqlDB)
	stationState := service.NewStationState()
	txStore := service.NewTransactionStore()

	sessionsClient := clients.NewSessionsClient(cfg.Services.SessionsURL, logger)
	billingClient := clients.NewBillingClient(cfg.Services.BillingURL, logger)
	telemetryClient := clients.NewTelemetryClient(cfg.Services.TelemetryURL, logger)

	router := ocpp.NewRouter()
	parser := ocpp.NewParser()
	processor := ocpp.NewProcessor(parser, router, logRepo, logger)

	router.Register(protocol.ActionBootNotification, handlers.NewBootNotificationHandler(stationRepo, stationState, logger))
	router.Register(protocol.ActionStatusNotification, handlers.NewStatusNotificationHandler(stationRepo, stationState, logger))
	router.Register(protocol.ActionStartTransaction, handlers.NewStartTransactionHandler(sessionsClient, billingClient, stationState, txStore, logger))
	router.Register(protocol.ActionStopTransaction, handlers.NewStopTransactionHandler(sessionsClient, billingClient, stationState, txStore, logger))
	router.Register(protocol.ActionHeartbeat, handlers.NewHeartbeatHandler())
	router.Register(protocol.ActionMeterValues, handlers.NewMeterValuesHandler(telemetryClient, txStore, logger))

	manager := ws.NewManager(cfg.PingInterval())
	wsServer := ws.NewServer(manager, processor, cfg.WriteTimeout(), logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/ocpp/ws", wsServer.HandleWS)

	httpServer := &http.Server{
		Addr:         cfg.HTTPAddress(),
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &App{
		httpServer: httpServer,
		db:         sqlDB,
		manager:    manager,
		logger:     logger,
	}, nil
}

// Run starts manager and HTTP server.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go a.manager.Start(ctx)

	go func() {
		a.logger.Info("starting ocpp http server", zap.String("addr", a.httpServer.Addr))
		if err := a.httpServer.ListenAndServe(); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return a.httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

// Close releases resources.
func (a *App) Close() {
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Warn("failed to close db", zap.Error(err))
		}
	}
}

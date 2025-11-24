package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"drivepower/backend/libs/logging"
	"drivepower/backend/services/telemetry-service/internal/app"
	"drivepower/backend/services/telemetry-service/internal/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger, err := logging.NewLogger()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	application, err := app.New(cfg, logger)
	if err != nil {
		logger.Fatal("failed to init application", zap.Error(err))
	}
	defer application.Close()

	if err := application.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal("application stopped with error", zap.Error(err))
	}
}


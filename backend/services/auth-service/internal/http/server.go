package httpserver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Server wraps http.Server with graceful shutdown logic.
type Server struct {
	server *http.Server
	logger *zap.Logger
}

// NewServer builds a Server instance.
func NewServer(addr string, handler http.Handler, logger *zap.Logger) *Server {
	return &Server{
		server: &http.Server{
			Addr:         addr,
			Handler:      handler,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		logger: logger,
	}
}

// Run starts listening and blocks until context is cancelled or server stops.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("starting http server", zap.String("addr", s.server.Addr))
		if err := s.server.ListenAndServe(); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		s.logger.Info("shutting down http server")
		if err := s.server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}


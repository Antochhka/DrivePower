package httpserver

import (
	"context"
	"errors"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Server wraps http.Server with middleware.
type Server struct {
	server *http.Server
	logger *zap.Logger
}

// NewServer builds HTTP server with provided handler.
func NewServer(addr string, handler http.Handler, logger *zap.Logger, middlewares ...func(http.Handler) http.Handler) *Server {
	h := handler
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return &Server{
		server: &http.Server{
			Addr:         addr,
			Handler:      h,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		logger: logger,
	}
}

// Run starts the HTTP server.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		s.logger.Info("starting api gateway", zap.String("addr", s.server.Addr))
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
		return s.server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}


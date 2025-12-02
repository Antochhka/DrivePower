package ws

import (
	"context"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Server upgrades HTTP connections to WebSockets for OCPP.
type Server struct {
	manager      *Manager
	processor    MessageProcessor
	logger       *zap.Logger
	writeTimeout time.Duration
	upgrader     websocket.Upgrader
}

// NewServer builds ws server.
func NewServer(manager *Manager, processor MessageProcessor, writeTimeout time.Duration, logger *zap.Logger) *Server {
	return &Server{
		manager:   manager,
		processor: processor,
		logger:    logger,
		writeTimeout: writeTimeout,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

// HandleWS is HTTP handler for /ocpp/ws endpoint.
func (s *Server) HandleWS(w http.ResponseWriter, r *http.Request) {
	stationID := r.URL.Query().Get("station_id")
	if stationID == "" {
		http.Error(w, "station_id is required", http.StatusBadRequest)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	connection := NewConnection(stationID, conn, s.processor, s.writeTimeout, s.logger, func(id string) {
		s.manager.Remove(id)
		cancel()
	})
	s.manager.Add(connection)

	go connection.Start(ctx)
	s.logger.Info("station connected", zap.String("station_id", stationID))
}

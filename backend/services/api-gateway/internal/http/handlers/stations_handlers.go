package handlers

import (
	"net/http"

	"go.uber.org/zap"

	"drivepower/backend/services/api-gateway/internal/clients"
)

// StationsHandlers proxies station status endpoints.
type StationsHandlers struct {
	client *clients.StationsClient
	logger *zap.Logger
}

// NewStationsHandlers returns handler.
func NewStationsHandlers(client *clients.StationsClient, logger *zap.Logger) *StationsHandlers {
	return &StationsHandlers{client: client, logger: logger}
}

// List handles GET /api/stations.
func (h *StationsHandlers) List(w http.ResponseWriter, r *http.Request) {
	status, body, err := h.client.ListStations(r.Context())
	if err != nil {
		h.logger.Error("stations proxy failed", zap.Error(err))
		writeError(w, http.StatusBadGateway, "stations service unavailable")
		return
	}
	writeRaw(w, status, body)
}


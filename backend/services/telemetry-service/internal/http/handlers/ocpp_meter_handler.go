package handlers

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"drivepower/backend/services/telemetry-service/internal/service"
)

// MeterHandler handles OCPP meter value callbacks.
type MeterHandler struct {
	service *service.TelemetryService
	logger  *zap.Logger
}

// NewMeterHandler returns handler.
func NewMeterHandler(service *service.TelemetryService, logger *zap.Logger) *MeterHandler {
	return &MeterHandler{
		service: service,
		logger:  logger,
	}
}

// ServeHTTP handles POST /internal/ocpp/meter-values.
func (h *MeterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var input service.MeterValueInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if input.SessionID == 0 {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	if err := h.service.StoreMeterValue(r.Context(), input); err != nil {
		h.logger.Error("failed to store meter value", zap.Error(err))
		http.Error(w, "failed to store meter value", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}


package handlers

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

	"drivepower/backend/services/billing-service/internal/service"
)

// OCPPStopHandler handles billing events from OCPP server.
type OCPPStopHandler struct {
	service *service.BillingService
	logger  *zap.Logger
}

// NewOCPPStopHandler builds handler.
func NewOCPPStopHandler(service *service.BillingService, logger *zap.Logger) *OCPPStopHandler {
	return &OCPPStopHandler{
		service: service,
		logger:  logger,
	}
}

type sessionStoppedRequest struct {
	SessionID int64   `json:"session_id"`
	UserID    int64   `json:"user_id"`
	EnergyKWh float64 `json:"energy_kwh"`
}

// ServeHTTP handles POST /internal/ocpp/session-stopped.
func (h *OCPPStopHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req sessionStoppedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.SessionID == 0 {
		writeError(w, http.StatusBadRequest, "session_id required")
		return
	}

	tx, err := h.service.CalculateAndCreateTransaction(r.Context(), service.CreateTransactionInput{
		SessionID: req.SessionID,
		UserID:    req.UserID,
		EnergyKWh: req.EnergyKWh,
	})
	if err != nil {
		h.logger.Error("failed to create billing transaction", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "billing calculation failed")
		return
	}

	writeJSON(w, http.StatusCreated, tx)
}


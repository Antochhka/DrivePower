package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"drivepower/backend/services/sessions-service/internal/service"
)

// OCPPCallbacksHandler holds endpoints invoked by OCPP server.
type OCPPCallbacksHandler struct {
	svc    *service.SessionsService
	logger *zap.Logger
}

// NewOCPPCallbacksHandler builds handler set.
func NewOCPPCallbacksHandler(svc *service.SessionsService, logger *zap.Logger) *OCPPCallbacksHandler {
	return &OCPPCallbacksHandler{
		svc:    svc,
		logger: logger,
	}
}

type sessionStartRequest struct {
	UserID        int64     `json:"user_id"`
	StationID     string    `json:"station_id"`
	ConnectorID   int       `json:"connector_id"`
	TransactionID string    `json:"transaction_id"`
	StartTime     time.Time `json:"start_time"`
}

type sessionStopRequest struct {
	TransactionID string    `json:"transaction_id"`
	EndTime       time.Time `json:"end_time"`
	EnergyKWh     float64   `json:"energy_kwh"`
}

// HandleSessionStart handles POST /internal/ocpp/session-start.
func (h *OCPPCallbacksHandler) HandleSessionStart(w http.ResponseWriter, r *http.Request) {
	var req sessionStartRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.TransactionID == "" {
		writeError(w, http.StatusBadRequest, "transaction_id is required")
		return
	}

	session, err := h.svc.StartSessionFromOCPP(r.Context(), service.StartSessionInput{
		UserID:        req.UserID,
		StationID:     req.StationID,
		ConnectorID:   req.ConnectorID,
		TransactionID: req.TransactionID,
		StartTime:     req.StartTime,
	})
	if err != nil {
		h.logger.Error("start session failed", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to start session")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]interface{}{"status": "ok", "session_id": session.ID})
}

// HandleSessionStop handles POST /internal/ocpp/session-stop.
func (h *OCPPCallbacksHandler) HandleSessionStop(w http.ResponseWriter, r *http.Request) {
	var req sessionStopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.TransactionID == "" {
		writeError(w, http.StatusBadRequest, "transaction_id is required")
		return
	}

	if err := h.svc.StopSessionFromOCPP(r.Context(), service.StopSessionInput{
		TransactionID: req.TransactionID,
		EndTime:       req.EndTime,
		EnergyKWh:     req.EnergyKWh,
	}); err != nil {
		h.logger.Error("stop session failed", zap.Error(err))
		writeError(w, http.StatusInternalServerError, "failed to stop session")
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "ok"})
}

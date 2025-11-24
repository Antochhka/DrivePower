package handlers

import (
	"net/http"

	"go.uber.org/zap"

	"drivepower/backend/services/api-gateway/internal/clients"
	"drivepower/backend/services/api-gateway/internal/http/middleware"
)

// SessionsHandlers proxies sessions-service endpoints.
type SessionsHandlers struct {
	client *clients.SessionsClient
	logger *zap.Logger
}

// NewSessionsHandlers returns handler.
func NewSessionsHandlers(client *clients.SessionsClient, logger *zap.Logger) *SessionsHandlers {
	return &SessionsHandlers{client: client, logger: logger}
}

// Me handles GET /api/sessions/me.
func (h *SessionsHandlers) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	status, respBody, err := h.client.GetSessionsForUser(r.Context(), userID)
	if err != nil {
		h.logger.Error("sessions proxy failed", zap.Error(err))
		writeError(w, http.StatusBadGateway, "sessions service unavailable")
		return
	}
	writeRaw(w, status, respBody)
}


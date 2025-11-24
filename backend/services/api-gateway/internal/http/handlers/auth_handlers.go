package handlers

import (
	"io"
	"net/http"

	"go.uber.org/zap"

	"drivepower/backend/services/api-gateway/internal/clients"
)

// AuthHandlers proxies auth-service endpoints.
type AuthHandlers struct {
	client *clients.AuthClient
	logger *zap.Logger
}

// NewAuthHandlers returns handler struct.
func NewAuthHandlers(client *clients.AuthClient, logger *zap.Logger) *AuthHandlers {
	return &AuthHandlers{client: client, logger: logger}
}

// Signup handles POST /api/auth/signup.
func (h *AuthHandlers) Signup(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	status, respBody, err := h.client.Signup(r.Context(), body)
	if err != nil {
		h.logger.Error("signup proxy failed", zap.Error(err))
		writeError(w, http.StatusBadGateway, "auth service unavailable")
		return
	}
	writeRaw(w, status, respBody)
}

// Login handles POST /api/auth/login.
func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid body")
		return
	}
	status, respBody, err := h.client.Login(r.Context(), body)
	if err != nil {
		h.logger.Error("login proxy failed", zap.Error(err))
		writeError(w, http.StatusBadGateway, "auth service unavailable")
		return
	}
	writeRaw(w, status, respBody)
}


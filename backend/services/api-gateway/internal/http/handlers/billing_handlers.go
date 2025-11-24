package handlers

import (
	"net/http"

	"go.uber.org/zap"

	"drivepower/backend/services/api-gateway/internal/clients"
	"drivepower/backend/services/api-gateway/internal/http/middleware"
)

// BillingHandlers proxies billing-service endpoints.
type BillingHandlers struct {
	client *clients.BillingClient
	logger *zap.Logger
}

// NewBillingHandlers returns handler.
func NewBillingHandlers(client *clients.BillingClient, logger *zap.Logger) *BillingHandlers {
	return &BillingHandlers{client: client, logger: logger}
}

// TransactionsMe handles GET /api/billing/me/transactions.
func (h *BillingHandlers) TransactionsMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	status, respBody, err := h.client.GetTransactionsForUser(r.Context(), userID)
	if err != nil {
		h.logger.Error("billing proxy failed", zap.Error(err))
		writeError(w, http.StatusBadGateway, "billing service unavailable")
		return
	}
	writeRaw(w, status, respBody)
}


package handlers

import (
	"net/http"
	"strconv"

	"drivepower/backend/services/billing-service/internal/service"
)

const userIDHeader = "X-User-ID"

// NewTransactionsMeHandler returns GET /billing/me/transactions handler.
func NewTransactionsMeHandler(svc *service.BillingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDStr := r.Header.Get(userIDHeader)
		if userIDStr == "" {
			writeError(w, http.StatusUnauthorized, "missing user id header")
			return
		}
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid user id header")
			return
		}

		transactions, err := svc.TransactionsForUser(r.Context(), userID, 50)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to load transactions")
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"transactions": transactions,
		})
	}
}


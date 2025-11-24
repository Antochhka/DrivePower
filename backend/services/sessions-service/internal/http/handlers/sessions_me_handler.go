package handlers

import (
	"net/http"
	"strconv"

	"drivepower/backend/services/sessions-service/internal/service"
)

const userIDHeader = "X-User-ID"

// NewSessionsMeHandler returns GET /sessions/me handler.
func NewSessionsMeHandler(svc *service.SessionsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userIDStr := r.Header.Get(userIDHeader)
		if userIDStr == "" {
			writeError(w, http.StatusUnauthorized, "missing user id header")
			return
		}
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid user id")
			return
		}

		sessions, err := svc.GetSessionsByUser(r.Context(), userID, 50)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to fetch sessions")
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"sessions": sessions,
		})
	}
}


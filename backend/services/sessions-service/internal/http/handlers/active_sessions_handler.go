package handlers

import (
	"net/http"

	"drivepower/backend/services/sessions-service/internal/service"
)

// NewActiveSessionsHandler returns GET /sessions/active handler.
func NewActiveSessionsHandler(svc *service.SessionsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessions, err := svc.GetActiveSessions(r.Context(), 50)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to fetch active sessions")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"sessions": sessions,
		})
	}
}


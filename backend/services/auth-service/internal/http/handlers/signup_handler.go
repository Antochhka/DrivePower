package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"drivepower/backend/services/auth-service/internal/service"
)

// NewSignupHandler returns HTTP handler for registration endpoint.
func NewSignupHandler(authService *service.AuthService) http.HandlerFunc {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	type response struct {
		ID    int64  `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		var req request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		req.Email = strings.TrimSpace(req.Email)
		req.Password = strings.TrimSpace(req.Password)
		if req.Email == "" || req.Password == "" {
			writeError(w, http.StatusBadRequest, "email and password are required")
			return
		}

		user, err := authService.Signup(r.Context(), req.Email, req.Password, req.Role)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrEmailInUse):
				writeError(w, http.StatusConflict, "email already registered")
			default:
				writeError(w, http.StatusInternalServerError, "failed to create user")
			}
			return
		}

		writeJSON(w, http.StatusCreated, response{
			ID:    user.ID,
			Email: user.Email,
			Role:  user.Role,
		})
	}
}


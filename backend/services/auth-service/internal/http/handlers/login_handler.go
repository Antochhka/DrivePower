package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"drivepower/backend/services/auth-service/internal/service"
)

// NewLoginHandler handles POST /auth/login.
func NewLoginHandler(authService *service.AuthService) http.HandlerFunc {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type response struct {
		Token     string `json:"token"`
		TokenType string `json:"token_type"`
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

		token, _, err := authService.Login(r.Context(), req.Email, req.Password)
		if err != nil {
			if errors.Is(err, service.ErrInvalidCredentials) {
				writeError(w, http.StatusUnauthorized, "invalid credentials")
				return
			}
			writeError(w, http.StatusInternalServerError, "failed to login")
			return
		}

		writeJSON(w, http.StatusOK, response{
			Token:     token,
			TokenType: "Bearer",
		})
	}
}


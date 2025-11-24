package httpserver

import (
	"net/http"

	"drivepower/backend/services/api-gateway/internal/http/handlers"
	"drivepower/backend/services/api-gateway/internal/http/middleware"
)

// RouterDeps collects handler dependencies.
type RouterDeps struct {
	AuthHandlers     *handlers.AuthHandlers
	StationsHandlers *handlers.StationsHandlers
	SessionsHandlers *handlers.SessionsHandlers
	BillingHandlers  *handlers.BillingHandlers
	HealthHandler    http.HandlerFunc
}

// NewRouter wires HTTP routes with middleware.
func NewRouter(deps RouterDeps, authMiddleware func(http.Handler) http.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/health", method(http.MethodGet, deps.HealthHandler))

	mux.Handle("/api/auth/signup", method(http.MethodPost, http.HandlerFunc(deps.AuthHandlers.Signup)))
	mux.Handle("/api/auth/login", method(http.MethodPost, http.HandlerFunc(deps.AuthHandlers.Login)))

	mux.Handle("/api/stations", method(http.MethodGet, http.HandlerFunc(deps.StationsHandlers.List)))

	authenticated := func(handler http.HandlerFunc) http.Handler {
		return middleware.Chain(handler, authMiddleware)
	}

	mux.Handle("/api/sessions/me", method(http.MethodGet, authenticated(http.HandlerFunc(deps.SessionsHandlers.Me))))
	mux.Handle("/api/billing/me/transactions", method(http.MethodGet, authenticated(http.HandlerFunc(deps.BillingHandlers.TransactionsMe))))

	return mux
}

func method(expected string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != expected {
			w.Header().Set("Allow", expected)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		handler.ServeHTTP(w, r)
	})
}


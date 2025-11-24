package httpserver

import "net/http"

// Routes groups HTTP handlers.
type Routes struct {
	SessionStopped http.Handler
	TransactionsMe http.HandlerFunc
	Health         http.HandlerFunc
}

// NewRouter registers service endpoints.
func NewRouter(routes Routes) http.Handler {
	mux := http.NewServeMux()
	if routes.SessionStopped != nil {
		mux.Handle("/internal/ocpp/session-stopped", method(http.MethodPost, routes.SessionStopped.ServeHTTP))
	}
	if routes.TransactionsMe != nil {
		mux.Handle("/billing/me/transactions", method(http.MethodGet, routes.TransactionsMe))
	}
	if routes.Health != nil {
		mux.Handle("/health", method(http.MethodGet, routes.Health))
	}
	return mux
}

func method(expected string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != expected {
			w.Header().Set("Allow", expected)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		handler(w, r)
	}
}


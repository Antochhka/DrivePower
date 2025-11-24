package httpserver

import "net/http"

// Routes groups handlers.
type Routes struct {
	SessionsMe       http.HandlerFunc
	ActiveSessions   http.HandlerFunc
	SessionStart     http.HandlerFunc
	SessionStop      http.HandlerFunc
	Health           http.HandlerFunc
}

// NewRouter registers endpoints.
func NewRouter(routes Routes) http.Handler {
	mux := http.NewServeMux()
	if routes.SessionsMe != nil {
		mux.Handle("/sessions/me", method(http.MethodGet, routes.SessionsMe))
	}
	if routes.ActiveSessions != nil {
		mux.Handle("/sessions/active", method(http.MethodGet, routes.ActiveSessions))
	}
	if routes.SessionStart != nil {
		mux.Handle("/internal/ocpp/session-start", method(http.MethodPost, routes.SessionStart))
	}
	if routes.SessionStop != nil {
		mux.Handle("/internal/ocpp/session-stop", method(http.MethodPost, routes.SessionStop))
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


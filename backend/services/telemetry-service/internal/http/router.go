package httpserver

import "net/http"

// Routes defines HTTP endpoints.
type Routes struct {
	MeterValues http.Handler
	Health      http.Handler
}

// NewRouter sets up HTTP routing.
func NewRouter(routes Routes) http.Handler {
	mux := http.NewServeMux()
	if routes.MeterValues != nil {
		mux.Handle("/internal/ocpp/meter-values", method(http.MethodPost, routes.MeterValues.ServeHTTP))
	}
	if routes.Health != nil {
		mux.Handle("/health", method(http.MethodGet, routes.Health.ServeHTTP))
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

